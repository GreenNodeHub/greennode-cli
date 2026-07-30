package login

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/config"
)

// resolveConfig is the pure cobra-slice seam; these tests cover client-id
// precedence (flag > env > baked-in per-env default), iam-env precedence
// (flag > GRN_IAM_ENV > prod default), the iam-env presets and piecewise
// overrides, scope splitting, and the public-client (empty-secret) path.
//
// They are deliberately non-parallel: resolveConfig reads process env
// (GRN_LOGIN_*, GRN_IAM_ENV) via os.Getenv, and t.Setenv is not safe across
// parallel tests in the same process.
func TestResolveConfig(t *testing.T) {
	// Start from a clean env so each case controls its inputs exactly.
	t.Setenv("GRN_LOGIN_CLIENT_ID", "")
	t.Setenv("GRN_LOGIN_CLIENT_SECRET", "")
	t.Setenv("GRN_IAM_ENV", "")

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
		envIamEnv     string
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
			name: "env client-id used when flag empty (env > embedded)",
			envClientID: "cid-env", iamEnv: "dev", // dev has an embedded default that env must beat
			wantClientID: "cid-env",
			wantAuthorize: iamEndpoints["dev"].Authorize, wantToken: iamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			// Was "missing client id errors" — now the baked-in prod placeholder
			// is used automatically, so login no longer refuses; prod just fails
			// at IAM until the placeholder is replaced.
			name: "embedded prod placeholder used when flag/env omitted",
			iamEnv: "prod",
			wantClientID: prodClientIDPlaceholder, wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "embedded dev client_id used when flag/env omitted",
			iamEnv: "dev",
			wantClientID: devClientID, wantClientSec: "",
			wantAuthorize: iamEndpoints["dev"].Authorize, wantToken: iamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "invalid iam-env errors",
			clientID: "cid", iamEnv: "staging",
			wantErr: "iam-env",
		},
		{
			name: "GRN_IAM_ENV selects dev when iam-env flag empty",
			clientID: "cid", iamEnv: "", envIamEnv: "dev",
			wantClientID: "cid",
			wantAuthorize: iamEndpoints["dev"].Authorize, wantToken: iamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			// Explicit --iam-env (non-empty) beats GRN_IAM_ENV. resolveConfig only
			// consults GRN_IAM_ENV when iamEnv == "".
			name: "explicit iam-env beats GRN_IAM_ENV",
			clientID: "cid", iamEnv: "prod", envIamEnv: "dev",
			wantClientID: "cid",
			wantAuthorize: iamEndpoints["prod"].Authorize, wantToken: iamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name: "iam-env empty + GRN_IAM_ENV empty defaults to prod",
			clientID: "cid", iamEnv: "", envIamEnv: "",
			wantClientID: "cid",
			wantAuthorize: iamEndpoints["prod"].Authorize, wantToken: iamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
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
			t.Setenv("GRN_IAM_ENV", tc.envIamEnv)

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

// resolveClientID is the pure client-id resolution helper (flag > env >
// embedded). The error path is unreachable via resolveConfig with the shipped
// presets (both have embedded defaults), so it is exercised directly here with
// an empty embedded default.
func TestResolveClientID(t *testing.T) {
	t.Setenv("GRN_LOGIN_CLIENT_ID", "")
	// flag wins over env and embedded.
	t.Setenv("GRN_LOGIN_CLIENT_ID", "cid-env")
	got, err := resolveClientID("cid-flag", "cid-embedded")
	if err != nil || got != "cid-flag" {
		t.Fatalf("flag should win: got=%q err=%v", got, err)
	}
	// env wins over embedded.
	got, err = resolveClientID("", "cid-embedded")
	if err != nil || got != "cid-env" {
		t.Fatalf("env should beat embedded: got=%q err=%v", got, err)
	}
	// embedded used when flag+env empty.
	t.Setenv("GRN_LOGIN_CLIENT_ID", "")
	got, err = resolveClientID("", "cid-embedded")
	if err != nil || got != "cid-embedded" {
		t.Fatalf("embedded fallback: got=%q err=%v", got, err)
	}
	// all empty → error (the defensive guard).
	_, err = resolveClientID("", "")
	if err == nil || !strings.Contains(err.Error(), "client-id") {
		t.Fatalf("expected client-id error when all empty, got %v", err)
	}
}

// resolveIamEnv applies flag > GRN_IAM_ENV > prod default. Built on a real
// cobra command whose --iam-env flag is registered with no default (mirroring
// init), so an unset flag falls through to env. Non-parallel: mutates GRN_IAM_ENV.
func TestResolveIamEnv(t *testing.T) {
	t.Setenv("GRN_IAM_ENV", "")
	newCmd := func() *cobra.Command {
		c := &cobra.Command{}
		c.Flags().String("iam-env", "", "")
		return c
	}
	// unset flag + unset env → prod default.
	c := newCmd()
	if got := resolveIamEnv(c); got != defaultIamEnv {
		t.Errorf("unset → %q, want prod", got)
	}
	// unset flag + env=dev → dev.
	t.Setenv("GRN_IAM_ENV", "dev")
	c = newCmd()
	if got := resolveIamEnv(c); got != "dev" {
		t.Errorf("env dev → %q, want dev", got)
	}
	// explicit flag beats env.
	t.Setenv("GRN_IAM_ENV", "dev")
	c = newCmd()
	_ = c.Flags().Set("iam-env", "prod")
	if got := resolveIamEnv(c); got != "prod" {
		t.Errorf("flag prod over env dev → %q, want prod", got)
	}
}

// runLogout resolves the profile from cmd (flag → GRN_PROFILE → default) and
// clears that profile's login keys via the config-layer writer. These isolate
// HOME at a temp dir so the writer targets a throwaway ~/.greennode. Non-parallel:
// they mutate process env (HOME, GRN_PROFILE).
func TestLogout_ClearsLoginKeysKeepsMachineCreds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	w := config.NewConfigFileWriter()
	// Seed the default profile with machine creds + a login token.
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	if err := w.WriteLoginToken("default", "rt-xyz", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}

	if err := runLogout(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runLogout: %v", err)
	}

	cfg, err := config.LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig after logout: %v", err)
	}
	// Login keys gone.
	if cfg.RefreshToken != "" {
		t.Errorf("RefreshToken=%q, want empty after logout", cfg.RefreshToken)
	}
	if cfg.AuthMode != "" {
		t.Errorf("AuthMode=%q, want empty after logout", cfg.AuthMode)
	}
	if cfg.LoginClientID != "" {
		t.Errorf("LoginClientID=%q, want empty after logout", cfg.LoginClientID)
	}
	// Machine creds survive logout (auth-only clear, not a credentials wipe).
	if cfg.ClientID != "cid" {
		t.Errorf("ClientID=%q, want cid (logout must keep machine creds)", cfg.ClientID)
	}
	if cfg.ClientSecret != "cs" {
		t.Errorf("ClientSecret=%q, want cs", cfg.ClientSecret)
	}
}

func TestLogout_IdempotentWhenNoCredsFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	// No credentials file at all — logout must be a no-op, not an error.
	if err := runLogout(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runLogout on missing creds file should be a no-op, got: %v", err)
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
