package vmonitorlog

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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func isolatedProfile(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, key := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
		t.Setenv(key, "")
	}
	t.Setenv("GRN_PROFILE", "fixture")
	w := config.NewConfigFileWriter()
	if err := w.WriteConfig("fixture", "HCM-3", "json", "fixture-project"); err != nil {
		t.Fatal(err)
	}
	if mode == "user" {
		if err := w.WriteLoginToken("fixture", "fixture-refresh-token", time.Now().Add(time.Hour), "user", "dev"); err != nil {
			t.Fatal(err)
		}
	} else if err := w.WriteCredentials("fixture", "fixture-client", "fixture-client-secret"); err != nil {
		t.Fatal(err)
	}
}

func integrationRoot() *cobra.Command {
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	for _, name := range []string{"profile", "region", "endpoint-url", "output", "query", "color"} {
		root.PersistentFlags().String(name, "", "")
	}
	for _, name := range []string{"debug", "no-verify-ssl", "allow-untrusted-endpoint"} {
		root.PersistentFlags().Bool(name, false, "")
	}
	root.PersistentFlags().Int("cli-read-timeout", 2, "")
	root.PersistentFlags().Int("cli-connect-timeout", 2, "")
	root.AddCommand(newVMonitorLogCommand())
	return root
}

// Redirect IAM to a local fixture server.
func redirectIAM(t *testing.T, endpoint string) {
	t.Helper()
	previous := http.DefaultTransport
	prod, _ := login.TokenURLForEnv("prod")
	dev, _ := login.TokenURLForEnv("dev")
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != prod && r.URL.String() != dev {
			return nil, fmt.Errorf("unexpected token URL in test")
		}
		clone := r.Clone(r.Context())
		u, _ := url.Parse(endpoint + "/token")
		clone.URL, clone.Host = u, u.Host
		return previous.RoundTrip(clone)
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func fixtureRequest(t *testing.T, f catalogRecord) (args []string, path string, query url.Values, body string) {
	t.Helper()
	op := findOperationBySignature(t, f.Method+" "+f.Path)
	args = []string{"vmonitor-log", op.Parent, op.Use}
	path, query = f.Path, url.Values{}
	for _, placeholder := range regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(path, -1) {
		name, value := placeholder[1], "fixture-001"
		if name == "cdn-domain" {
			value = "fixture.example.com"
		}
		if name == "bucket-name" {
			value = "fixture archive"
		}
		args = append(args, "--"+flagName(name), value)
		path = strings.ReplaceAll(path, placeholder[0], url.PathEscape(value))
	}
	for _, p := range f.Parameters {
		if p.In != "query" {
			continue
		}
		value := "fixture & filter"
		switch p.Type {
		case "integer", "number":
			value = "1"
		case "boolean":
			value = "true"
		}
		flag := flagName(p.Name)
		if p.Name == "query" {
			flag = "search"
		}
		args = append(args, "--"+flag, value)
		query.Set(p.Name, value)
	}
	if f.RequestBody != nil {
		raw, err := json.Marshal(f.RequestBody.Example)
		if err != nil {
			t.Fatal(err)
		}
		body = string(raw)
		args = append(args, "--body", body)
	}
	if op.Destructive {
		args = append(args, "--force")
	}
	if op.Download {
		args = append(args, "--output-file", filepath.Join(t.TempDir(), "fixture-certificate.zip"))
	}
	return
}

func TestAllOperationsUseSharedAuthAndPublishedWireContracts(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			isolatedProfile(t, mode)
			var current catalogRecord
			var expectedPath, expectedBody string
			var expectedQuery url.Values
			var apiCalls, tokenCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokenCalls.Add(1)
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					grant := "client_credentials"
					if mode == "user" {
						grant = "refresh_token"
					}
					if r.Form.Get("grant_type") != grant {
						t.Error("unexpected auth grant")
					}
					fmt.Fprint(w, `{"access_token":"fixture-access-token","token_type":"Bearer","expires_in":3600,"refresh_token":"fixture-rotated-token"}`)
					return
				}
				apiCalls.Add(1)
				if r.Method != current.Method || r.URL.EscapedPath() != expectedPath || !reflect.DeepEqual(r.URL.Query(), expectedQuery) {
					t.Errorf("%s wire mismatch: %s %s", current.OperationID, r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" {
					t.Error("unexpected auth headers")
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if expectedBody != "" {
					var got, want any
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Error(err)
					}
					if err := json.Unmarshal([]byte(expectedBody), &want); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Error("request body mismatch")
					}
				} else if len(raw) != 0 {
					t.Error("unexpected body")
				}
				response := current.Responses[0]
				if response.Format == "binary" {
					w.Header().Set("Content-Type", response.MediaType)
				}
				w.WriteHeader(response.Status)
				switch response.Type {
				case "object":
					fmt.Fprint(w, `{"id":"fixture-result"}`)
				case "array":
					fmt.Fprint(w, `[{"id":"fixture-result"}]`)
				case "boolean":
					fmt.Fprint(w, "true")
				case "string":
					if response.Format == "binary" {
						fmt.Fprint(w, "fixture-certificate-bytes")
					} else {
						fmt.Fprint(w, `"{\"id\":\"fixture-result\"}"`)
					}
				}
			}))
			defer server.Close()
			redirectIAM(t, server.URL)
			records := readVMonitorLogCatalog(t)
			for _, current = range records {
				args, path, query, body := fixtureRequest(t, current)
				expectedPath, expectedQuery, expectedBody = path, query, body
				args = append(args, "--endpoint-url", server.URL, "--allow-untrusted-endpoint")
				root := integrationRoot()
				root.SetArgs(args)
				before := apiCalls.Load()
				output := testutil.CaptureStdout(t, func() {
					if err := root.Execute(); err != nil {
						t.Fatalf("%s: %v", current.OperationID, err)
					}
				})
				switch current.Responses[0].Type {
				case "empty":
					if output != "" {
						t.Fatal("empty response produced output")
					}
				case "boolean":
					if strings.TrimSpace(output) != "true" {
						t.Fatal("boolean response changed")
					}
				default:
					if current.Responses[0].Format != "binary" && !strings.Contains(output, "fixture-result") {
						t.Fatal("response output changed")
					}
				}
				if apiCalls.Load() != before+1 {
					t.Fatalf("%s did not send exactly one request", current.OperationID)
				}
				if current.Responses[0].Format == "binary" {
					for i, a := range args {
						if a == "--output-file" {
							raw, err := os.ReadFile(args[i+1])
							if err != nil || string(raw) != "fixture-certificate-bytes" {
								t.Fatal("download content mismatch")
							}
							info, err := os.Stat(args[i+1])
							if err != nil || info.Mode().Perm() != 0600 {
								t.Fatal("download permissions mismatch")
							}
						}
					}
				}
			}
			if int(apiCalls.Load()) != len(records) || int(tokenCalls.Load()) != len(records) {
				t.Fatal("unexpected request counts")
			}
			if mode == "user" {
				cfg, err := config.LoadConfig("fixture")
				if err != nil || cfg.RefreshToken != "fixture-rotated-token" || cfg.AuthMode != "user" {
					t.Fatal("token rotation lost profile state")
				}
			}
		})
	}
}

