package vbackup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/cli"
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
	root.AddCommand(newVBackupCommand())
	return root
}

func fixtureRequest(f operationFixture) (args []string, path string, query url.Values, body string) {
	args = append([]string{"vbackup"}, strings.Split(f.Command, " ")...)
	path, query = f.Path, url.Values{}
	for _, p := range f.Parameters {
		value := "fixture-001"
		if p.Type == "integer" {
			value = "25"
			if p.Name == "page" {
				value = "1"
			}
		}
		if p.Name == "name" || p.Name == "backend" {
			value = "fixture & report"
		}
		args = append(args, "--"+p.Flag, value)
		if p.In == "path" {
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(value))
		} else {
			query.Set(p.Name, value)
		}
	}
	if f.Request != nil {
		bodies := map[string]string{
			"policy create":           `{"backendId":"fixture-backend","projectId":"fixture-project","name":"fixture-policy","config":{"dailyEnabled":true}}`,
			"policy update":           `{"name":"fixture-updated-policy"}`,
			"server create":           `{"backendId":"fixture-backend","projectId":"fixture-project","backupEnabled":true}`,
			"server update-policy":    `{"id":"fixture-policy"}`,
			"server update-volume":    `{"volumeId":"fixture-volume","backupEnabled":true}`,
			"vserver create-instance": `{"projectId":"fixture-project","serverIds":["fixture-server"]}`,
			"volume usage":            `{"backendId":"fixture-backend","projectId":"fixture-project","volumeIds":["fixture-volume"]}`,
		}
		body = bodies[f.Command]
		args = append(args, "--body", body)
	}
	if f.Destructive {
		args = append(args, "--force")
	}
	return
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

func TestAllOperationsUseProfileAuthAndPublishedWireContracts(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			isolatedProfile(t, mode)
			var current operationFixture
			var expectedPath, expectedBody string
			var expectedQuery url.Values
			var tokenCalls, apiCalls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokenCalls.Add(1)
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					want := "client_credentials"
					if mode == "user" {
						want = "refresh_token"
					}
					if r.Form.Get("grant_type") != want {
						t.Errorf("wrong grant: %s", r.Form.Get("grant_type"))
					}
					fmt.Fprint(w, `{"access_token":"fixture-access-token","token_type":"Bearer","expires_in":3600,"refresh_token":"fixture-rotated-token"}`)
					return
				}
				apiCalls.Add(1)
				if r.Method != current.Method || r.URL.EscapedPath() != expectedPath || !reflect.DeepEqual(r.URL.Query(), expectedQuery) {
					t.Errorf("%s wire request: %s %s", current.Command, r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("unexpected request headers")
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				var got, want any
				if expectedBody != "" {
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Error(err)
					}
					if err := json.Unmarshal([]byte(expectedBody), &want); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Error("request body differs from fixture")
					}
				} else if len(raw) != 0 {
					t.Error("unexpected request body")
				}
				w.WriteHeader(current.Response.Status)
				switch current.Response.Type {
				case "object":
					fmt.Fprint(w, `{"id":"fixture-result"}`)
				case "array":
					fmt.Fprint(w, `[{"id":"fixture-result"}]`)
				}
			}))
			defer srv.Close()
			redirectIAM(t, srv.URL)
			for _, f := range readContractFixtures(t) {
				current = f
				args, path, query, body := fixtureRequest(f)
				expectedPath, expectedQuery, expectedBody = path, query, body
				args = append(args, "--endpoint-url", srv.URL, "--allow-untrusted-endpoint")
				root := integrationRoot()
				root.SetArgs(args)
				before := apiCalls.Load()
				out := testutil.CaptureStdout(t, func() {
					if err := root.Execute(); err != nil {
						t.Fatalf("%s: %v", f.Command, err)
					}
				})
				if apiCalls.Load() != before+1 {
					t.Fatalf("%s did not send exactly one request", f.Command)
				}
				if f.Response.Type == "empty" && out != "" {
					t.Fatalf("%s printed an empty response", f.Command)
				}
				if f.Response.Type != "empty" && !strings.Contains(out, "fixture-result") {
					t.Fatalf("%s lost response output", f.Command)
				}
			}
			if tokenCalls.Load() != 29 || apiCalls.Load() != 29 {
				t.Fatal("unexpected request count")
			}
			if mode == "user" {
				cfg, err := config.LoadConfig("fixture")
				if err != nil || cfg.RefreshToken != "fixture-rotated-token" || cfg.AuthMode != "user" || cfg.IamEnv != "dev" {
					t.Fatal("login rotation lost profile state")
				}
			}
		})
	}
}

func TestMutationsAreNotRetried(t *testing.T) {
	isolatedProfile(t, "machine")
	var status int
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			fmt.Fprint(w, `{"access_token":"fixture-access-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		calls.Add(1)
		w.WriteHeader(status)
		fmt.Fprint(w, `{"message":"fixture failure"}`)
	}))
	defer srv.Close()
	redirectIAM(t, srv.URL)
	for _, f := range readContractFixtures(t) {
		if !f.Mutation {
			continue
		}
		for _, status = range []int{401, 429, 503} {
			args, _, _, _ := fixtureRequest(f)
			args = append(args, "--endpoint-url", srv.URL, "--allow-untrusted-endpoint")
			root := integrationRoot()
			root.SetArgs(args)
			before := calls.Load()
			if err := root.Execute(); err == nil {
				t.Fatalf("%s accepted HTTP %d", f.Command, status)
			}
			if calls.Load() != before+1 {
				t.Fatalf("%s retried HTTP %d", f.Command, status)
			}
		}
	}
}

func TestConfiguredRegionAndOfflineValidation(t *testing.T) {
	isolatedProfile(t, "machine")
	w := config.NewConfigFileWriter()
	if err := w.WriteConfig("fixture", "HAN", "json", ""); err != nil {
		t.Fatal(err)
	}
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) { t.Fatal("unexpected client construction"); return nil, nil }
	t.Cleanup(func() { newClient = previous })
	root := integrationRoot()
	root.SetArgs([]string{"vbackup", "backend", "list"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "HCM-3") {
		t.Fatalf("configured region: %v", err)
	}
	for _, body := range []string{"", "[]", "null", "{", "{} {}"} {
		root = integrationRoot()
		root.SetArgs([]string{"vbackup", "policy", "create", "--body", body, "--dry-run"})
		if err := root.Execute(); err == nil {
			t.Errorf("accepted malformed body %q", body)
		}
	}
	for _, page := range []string{"0", "-1", "1.5", "2147483648"} {
		root = integrationRoot()
		root.SetArgs([]string{"vbackup", "backend", "list", "--page", page})
		if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "page") {
			t.Errorf("page validation: %v", err)
		}
	}
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	for _, f := range readContractFixtures(t) {
		if !f.Mutation {
			continue
		}
		args, _, _, _ := fixtureRequest(f)
		args = append(args, "--dry-run", "--profile", "missing")
		root = integrationRoot()
		root.SetArgs(args)
		testutil.CaptureStdout(t, func() {
			if err := root.Execute(); err != nil {
				t.Fatalf("%s offline: %v", f.Command, err)
			}
		})
	}
}
