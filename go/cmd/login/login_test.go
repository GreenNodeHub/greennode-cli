package login

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// resolveConfig is the pure cobra-slice seam; these tests cover flag>env
// precedence, the iam-env presets and piecewise overrides, scope splitting,
// the public-client (empty-secret) path, and the missing-client-id failure.
//
// They are deliberately non-parallel: resolveConfig reads process env
// (GRN_LOGIN_*) via os.Getenv, and t.Setenv is not safe across parallel tests
// in the same process.
func TestResolveConfig(t *testing.T) {
	// Start from a clean env so each case controls its inputs exactly.
	t.Setenv("GRN_LOGIN_CLIENT_ID", "")
	t.Setenv("GRN_LOGIN_CLIENT_SECRET", "")

	cases := []struct {
		name          string
		clientID      string
		clientSecret  string
		iamEnv        string
		authorizeURL  string
		tokenURL      string
		scope         string
		timeout       time.Duration
		envClientID   string
		envClientSec  string
		wantErr       string // non-empty → expect an error containing this
		wantClientID  string
		wantClientSec string
		wantAuthorize string
		wantToken     string
		wantScopes    []string
		wantTimeout   time.Duration
	}{
		{
			name: "flag client-id wins over env",
			clientID: "cid-flag", envClientID: "cid-env", iamEnv: "prod",
			wantClientID: "cid-flag", wantClientSec: "",
			wantAuthorize: iamEndpoints["prod"].Authorize, wantToken: iamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "env client-id used when flag empty",
			envClientID: "cid-env", iamEnv: "prod",
			wantClientID: "cid-env",
			wantAuthorize: iamEndpoints["prod"].Authorize, wantToken: iamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "missing client id errors", iamEnv: "prod",
			wantErr: "client-id",
		},
		{
			name: "client-secret flag wins over env",
			clientID: "cid", clientSecret: "cs-flag", envClientSec: "cs-env", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "cs-flag",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "client-secret env fallback",
			clientID: "cid", envClientSec: "cs-env", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "cs-env",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "empty secret → public client path",
			clientID: "cid", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "dev preset swaps token URL",
			clientID: "cid", iamEnv: "dev",
			wantClientID: "cid",
			wantAuthorize: iamEndpoints["dev"].Authorize, wantToken: iamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "invalid iam-env errors",
			clientID: "cid", iamEnv: "staging",
			wantErr: "iam-env",
		},
		{
			name: "authorize-url overrides preset only",
			clientID: "cid", iamEnv: "prod", authorizeURL: "https://custom/auth",
			wantClientID: "cid", wantClientSec: "",
			wantAuthorize: "https://custom/auth", wantToken: iamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "token-url overrides preset only",
			clientID: "cid", iamEnv: "prod", tokenURL: "https://custom/token",
			wantClientID: "cid", wantClientSec: "",
			wantAuthorize: iamEndpoints["prod"].Authorize, wantToken: "https://custom/token",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "default scope is openid",
			clientID: "cid", iamEnv: "prod", scope: "",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "scope splits on spaces",
			clientID: "cid", iamEnv: "prod", scope: "openid profile email",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid", "profile", "email"}, wantTimeout: defaultTimeout,
		},
		{
			name: "timeout preserved when positive",
			clientID: "cid", iamEnv: "prod", timeout: 42 * time.Second,
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: 42 * time.Second,
		},
		{
			name: "timeout<=0 falls back to default",
			clientID: "cid", iamEnv: "prod", timeout: 0,
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GRN_LOGIN_CLIENT_ID", tc.envClientID)
			t.Setenv("GRN_LOGIN_CLIENT_SECRET", tc.envClientSec)

			cfg, gotTimeout, err := resolveConfig(tc.clientID, tc.clientSecret, tc.iamEnv, tc.authorizeURL, tc.tokenURL, tc.scope, tc.timeout)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%q, want it to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.ClientID != tc.wantClientID {
				t.Errorf("ClientID=%q, want %q", cfg.ClientID, tc.wantClientID)
			}
			if cfg.ClientSecret != tc.wantClientSec {
				t.Errorf("ClientSecret=%q, want %q", cfg.ClientSecret, tc.wantClientSec)
			}
			if tc.wantAuthorize != "" && cfg.AuthorizeURL != tc.wantAuthorize {
				t.Errorf("AuthorizeURL=%q, want %q", cfg.AuthorizeURL, tc.wantAuthorize)
			}
			if tc.wantToken != "" && cfg.TokenURL != tc.wantToken {
				t.Errorf("TokenURL=%q, want %q", cfg.TokenURL, tc.wantToken)
			}
			if !slices.Equal(cfg.Scopes, tc.wantScopes) {
				t.Errorf("Scopes=%v, want %v", cfg.Scopes, tc.wantScopes)
			}
			if gotTimeout != tc.wantTimeout {
				t.Errorf("timeout=%v, want %v", gotTimeout, tc.wantTimeout)
			}
		})
	}
}

// runLogout ignores its *cobra.Command, so a zero value is fine for white-box
// testing. These mutate the package-level storePathFn seam, so non-parallel.
func TestLogout_ClearsTokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, tokenFileName)
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	orig := storePathFn
	storePathFn = func() string { return path }
	defer func() { storePathFn = orig }()

	if err := runLogout(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runLogout: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected token file removed; stat err=%v", err)
	}
}

func TestLogout_IdempotentWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, tokenFileName) // deliberately absent

	orig := storePathFn
	storePathFn = func() string { return path }
	defer func() { storePathFn = orig }()

	if err := runLogout(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runLogout on missing file should be a no-op, got: %v", err)
	}
}

// Ensure the commands carry non-empty Short (the cmd package's
// conventions_test.go enforces this at runtime by walking rootCmd).
func TestCommands_HaveShortHelp(t *testing.T) {
	for _, c := range []*cobra.Command{LoginCmd, LogoutCmd} {
		if c.Short == "" {
			t.Errorf("%q has no Short", c.Name())
		}
	}
}