func TestWritesAndDownloadsAreOfflineOrNotRetried(t *testing.T) {
	isolatedProfile(t, "machine")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			fmt.Fprint(w, `{"access_token":"fixture-access-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		calls.Add(1)
		w.WriteHeader(503)
		fmt.Fprint(w, `{"message":"fixture failure"}`)
	}))
	defer server.Close()
	redirectIAM(t, server.URL)
	for _, f := range readVMonitorLogCatalog(t) {
		op := findOperationBySignature(t, f.Method+" "+f.Path)
		if !op.Mutation && !op.Download {
			continue
		}
		args, _, _, _ := fixtureRequest(t, f)
		root := integrationRoot()
		root.SetArgs(append(append([]string{}, args...), "--dry-run", "--profile", "fixture-missing"))
		testutil.CaptureStdout(t, func() {
			if err := root.Execute(); err != nil {
				t.Fatalf("%s dry-run: %v", f.OperationID, err)
			}
		})
		if op.Download {
			for i, a := range args {
				if a == "--output-file" {
					if _, err := os.Stat(args[i+1]); !os.IsNotExist(err) {
						t.Fatal("dry-run wrote output")
					}
				}
			}
			continue
		}
		root = integrationRoot()
		root.SetArgs(append(args, "--endpoint-url", server.URL, "--allow-untrusted-endpoint"))
		before := calls.Load()
		if err := root.Execute(); err == nil {
			t.Fatal("accepted failed mutation")
		}
		if calls.Load() != before+1 {
			t.Fatal("mutation was retried")
		}
	}
}

func TestCredentialResponsesNeverLeakThroughDebugOrErrors(t *testing.T) {
	isolatedProfile(t, "machine")
	var status int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			fmt.Fprint(w, `{"access_token":"fixture-access-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		w.WriteHeader(status)
		fmt.Fprint(w, `{"storageSettings":{"accessKey":"fixture-private-key"}}`)
	}))
	defer server.Close()
	redirectIAM(t, server.URL)
	for _, status = range []int{200, 403} {
		for _, show := range []bool{false, true} {
			args := []string{"vmonitor-log", "archive", "get", "--archive-id", "fixture-archive", "--endpoint-url", server.URL, "--allow-untrusted-endpoint", "--debug"}
			if show {
				args = append(args, "--show-secret")
			}
			root := integrationRoot()
			root.SetArgs(args)
			var out string
			var runErr error
			debug := testutil.CaptureStderr(t, func() { out = testutil.CaptureStdout(t, func() { runErr = root.Execute() }) })
			if strings.Contains(debug, "fixture-private-key") || strings.Contains(debug, "fixture-access-token") {
				t.Fatal("debug exposed credentials")
			}
			if status == 403 {
				if runErr == nil || strings.Contains(runErr.Error(), "fixture-private-key") {
					t.Fatal("unsafe credential error")
				}
			} else {
				if runErr != nil {
					t.Fatal(runErr)
				}
				if strings.Contains(out, "fixture-private-key") != show {
					t.Fatal("credential output policy mismatch")
				}
			}
		}
	}
}
