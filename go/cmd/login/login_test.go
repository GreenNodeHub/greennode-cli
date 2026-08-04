package login

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"

	"github.com/greennodehub/greennode-cli/internal/config"
	loginpkg "github.com/greennodehub/greennode-cli/internal/login"
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
			name:     "flag client-id wins over env",
			clientID: "cid-flag", envClientID: "cid-env", iamEnv: "prod",
			wantClientID: "cid-flag", wantClientSec: "",
			wantAuthorize: loginpkg.IamEndpoints["prod"].Authorize, wantToken: loginpkg.IamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:        "env client-id used when flag empty (env > embedded)",
			envClientID: "cid-env", iamEnv: "dev", // dev has an embedded default that env must beat
			wantClientID:  "cid-env",
			wantAuthorize: loginpkg.IamEndpoints["dev"].Authorize, wantToken: loginpkg.IamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			// The baked-in prod client_id is used automatically when flag/env
			// omit it — a real registered public client, so prod login works.
			name:         "embedded prod client_id used when flag/env omitted",
			iamEnv:       "prod",
			wantClientID: loginpkg.ProdClientID, wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:         "embedded dev client_id used when flag/env omitted",
			iamEnv:       "dev",
			wantClientID: loginpkg.DevClientID, wantClientSec: "",
			wantAuthorize: loginpkg.IamEndpoints["dev"].Authorize, wantToken: loginpkg.IamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "invalid iam-env errors",
			clientID: "cid", iamEnv: "staging",
			wantErr: "iam-env",
		},
		{
			name:     "GRN_IAM_ENV selects dev when iam-env flag empty",
			clientID: "cid", iamEnv: "", envIamEnv: "dev",
			wantClientID:  "cid",
			wantAuthorize: loginpkg.IamEndpoints["dev"].Authorize, wantToken: loginpkg.IamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			// Explicit --iam-env (non-empty) beats GRN_IAM_ENV. resolveConfig only
			// consults GRN_IAM_ENV when iamEnv == "".
			name:     "explicit iam-env beats GRN_IAM_ENV",
			clientID: "cid", iamEnv: "prod", envIamEnv: "dev",
			wantClientID:  "cid",
			wantAuthorize: loginpkg.IamEndpoints["prod"].Authorize, wantToken: loginpkg.IamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "iam-env empty + GRN_IAM_ENV empty defaults to prod",
			clientID: "cid", iamEnv: "", envIamEnv: "",
			wantClientID:  "cid",
			wantAuthorize: loginpkg.IamEndpoints["prod"].Authorize, wantToken: loginpkg.IamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "client-secret flag wins over env",
			clientID: "cid", clientSecret: "cs-flag", envClientSec: "cs-env", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "cs-flag",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "client-secret env fallback",
			clientID: "cid", envClientSec: "cs-env", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "cs-env",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "empty secret → public client path",
			clientID: "cid", iamEnv: "prod",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "dev preset swaps token URL",
			clientID: "cid", iamEnv: "dev",
			wantClientID:  "cid",
			wantAuthorize: loginpkg.IamEndpoints["dev"].Authorize, wantToken: loginpkg.IamEndpoints["dev"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "authorize-url overrides preset only",
			clientID: "cid", iamEnv: "prod", authorizeURL: "https://custom/auth",
			wantClientID: "cid", wantClientSec: "",
			wantAuthorize: "https://custom/auth", wantToken: loginpkg.IamEndpoints["prod"].Token,
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "token-url overrides preset only",
			clientID: "cid", iamEnv: "prod", tokenURL: "https://custom/token",
			wantClientID: "cid", wantClientSec: "",
			wantAuthorize: loginpkg.IamEndpoints["prod"].Authorize, wantToken: "https://custom/token",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "default scope is openid",
			clientID: "cid", iamEnv: "prod", scope: "",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "scope splits on spaces",
			clientID: "cid", iamEnv: "prod", scope: "openid profile email",
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid", "profile", "email"}, wantTimeout: defaultTimeout,
		},
		{
			name:     "timeout preserved when positive",
			clientID: "cid", iamEnv: "prod", timeout: 42 * time.Second,
			wantClientID: "cid", wantClientSec: "",
			wantScopes: []string{"openid"}, wantTimeout: 42 * time.Second,
		},
		{
			name:     "timeout<=0 falls back to default",
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
	if got := resolveIamEnv(c); got != loginpkg.DefaultIamEnv {
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
	if err := w.WriteLoginToken("default", "rt-xyz", time.Now().UTC(), "user", "dev"); err != nil {
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

// TestPromptWithDefault covers the three arms of the prompt: empty input keeps
// the default, EOF keeps the default (non-interactive stdin), typed input wins.
func TestPromptWithDefault(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		def   string
		want  string
	}{
		{"empty input keeps default", "\n", "HCM-3", "HCM-3"},
		{"EOF keeps default (non-tty)", "", "HCM-3", "HCM-3"},
		{"typed input wins", "HAN\n", "HCM-3", "HAN"},
		{"typed input trimmed", "  HAN  \n", "HCM-3", "HAN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bufio.NewReader(strings.NewReader(tc.input))
			if got := promptWithDefault(r, "Default region name", tc.def); got != tc.want {
				t.Errorf("promptWithDefault(%q, %q)=%q, want %q", tc.input, tc.def, got, tc.want)
			}
		})
	}
}

// TestResolveDefaultRegion covers validation: a known region passes through, an
// unknown/empty region falls back to "HCM-3" (matching `grn configure`), and an
// empty input with an existing current region keeps it.
func TestResolveDefaultRegion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		current string
		want    string
	}{
		{"known region kept", "HAN\n", "", "HAN"},
		{"unknown region falls back", "bogus\n", "", "HCM-3"},
		{"empty input keeps current", "\n", "HAN", "HAN"},
		{"empty input + no current falls back", "\n", "", "HCM-3"},
		{"EOF + no current falls back", "", "", "HCM-3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := bufio.NewReader(strings.NewReader(tc.input))
			if got := resolveDefaultRegion(r, tc.current); got != tc.want {
				t.Errorf("resolveDefaultRegion(%q, current=%q)=%q, want %q", tc.input, tc.current, got, tc.want)
			}
		})
	}
}

