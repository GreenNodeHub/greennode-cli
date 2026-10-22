package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/iamclient"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func fixtureForRequest(t *testing.T, method, path string) iamCatalogRecord {
	t.Helper()
	for _, record := range readIAMFixtures(t) {
		if record.Method != method {
			continue
		}
		pattern := regexp.QuoteMeta(record.Path)
		pattern = regexp.MustCompile(`\\\{[^{}]+\\\}`).ReplaceAllString(pattern, "[^/]+")
		if regexp.MustCompile("^" + pattern + "$").MatchString(path) {
			return record
		}
	}
	t.Fatalf("no public fixture for %s %s", method, path)
	return iamCatalogRecord{}
}

func TestIAMPublicSurfaceCounts(t *testing.T) {
	records := readIAMFixtures(t)
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, record := range records {
		key := record.Method + " " + record.Path
		if seen[key] {
			t.Fatalf("duplicate fixture: %s", key)
		}
		seen[key] = true
		counts[record.ServiceID]++
	}
	if counts["iam_accounts"] != 72 || counts["iam_policies"] != 34 {
		t.Fatalf("fixture counts = %v", counts)
	}
	reads := map[string]bool{}
	collectIAMReadRoutes(IamCmd, nil, reads)
	var leaves int
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if len(cmd.Commands()) == 0 && cmd.RunE != nil {
			leaves++
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(IamCmd)
	if len(reads) != 34 || len(allIAMMutationOperations()) != 61 || leaves != 103 {
		t.Fatalf("reads/mutations/leaves = %d/%d/%d", len(reads), len(allIAMMutationOperations()), leaves)
	}
}

func assertPublishedQuery(t *testing.T, record iamCatalogRecord, query url.Values) {
	t.Helper()
	allowed := map[string]bool{}
	if record.Method == http.MethodGet && record.Path == "/v1/service-accounts" {
		allowed["pageNumber"], allowed["pageSize"] = true, true
	}
	for _, p := range record.Parameters {
		if p.In != "query" {
			continue
		}
		allowed[p.Name] = true
		if p.Required && query.Get(p.Name) == "" {
			t.Errorf("missing required query %s", p.Name)
		}
	}
	for name := range query {
		if !allowed[name] {
			t.Errorf("undocumented query %s", name)
		}
	}
}

func TestEveryMutationUsesPublishedWireContract(t *testing.T) {
	records := readIAMMutationCatalog(t)
	for _, op := range allIAMMutationOperations() {
		t.Run(op.Key, func(t *testing.T) {
			record := records[op.Method+" "+op.Path]
			flags := validIAMMutationFlags(t, op)
			flags["force"] = "true"
			path := record.Path
			for _, parameter := range op.Paths {
				path = strings.ReplaceAll(path, "{"+parameter.Placeholder+"}", url.PathEscape(flags[parameter.Flag]))
			}
			var expectedBody any
			if record.RequestBody != nil {
				if record.Path == "/v1/compose-policy" {
					expectedBody = []any{}
				} else {
					body := map[string]any{}
					for _, field := range record.RequestBody.Content["application/json"].Schema.Required {
						body[field] = "value"
					}
					expectedBody = body
				}
			}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/v1/auth/userinfo" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"userId":"fixture-actor","userType":"iam-user"}`)
					return
				}
				calls.Add(1)
				if r.Method != record.Method || r.URL.EscapedPath() != path || r.URL.RawQuery != "" {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL, record.Method, path)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-token" || r.Header.Get("portal-user-id") != "" {
					t.Error("incorrect auth headers")
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				var got any
				if len(raw) > 0 {
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Error(err)
					}
				}
				if !reflect.DeepEqual(got, expectedBody) {
					t.Errorf("body = %#v, want %#v", got, expectedBody)
				}
				for status, response := range record.Responses {
					code, err := strconv.Atoi(status)
					if err != nil {
						t.Error(err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(code)
					if len(response.Content) > 0 {
						_, _ = io.WriteString(w, `{"id":"fixture-result"}`)
					}
					break
				}
			}))
			defer server.Close()
			previous := accountsClientFactory
			accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) { return fixtureClient(server.URL, false), nil }
			defer func() { accountsClientFactory = previous }()
			op.BuildClient = accountsClientFactory
			command := newIAMMutationCommand(op)
			setIAMMutationFlags(t, command, flags)
			testutil.CaptureStdout(t, func() {
				if err := command.RunE(command, nil); err != nil {
					t.Fatal(err)
				}
			})
			if calls.Load() != 1 {
				t.Fatalf("mutation calls = %d, want 1", calls.Load())
			}
		})
	}
}

func fixtureClient(endpoint string, debug bool) *client.GreennodeClient {
	tokens := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "http://127.0.0.1:1/fixture-token")
	tokens.SetToken("fixture-token", time.Now().Add(time.Hour))
	return client.NewGreennodeClient(endpoint, tokens, time.Second, time.Second, true, debug)
}

