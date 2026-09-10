package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

func foundationCommand() *cobra.Command {
	cmd := &cobra.Command{}
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

func foundationHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"GRN_PROFILE", "GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
		t.Setenv(name, "")
	}
}

func TestFoundationBuildClientBothAuthModes(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		t.Run(mode, func(t *testing.T) {
			foundationHome(t)
			t.Setenv("GRN_PROFILE", "selected")
			writer := config.NewConfigFileWriter()
			if err := writer.WriteConfig("selected", "HCM-3", "json", "project"); err != nil {
				t.Fatal(err)
			}
			if mode == "machine" {
				if err := writer.WriteCredentials("selected", "client", "secret"); err != nil {
					t.Fatal(err)
				}
			} else if err := writer.WriteLoginToken("selected", "old-refresh", time.Now().Add(time.Hour), "user", "dev"); err != nil {
				t.Fatal(err)
			}
			if err := writer.WritePortalUserID("selected", "42"); err != nil {
				t.Fatal(err)
			}
			tokens, requests := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokens++
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					grant := "client_credentials"
					if mode == "user" {
						grant = "refresh_token"
					}
					if r.Form.Get("grant_type") != grant {
						t.Errorf("grant = %s", r.Form.Get("grant_type"))
					}
					fmt.Fprint(w, `{"access_token":"access","token_type":"Bearer","expires_in":3600,"refresh_token":"rotated"}`)
					return
				}
				requests++
				if r.Header.Get("Authorization") != "Bearer access" || r.Header.Get("X-Test-Profile") != "selected" {
					t.Error("builder lost token or profile hook")
				}
				fmt.Fprint(w, `{"ok":true}`)
			}))
			defer srv.Close()
			previous := tokenURLForEnv
			tokenURLForEnv = func(env string) (string, error) {
				if mode == "user" && env != "dev" {
					t.Errorf("IAM env = %q", env)
				}
				return srv.URL + "/token", nil
			}
			t.Cleanup(func() { tokenURLForEnv = previous })
			cmd := foundationCommand()
			cmd.Flags().Set("endpoint-url", srv.URL)
			cmd.Flags().Set("allow-untrusted-endpoint", "true")
			api, cfg, err := BuildClient(cmd, ClientOptions{
				ResolveEndpoint: func(*config.Config) (string, error) { t.Fatal("override ignored"); return "", nil },
				AfterBuild: func(_ *cobra.Command, cfg *config.Config, c *client.GreennodeClient) error {
					c.SetHeader("X-Test-Profile", cfg.Profile)
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if tokens != 0 || requests != 0 {
				t.Fatal("builder performed network I/O")
			}
			if cfg.Profile != "selected" || cfg.PortalUserID != "42" {
				t.Fatalf("profile not preserved: %q", cfg.Profile)
			}
			if _, err := api.Get("/resource", nil); err != nil {
				t.Fatal(err)
			}
			if tokens != 1 || requests != 1 {
				t.Fatalf("token/API calls = %d/%d", tokens, requests)
			}
			loaded, err := config.LoadConfig("selected")
			if err != nil {
				t.Fatal(err)
			}
			if mode == "user" && loaded.RefreshToken != "rotated" {
				t.Fatal("rotation not persisted to resolved profile")
			}
			if loaded.PortalUserID != "42" {
				t.Fatal("rotation erased config")
			}
		})
	}
}

func TestFoundationBuilderRegionAndCancellation(t *testing.T) {
	foundationHome(t)
	t.Setenv("GRN_CLIENT_ID", "client")
	t.Setenv("GRN_CLIENT_SECRET", "secret")
	t.Setenv("GRN_DEFAULT_REGION", "HAN")
	cmd := foundationCommand()
	cmd.Flags().Set("region", "HCM-3")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	api, cfg, err := BuildClient(cmd, ClientOptions{
		ResolveEndpoint: func(cfg *config.Config) (string, error) {
			if cfg.Region != "HCM-3" {
				t.Fatalf("region = %s", cfg.Region)
			}
			return cfg.GetEndpoint("vbackup")
		},
		ApplyRegion: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Region != "HCM-3" {
		t.Fatal("region override lost")
	}
	if _, err := api.Get("/resource", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
	_, cfg, err = BuildClient(cmd, ClientOptions{ResolveEndpoint: func(*config.Config) (string, error) { return "https://api.vngcloud.vn", nil }})
	if err != nil || cfg.Region != "HAN" {
		t.Fatalf("global service applied regional override: %v", err)
	}
	if _, err := NewClientWithEndpoint(cmd, ""); err == nil {
		t.Fatal("empty global endpoint accepted")
	}
	if _, _, err := BuildClient(cmd, ClientOptions{}); err == nil {
		t.Fatal("missing resolver accepted")
	}
	if _, err := NewClient(&cobra.Command{}, "vks"); err == nil {
		t.Fatal("missing flags accepted")
	}
	cmd.Flags().Set("cli-read-timeout", "-1")
	if _, err := NewClient(cmd, "vks"); err == nil {
		t.Fatal("negative timeout accepted")
	}
}

func TestFoundationEndpointRejectsMalformedURLs(t *testing.T) {
	for _, endpoint := range []string{"%", "localhost", "https:///missing-host", "ftp://example.com", "https://user:secret@api.vngcloud.vn", "https://api.vngcloud.vn/#secret"} {
		if err := CheckEndpoint(endpoint, false, true); err == nil {
			t.Errorf("accepted %q", endpoint)
		}
	}
}

func TestFoundationConfirmationDoesNotReadStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if _, err := w.WriteString("yes\n"); err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = previous; SetNonInteractive(false) })
	SetNonInteractive(true)
	if Confirm(false, "delete?") || ConfirmationError() == nil {
		t.Fatal("confirmation did not refuse")
	}
	buf := make([]byte, 4)
	if _, err := r.Read(buf); err != nil || string(buf) != "yes\n" {
		t.Fatal("confirmation consumed stdin")
	}
	SetNonInteractive(true)
	if !Confirm(true, "delete?") || ConfirmationError() != nil {
		t.Fatal("forced confirmation failed")
	}
}

func TestFoundationOutputRejectsInvalidConfiguredFormat(t *testing.T) {
	foundationHome(t)
	if err := config.NewConfigFileWriter().WriteConfig("default", "HCM-3", "typo", ""); err != nil {
		t.Fatal(err)
	}
	cmd := foundationCommand()
	if err := Output(cmd, map[string]any{"ok": true}); err == nil {
		t.Fatal("invalid configured output silently accepted")
	}
	cmd.Flags().Set("output", "json")
	if err := Output(cmd, map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
}