// TestPromptDefaultRegion_PreservesExistingConfig asserts the region prompt
// only changes region and leaves output/project_id untouched — login must not
// clobber a prior `grn configure` (especially an auto-detected project_id).
//
// NOT parallel: mutates process HOME.
func TestPromptDefaultRegion_PreservesExistingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Pre-seed a configured profile: region=HCM-3, output=table, project_id=proj-1.
	if err := config.NewConfigFileWriter().WriteConfig("default", "HCM-3", "table", "proj-1"); err != nil {
		t.Fatalf("seed WriteConfig: %v", err)
	}
	// Feed "HAN" at the prompt → region becomes HAN; output/project_id preserved.
	r := bufio.NewReader(strings.NewReader("HAN\n"))
	region, err := promptDefaultRegion("default", r)
	if err != nil {
		t.Fatalf("promptDefaultRegion: %v", err)
	}
	if region != "HAN" {
		t.Errorf("region=%q, want HAN", region)
	}
	cfg, err := config.LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Region != "HAN" {
		t.Errorf("persisted region=%q, want HAN", cfg.Region)
	}
	if cfg.Output != "table" {
		t.Errorf("output=%q, want table (preserved)", cfg.Output)
	}
	if cfg.ProjectID != "proj-1" {
		t.Errorf("project_id=%q, want proj-1 (preserved)", cfg.ProjectID)
	}
}

// TestPromptDefaultRegion_BootstrapsFreshProfile asserts a brand-new profile
// (no config file) gets a usable default: non-interactive stdin (EOF) falls
// back to "HCM-3" and output defaults to "json", so a login-only profile is
// immediately ready for vks/vserver without a separate `grn configure`.
//
// NOT parallel: mutates process HOME.
func TestPromptDefaultRegion_BootstrapsFreshProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// No config file yet; EOF stdin (headless login).
	r := bufio.NewReader(strings.NewReader(""))
	region, err := promptDefaultRegion("default", r)
	if err != nil {
		t.Fatalf("promptDefaultRegion: %v", err)
	}
	if region != "HCM-3" {
		t.Errorf("region=%q, want HCM-3 (fallback for headless fresh profile)", region)
	}
	// The config file must now exist with region=HCM-3, output=json.
	iniFile, err := ini.Load(filepath.Join(config.DefaultConfigDir(), "config"))
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	sec, err := iniFile.GetSection("default")
	if err != nil {
		t.Fatalf("no [default] section: %v", err)
	}
	if got := sec.Key("region").String(); got != "HCM-3" {
		t.Errorf("persisted region=%q, want HCM-3", got)
	}
	if got := sec.Key("output").String(); got != "json" {
		t.Errorf("output=%q, want json (default for a fresh profile)", got)
	}
	// promptDefaultRegion must NOT touch the credentials file (login token only).
	if _, err := os.Stat(filepath.Join(config.DefaultConfigDir(), "credentials")); err == nil {
		t.Error("credentials file was created by the region prompt (should only write config)")
	}
}
