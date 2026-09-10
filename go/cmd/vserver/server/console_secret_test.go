package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type consoleTransport func(*http.Request) (*http.Response, error)

func (f consoleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConsoleURLDisclosure(t *testing.T) {
	raw, err := os.ReadFile("../testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Operations []struct {
			OperationID  string `json:"operation_id"`
			Method, Path string
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	path := ""
	for _, op := range fixture.Operations {
		if op.OperationID == "getConsoleUrlUsingGET" && op.Method == "GET" {
			path = strings.NewReplacer("{projectId}", "fixture-project", "{serverId}", "fixture-server").Replace(op.Path)
		}
	}
	if path == "" || getConsoleURLCmd.Flags().Lookup("show-secret") == nil {
		t.Fatal("missing console contract or disclosure flag")
	}
	for _, mode := range []string{"machine", "user"} {
		for _, format := range []string{"json", "table", "text"} {
			for _, show := range []bool{false, true} {
				for _, status := range []int{200, 401, 403, 503} {
					for index, body := range []string{`"https://fixture.example/fixture-secret"`, `{"data":"https://fixture.example/fixture-secret"}`, `{"data":{"url":"https://fixture.example/fixture-secret"}}`} {
						t.Run(fmt.Sprintf("%s/%s/%t/%d/%d", mode, format, show, status, index), func(t *testing.T) {
							t.Setenv("HOME", t.TempDir())
							for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
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
							var calls, tokens atomic.Int32
							srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								w.Header().Set("Content-Type", "application/json")
								if r.URL.Path == "/fixture-token" {
									tokens.Add(1)
									fmt.Fprint(w, `{"access_token":"fixture-access","refresh_token":"fixture-rotated","expires_in":3600,"token_type":"Bearer"}`)
									return
								}
								calls.Add(1)
								if r.Method != "GET" || r.URL.Path != path || r.Header.Get("portal-user-id") != "12345" || r.Header.Get("Authorization") != "Bearer fixture-access" {
									t.Error("incorrect console request")
								}
								w.WriteHeader(status)
								fmt.Fprint(w, body)
							}))
							defer srv.Close()
							previous := http.DefaultTransport
							prod, _ := login.TokenURLForEnv("prod")
							dev, _ := login.TokenURLForEnv("dev")
							http.DefaultTransport = consoleTransport(func(r *http.Request) (*http.Response, error) {
								if r.URL.String() != prod && r.URL.String() != dev {
									return nil, fmt.Errorf("unexpected fixture token URL")
								}
								clone := r.Clone(r.Context())
								target, _ := url.Parse(srv.URL + "/fixture-token")
								clone.URL, clone.Host = target, target.Host
								return previous.RoundTrip(clone)
							})
							t.Cleanup(func() { http.DefaultTransport = previous })
							cmd := &cobra.Command{Use: "get-console-url", RunE: runGetConsoleURL}
							cmd.Flags().String("server-id", "fixture-server", "")
							cmd.Flags().Bool("show-secret", show, "")
							root := testutil.ProfileRoot(cmd)
							root.SetArgs([]string{"get-console-url", "--endpoint-url", srv.URL, "--allow-untrusted-endpoint", "--debug", "--output", format})
							var out string
							debug := testutil.CaptureStderr(t, func() {
								out = testutil.CaptureStdout(t, func() {
									err := root.Execute()
									if status == 200 {
										if err != nil {
											t.Error(err)
										}
									} else if err == nil || strings.Contains(err.Error(), "fixture-secret") {
										t.Error("unsafe console error")
									}
								})
							})
							if strings.Contains(out, "fixture-secret") != (status == 200 && show) {
								t.Error("incorrect console disclosure")
							}
							if strings.Contains(debug, "fixture-secret") || strings.Contains(debug, "fixture-access") {
								t.Error("unsafe console debug output")
							}
							if calls.Load() != 1 || tokens.Load() != 1 {
								t.Error("console request replayed")
							}
						})
					}
				}
			}
		}
	}
}