func TestFactoriesUseBothAuthModes(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		for _, service := range []string{"accounts", "policies"} {
			for _, mutation := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/mutation=%t", mode, service, mutation), func(t *testing.T) {
					path, method, status := "/v1/auth/userinfo", "GET", 200
					build := iamclient.BuildAccountsClient
					key := "service-account-create"
					if service == "policies" {
						path = "/v1/groups"
						build = iamclient.BuildPoliciesClient
						key = "group-create"
					}
					if mutation {
						method = "POST"
						status = 201
						if service == "accounts" {
							path = "/v1/service-accounts"
						}
					}
					var calls atomic.Int32
					server := testutil.ProfileServer(t, mode, "", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != method || r.URL.Path != path {
							t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						if !mutation || service == "policies" {
							_, _ = io.WriteString(w, `{"id":"fixture-result"}`)
						}
					})
					var command *cobra.Command
					args := []string{}
					if mutation {
						op := findIAMMutation(t, key)
						op.BuildClient = func(cmd *cobra.Command) (*client.GreennodeClient, error) {
							if err := cmd.Flags().Set("endpoint-url", server.URL); err != nil {
								return nil, err
							}
							defer cmd.Flags().Set("endpoint-url", "")
							return build(cmd)
						}
						command = newIAMMutationCommand(op)
						args = []string{op.Use, "--body", `{"name":"fixture-resource"}`, "--force", "--allow-untrusted-endpoint"}
					} else {
						command = newReadCommand("read", "Fixture read", build, staticPath(path), noQuery)
						args = []string{"read", "--endpoint-url", server.URL, "--allow-untrusted-endpoint"}
					}
					root := testutil.ProfileRoot(command)
					root.SetArgs(args)
					testutil.CaptureStdout(t, func() {
						if err := root.Execute(); err != nil {
							t.Fatal(err)
						}
					})
					if calls.Load() != 1 {
						t.Fatalf("API calls = %d, want 1", calls.Load())
					}
				})
			}
		}
	}
}

func TestMutationsNeverRetry(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(status) }))
			defer server.Close()
			op := findIAMMutation(t, "group-create")
			op.BuildClient = func(*cobra.Command) (*client.GreennodeClient, error) { return fixtureClient(server.URL, false), nil }
			command := newIAMMutationCommand(op)
			setIAMMutationFlags(t, command, map[string]string{"body": `{"name":"fixture-group"}`, "force": "true"})
			if err := command.RunE(command, nil); err == nil {
				t.Fatal("expected failure")
			}
			if calls.Load() != 1 {
				t.Fatalf("calls = %d, want 1", calls.Load())
			}
		})
	}
}

func TestSecretFilesRejectLinksDirectoriesAndPublicPermissions(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "fixture-private.json")
	public := filepath.Join(dir, "fixture-public.json")
	link := filepath.Join(dir, "fixture-link.json")
	for path, mode := range map[string]os.FileMode{private: 0600, public: 0644} {
		if err := os.WriteFile(path, []byte(`{"password":"fixture-secret"}`), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{public, link, dir} {
		command := newIAMMutationCommand(findIAMMutation(t, "iam-user-create"))
		setIAMMutationFlags(t, command, map[string]string{"body-file": path, "dry-run": "true"})
		if err := command.RunE(command, nil); err == nil {
			t.Errorf("accepted unsafe body file %s", filepath.Base(path))
		}
	}
}

func TestNonInteractiveMutationReturnsError(t *testing.T) {
	cli.SetNonInteractive(true)
	defer cli.SetNonInteractive(false)
	op := findIAMMutation(t, "group-create")
	op.BuildClient = func(*cobra.Command) (*client.GreennodeClient, error) {
		t.Fatal("client created after refusal")
		return nil, nil
	}
	command := newIAMMutationCommand(op)
	setIAMMutationFlags(t, command, map[string]string{"body": `{"name":"fixture-group"}`})
	if err := command.RunE(command, nil); err == nil {
		t.Fatal("expected confirmation refusal")
	}
}

func TestOpaqueMFAResponsesRequireExplicitOutputOptIn(t *testing.T) {
	for _, payload := range []string{`{"data":"fixture-secret"}`, `{"qrCode":"fixture-secret"}`, `"fixture-secret"`, "fixture-secret"} {
		for _, show := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/show=%t", payload, show), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, payload) }))
				defer server.Close()
				op := findIAMMutation(t, "iam-user-current-set-up-google")
				op.BuildClient = func(*cobra.Command) (*client.GreennodeClient, error) { return fixtureClient(server.URL, true), nil }
				command := newIAMMutationCommand(op)
				setIAMMutationFlags(t, command, map[string]string{"force": "true", "show-secret": strconv.FormatBool(show)})
				var output string
				logs := testutil.CaptureStderr(t, func() {
					output = testutil.CaptureStdout(t, func() {
						if err := command.RunE(command, nil); err != nil {
							t.Fatal(err)
						}
					})
				})
				if strings.Contains(logs, "fixture-secret") {
					t.Fatal("debug disclosed secret")
				}
				if strings.Contains(output, "fixture-secret") != show {
					t.Fatalf("unexpected output disclosure: %q", output)
				}
				if !show && !strings.Contains(output, client.RedactedValue) {
					t.Fatal("missing redaction marker")
				}
			})
		}
	}
}

func TestSecretErrorsStayRedactedWithOutputOptIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"data":"fixture-secret"}`)
	}))
	defer server.Close()
	op := findIAMMutation(t, "iam-user-current-set-up-google")
	op.BuildClient = func(*cobra.Command) (*client.GreennodeClient, error) { return fixtureClient(server.URL, true), nil }
	command := newIAMMutationCommand(op)
	setIAMMutationFlags(t, command, map[string]string{"force": "true", "show-secret": "true"})
	var runErr error
	logs := testutil.CaptureStderr(t, func() { runErr = command.RunE(command, nil) })
	if runErr == nil {
		t.Fatal("expected API failure")
	}
	if strings.Contains(logs+runErr.Error(), "fixture-secret") {
		t.Fatal("error/debug disclosed secret")
	}
}
