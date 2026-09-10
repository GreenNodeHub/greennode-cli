package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/login"
	"github.com/spf13/cobra"
)

type profileTransport func(*http.Request) (*http.Response, error)

func (f profileTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// ProfileServer isolates profiles and redirects IAM to local fixtures.
func ProfileServer(t *testing.T, mode, region string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	if mode != "machine" && mode != "user" {
		t.Fatalf("invalid fixture auth mode %q", mode)
	}
	t.Setenv("HOME", t.TempDir())
	for _, key := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
		t.Setenv(key, "")
	}
	t.Setenv("GRN_PROFILE", "fixture")
	writer := config.NewConfigFileWriter()
	if err := writer.WriteConfig("fixture", region, "json", "fixture-project"); err != nil {
		t.Fatal(err)
	}
	if mode == "user" {
		if err := writer.WriteLoginToken("fixture", "fixture-refresh-token", time.Now().Add(time.Hour), "user", "dev"); err != nil {
			t.Fatal(err)
		}
	} else if err := writer.WriteCredentials("fixture", "fixture-client", "fixture-client-secret"); err != nil {
		t.Fatal(err)
	}
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fixture-token" {
			tokenCalls.Add(1)
			if r.Method != http.MethodPost {
				t.Error("incorrect IAM method")
			}
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if mode == "user" {
				if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "fixture-refresh-token" {
					t.Error("incorrect user grant")
				}
			} else {
				id, secret, ok := r.BasicAuth()
				if r.Form.Get("grant_type") != "client_credentials" || !ok || id != "fixture-client" || secret != "fixture-client-secret" {
					t.Error("incorrect machine grant")
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"access_token":"fixture-access-token","refresh_token":"fixture-rotated-token","expires_in":3600,"refresh_expires_in":7200,"token_type":"Bearer"}`)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fixture-access-token" || r.Header.Get("portal-user-id") != "" {
			t.Error("incorrect API authentication headers")
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	previous := http.DefaultTransport
	prod, _ := login.TokenURLForEnv("prod")
	dev, _ := login.TokenURLForEnv("dev")
	http.DefaultTransport = profileTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != prod && r.URL.String() != dev {
			return nil, fmt.Errorf("unexpected fixture token URL")
		}
		clone := r.Clone(r.Context())
		target, _ := url.Parse(server.URL + "/fixture-token")
		clone.URL, clone.Host = target, target.Host
		return previous.RoundTrip(clone)
	})
	t.Cleanup(func() {
		http.DefaultTransport = previous
		if tokenCalls.Load() != 1 {
			t.Errorf("token calls = %d, want 1", tokenCalls.Load())
		}
		if mode == "user" {
			cfg, err := config.LoadConfig("fixture")
			if err != nil {
				t.Error(err)
			} else if cfg.RefreshToken != "fixture-rotated-token" {
				t.Error("refresh token rotation was not persisted")
			}
		}
	})
	return server
}

// ProfileRoot supplies shared flags to a service command.
func ProfileRoot(command *cobra.Command) *cobra.Command {
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	for _, name := range []string{"profile", "region", "endpoint-url", "output", "query", "color"} {
		root.PersistentFlags().String(name, "", "")
	}
	for _, name := range []string{"debug", "no-verify-ssl", "allow-untrusted-endpoint"} {
		root.PersistentFlags().Bool(name, false, "")
	}
	root.PersistentFlags().Int("cli-read-timeout", 2, "")
	root.PersistentFlags().Int("cli-connect-timeout", 2, "")
	root.AddCommand(command)
	return root
}
