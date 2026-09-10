package vdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type wireFixture struct {
	OperationID  string `json:"operation_id"`
	Method, Path string
	Parameters   []struct {
		Name, In string
		Schema   struct{ Type string }
	}
	RequestExample  json.RawMessage `json:"request_example"`
	ResponseExample json.RawMessage `json:"response_example"`
	Responses       map[string]struct{ Content map[string]json.RawMessage }
}

type localTransport func(*http.Request) (*http.Response, error)

func (f localTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func profileFixture(t *testing.T, mode string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_PORTAL_USER_ID", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
		t.Setenv(name, "")
	}
	t.Setenv("GRN_PROFILE", "fixture")
	t.Setenv("GRN_PORTAL_USER_ID", "12345")
	writer := config.NewConfigFileWriter()
	if err := writer.WriteConfig("fixture", "HCM-3", "json", "fixture-project"); err != nil {
		t.Fatal(err)
	}
	if mode == "user" {
		if err := writer.WriteLoginToken("fixture", "fixture-refresh", time.Now().Add(time.Hour), "user", "dev"); err != nil {
			t.Fatal(err)
		}
	} else if err := writer.WriteCredentials("fixture", "fixture-client", "fixture-client-secret"); err != nil {
		t.Fatal(err)
	}
}

func commandFixture(op operation) *cobra.Command {
	cmd := newOperationCommand(op)
	for _, name := range []string{"profile", "region", "endpoint-url", "output", "query", "color"} {
		cmd.Flags().String(name, "", "")
	}
	for _, name := range []string{"debug", "no-verify-ssl", "allow-untrusted-endpoint"} {
		cmd.Flags().Bool(name, false, "")
	}
	cmd.Flags().Int("cli-read-timeout", 2, "")
	cmd.Flags().Int("cli-connect-timeout", 2, "")
	return cmd
}

func wireArguments(t *testing.T, f wireFixture, op operation, cmd *cobra.Command) (string, url.Values) {
	t.Helper()
	path, query := f.Path, url.Values{}
	for _, p := range f.Parameters {
		if p.In == "header" {
			continue
		}
		flag := ""
		if p.In == "path" {
			flag = flagName(p.Name)
		}
		for _, q := range op.Queries {
			if q.WireName == p.Name {
				flag = q.Flag
				if flag == "" {
					flag = flagName(p.Name)
				}
			}
		}
		if flag == "" {
			t.Fatalf("missing flag for %s %s", p.In, p.Name)
		}
		value := "fixture-001"
		switch p.Schema.Type {
		case "integer", "number":
			value = "1"
		case "boolean":
			value = "true"
		case "object":
			value = `{}`
		}
		if p.Name == "name" {
			value = "fixture & report"
		}
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatal(err)
		}
		if p.In == "path" {
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(value))
		} else {
			query.Set(p.Name, value)
		}
	}
	if len(f.RequestExample) > 0 {
		if err := cmd.Flags().Set("body", string(f.RequestExample)); err != nil {
			t.Fatal(err)
		}
	}
	if cmd.Flags().Lookup("force") != nil {
		if err := cmd.Flags().Set("force", "true"); err != nil {
			t.Fatal(err)
		}
	}
	return path, query
}

