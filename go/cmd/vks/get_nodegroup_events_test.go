package vks

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type vksReadTransport func(*http.Request) (*http.Response, error)

func (f vksReadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func vksReadCommand(t *testing.T, source *cobra.Command, flags map[string]string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: source.Use, RunE: source.RunE}
	source.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		switch f.Value.Type() {
		case "string":
			cmd.Flags().String(f.Name, f.DefValue, "")
		case "int":
			n, _ := strconv.Atoi(f.DefValue)
			cmd.Flags().Int(f.Name, n, "")
		case "bool":
			value, _ := strconv.ParseBool(f.DefValue)
			cmd.Flags().Bool(f.Name, value, "")
		default:
			t.Fatalf("unexpected flag type %s", f.Value.Type())
		}
	})
	for _, name := range []string{"endpoint-url", "output", "query", "color", "profile", "region"} {
		if cmd.Flags().Lookup(name) == nil {
			cmd.Flags().String(name, "", "")
		}
	}
	for _, name := range []string{"allow-untrusted-endpoint", "debug", "no-verify-ssl"} {
		if cmd.Flags().Lookup(name) == nil {
			cmd.Flags().Bool(name, false, "")
		}
	}
	for _, name := range []string{"cli-connect-timeout", "cli-read-timeout"} {
		if cmd.Flags().Lookup(name) == nil {
			cmd.Flags().Int(name, 2, "")
		}
	}
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

// Source: https://docs.api.greennode.ai/service-docs/vks-api.html (VKS API).
func assertVKSReadWire(t *testing.T, source *cobra.Command, flags map[string]string, path string, query url.Values, response, outputQuery string) {
	t.Helper()
	for _, mode := range []string{"machine", "user"} {
		for _, status := range []int{202, 400} {
			t.Run(mode+"/"+strconv.Itoa(status), func(t *testing.T) {
				t.Setenv("HOME", t.TempDir())
				for _, key := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
					t.Setenv(key, "")
				}
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
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != "GET" || r.URL.Path != path || !reflect.DeepEqual(r.URL.Query(), query) {
						t.Errorf("request: %s %s", r.Method, r.URL.String())
					}
					if r.Header.Get("Authorization") != "Bearer fixture-access" {
						t.Error("missing bearer token")
					}
					if data, _ := io.ReadAll(r.Body); len(data) != 0 {
						t.Error("unexpected request body")
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if status == 400 {
						fmt.Fprint(w, `{"message":"fixture-error"}`)
					} else {
						fmt.Fprint(w, response)
					}
				}))
				defer server.Close()
				previous := http.DefaultTransport
				prod, _ := login.TokenURLForEnv("prod")
				dev, _ := login.TokenURLForEnv("dev")
				http.DefaultTransport = vksReadTransport(func(r *http.Request) (*http.Response, error) {
					if r.URL.String() != prod && r.URL.String() != dev {
						return nil, fmt.Errorf("unexpected token URL")
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"access_token":"fixture-access","refresh_token":"fixture-rotated","token_type":"Bearer","expires_in":3600}`)), Request: r}, nil
				})
				t.Cleanup(func() { http.DefaultTransport = previous })
				cmd := vksReadCommand(t, source, flags)
				for key, value := range map[string]string{"endpoint-url": server.URL, "allow-untrusted-endpoint": "true", "profile": "fixture", "output": "json", "query": outputQuery, "color": "off"} {
					if err := cmd.Flags().Set(key, value); err != nil {
						t.Fatal(err)
					}
				}
				var err error
				out := testutil.CaptureStdout(t, func() { err = cmd.RunE(cmd, nil) })
				if calls != 1 {
					t.Fatalf("requests = %d; error = %v", calls, err)
				}
				if status == 400 {
					if err == nil {
						t.Fatal("expected API error")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out, "fixture-result") {
					t.Fatalf("query output: %s", out)
				}
			})
		}
	}
}

func TestGetNodegroupEventsWire(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(strconv.FormatBool(explicit), func(t *testing.T) {
			flags := map[string]string{"cluster-id": "fixture-cluster", "nodegroup-id": "fixture-nodegroup"}
			query := url.Values{}
			if explicit {
				flags["page"] = "0"
				flags["page-size"] = "10"
				flags["action"] = "fixture & action"
				flags["type"] = "Normal"
				query = url.Values{"page": {"0"}, "pageSize": {"10"}, "action": {"fixture & action"}, "type": {"Normal"}}
			}
			assertVKSReadWire(t, getNodegroupEventsCmd, flags, "/v1/clusters/fixture-cluster/node-groups/fixture-nodegroup/events", query, `{"items":[{"id":"fixture-result"}],"page":0,"pageSize":10,"total":1}`, "items[0].id")
		})
	}
}

func TestVKSReadValidationBeforeClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, cmd := range []*cobra.Command{getNodegroupEventsCmd, getUpgradeInsightsCmd} {
		for name, bad := range map[string]string{"cluster-id": "../fixture", "page": "-1", "page-size": "0"} {
			t.Run(cmd.Name()+"/"+name, func(t *testing.T) {
				flags := map[string]string{"cluster-id": "fixture-cluster"}
				if cmd == getNodegroupEventsCmd {
					flags["nodegroup-id"] = "fixture-nodegroup"
				}
				flags[name] = bad
				c := vksReadCommand(t, cmd, flags)
				if err := c.RunE(c, nil); err == nil || !strings.Contains(err.Error(), name) {
					t.Fatalf("error = %v", err)
				}
			})
		}
	}
	c := vksReadCommand(t, getNodegroupEventsCmd, map[string]string{"cluster-id": "fixture-cluster", "nodegroup-id": "../fixture"})
	if err := c.RunE(c, nil); err == nil || !strings.Contains(err.Error(), "nodegroup-id") {
		t.Fatalf("nodegroup error = %v", err)
	}
}