func TestPublishedWireContractsWithBothAuthModes(t *testing.T) {
	var fixture struct{ Operations []wireFixture }
	readContractFixture(t, &fixture)
	operations := make(map[string]operation)
	for _, op := range allOperations() {
		operations[op.Method+" "+op.Path] = op
	}
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			profileFixture(t, mode)
			var current wireFixture
			var expectedPath string
			var expectedQuery url.Values
			var errorStatus int
			var tokens, calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokens.Add(1)
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					grant := "client_credentials"
					if mode == "user" {
						grant = "refresh_token"
					}
					if r.Form.Get("grant_type") != grant {
						t.Error("incorrect token grant")
					}
					fmt.Fprint(w, `{"access_token":"fixture-access","expires_in":3600,"token_type":"Bearer","refresh_token":"fixture-rotated"}`)
					return
				}
				calls.Add(1)
				if r.Method != current.Method || r.URL.EscapedPath() != expectedPath || !reflect.DeepEqual(r.URL.Query(), expectedQuery) {
					t.Errorf("%s wire mismatch: %s %s", current.OperationID, r.Method, r.URL)
				}
				portal := ""
				for _, p := range current.Parameters {
					if p.In == "header" && p.Name == "portal-user-id" {
						portal = "12345"
					}
				}
				if r.Header.Get("Authorization") != "Bearer fixture-access" || r.Header.Get("portal-user-id") != portal || r.Header.Get("user-type") != "" {
					t.Error("incorrect request headers")
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if len(current.RequestExample) > 0 {
					var got, want any
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Error(err)
					}
					if err := json.Unmarshal(current.RequestExample, &want); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Error("request body differs from fixture")
					}
				} else if len(raw) > 0 {
					t.Error("unexpected request body")
				}
				if errorStatus != 0 {
					w.WriteHeader(errorStatus)
					fmt.Fprint(w, `{"message":"fixture failure"}`)
					return
				}
				status := 0
				for code := range current.Responses {
					n, _ := strconv.Atoi(code)
					if n >= 200 && n < 300 && (status == 0 || n < status) {
						status = n
					}
				}
				w.WriteHeader(status)
				if string(current.ResponseExample) != "null" {
					_, _ = w.Write(current.ResponseExample)
				}
			}))
			defer srv.Close()
			previous := http.DefaultTransport
			prod, _ := login.TokenURLForEnv("prod")
			dev, _ := login.TokenURLForEnv("dev")
			http.DefaultTransport = localTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != prod && r.URL.String() != dev {
					return nil, fmt.Errorf("unexpected token URL")
				}
				clone := r.Clone(r.Context())
				u, _ := url.Parse(srv.URL + "/token")
				clone.URL, clone.Host = u, u.Host
				return previous.RoundTrip(clone)
			})
			t.Cleanup(func() { http.DefaultTransport = previous })
			for _, f := range fixture.Operations {
				current = f
				op, ok := operations[f.Method+" "+f.Path]
				if !ok {
					t.Fatal("fixture has no command")
				}
				cmd := commandFixture(op)
				expectedPath, expectedQuery = wireArguments(t, f, op, cmd)
				if err := cmd.Flags().Set("endpoint-url", srv.URL); err != nil {
					t.Fatal(err)
				}
				if err := cmd.Flags().Set("allow-untrusted-endpoint", "true"); err != nil {
					t.Fatal(err)
				}
				before := calls.Load()
				testutil.CaptureStdout(t, func() {
					if err := cmd.RunE(cmd, nil); err != nil {
						t.Fatalf("%s: %v", f.OperationID, err)
					}
				})
				if calls.Load() != before+1 {
					t.Fatalf("%s did not send exactly one request", f.OperationID)
				}
			}
			if int(calls.Load()) != len(fixture.Operations) || tokens.Load() != calls.Load() {
				t.Fatal("unexpected request count")
			}
			for _, f := range fixture.Operations {
				if f.Method == "GET" && f.OperationID != "refreshRepoUserUsingGET_1" {
					continue
				}
				for _, code := range []int{401, 429, 503} {
					current, errorStatus = f, code
					op := operations[f.Method+" "+f.Path]
					cmd := commandFixture(op)
					expectedPath, expectedQuery = wireArguments(t, f, op, cmd)
					if err := cmd.Flags().Set("endpoint-url", srv.URL); err != nil {
						t.Fatal(err)
					}
					if err := cmd.Flags().Set("allow-untrusted-endpoint", "true"); err != nil {
						t.Fatal(err)
					}
					beforeCalls, beforeTokens := calls.Load(), tokens.Load()
					if err := cmd.RunE(cmd, nil); err == nil {
						t.Fatalf("%s accepted HTTP %d", f.OperationID, code)
					}
					if calls.Load() != beforeCalls+1 || tokens.Load() != beforeTokens+1 {
						t.Fatalf("%s retried HTTP %d", f.OperationID, code)
					}
				}
			}
			if mode == "user" {
				cfg, err := config.LoadConfig("fixture")
				if err != nil || cfg.RefreshToken != "fixture-rotated" || cfg.AuthMode != "user" || cfg.IamEnv != "dev" {
					t.Fatal("profile rotation lost state")
				}
			}
		})
	}
}
