package agentbase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ini.v1"

	agentbaseconfig "github.com/vngcloud/greennode-cli/internal/agentbase/config"
	"github.com/vngcloud/greennode-cli/internal/auth"
	coreconfig "github.com/vngcloud/greennode-cli/internal/config"
)

// envFromIamEnv maps the profile's iam_env to an agentbase env. The mapping is
// agentbase-specific (not covered by the cli token-provider tests): "dev" → dev,
// anything else (including "" and "prod") → prod, preserving the prod default.
func TestEnvFromIamEnv(t *testing.T) {
	t.Parallel()
	cases := map[string]agentbaseconfig.Env{
		"dev":     agentbaseconfig.EnvDev,
		"prod":    agentbaseconfig.EnvProd,
		"":        agentbaseconfig.EnvProd,
		"staging": agentbaseconfig.EnvProd, // unknown degrades to prod, never panics
	}
	for in, want := range cases {
		if got := envFromIamEnv(in); got != want {
			t.Errorf("envFromIamEnv(%q)=%q, want %q", in, got, want)
		}
	}
}

// newAuthProvider is the unification chokepoint: it must delegate to the shared
// cli.NewTokenProvider so agentbase speaks the same auth idiom as vks/vserver.
// These asserts prove the wiring (agentbaseCtx → shared selector → concrete
// provider type) without re-deriving the env-resolution/rotation coverage the
// cli token-provider tests already own.

func TestNewAuthProvider_UserMode_BuildsLoginTokenProvider(t *testing.T) {
	t.Parallel()
	ab := &agentbaseCtx{shared: &coreconfig.Config{
		Profile:      "default",
		AuthMode:     "user",
		RefreshToken: "rt-123",
		IamEnv:       "dev",
	}}
	tp, err := newAuthProvider(ab)
	if err != nil {
		t.Fatalf("newAuthProvider: %v", err)
	}
	if _, ok := tp.(*auth.LoginTokenProvider); !ok {
		t.Errorf("tp=%T, want *auth.LoginTokenProvider (user mode)", tp)
	}
}

func TestNewAuthProvider_MachineMode_BuildsMachineTokenProvider(t *testing.T) {
	t.Parallel()
	ab := &agentbaseCtx{shared: &coreconfig.Config{
		Profile:      "default",
		ClientID:     "cid",
		ClientSecret: "cs",
	}}
	tp, err := newAuthProvider(ab)
	if err != nil {
		t.Fatalf("newAuthProvider: %v", err)
	}
	if _, ok := tp.(*auth.MachineTokenProvider); !ok {
		t.Errorf("tp=%T, want *auth.MachineTokenProvider (machine mode)", tp)
	}
}

func TestNewAuthProvider_UserMissingRefresh_Errors(t *testing.T) {
	t.Parallel()
	ab := &agentbaseCtx{shared: &coreconfig.Config{Profile: "default", AuthMode: "user", IamEnv: "dev"}}
	_, err := newAuthProvider(ab)
	if err == nil || !strings.Contains(err.Error(), "grn login") {
		t.Errorf("err=%v, want guidance mentioning `grn login`", err)
	}
}

func TestNewAuthProvider_MachineMissingCreds_Errors(t *testing.T) {
	t.Parallel()
	ab := &agentbaseCtx{shared: &coreconfig.Config{Profile: "default"}}
	_, err := newAuthProvider(ab)
	if err == nil || !strings.Contains(err.Error(), "grn configure") {
		t.Errorf("err=%v, want guidance mentioning `grn configure`", err)
	}
}

// --- context switch ---

// seedCreds writes a [profile] section into an isolated HOME's credentials INI
// and clears the GRN_* env vars LoadConfig reads so the file is the sole source.
func seedCreds(t *testing.T, section string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"GRN_PROFILE", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
		t.Setenv(k, "")
	}
	t.Setenv("GRN_PROFILE", "default")
	dir := filepath.Join(coreconfig.DefaultConfigDir())
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte(section), 0600); err != nil {
		t.Fatalf("write creds: %v", err)
	}
}

// A user profile's iam_env is bound to its login token; context switch must
// REFUSE rather than repoint it (which would invalidate the refresh token).
func TestContextSwitch_UserProfileRefused(t *testing.T) {
	seedCreds(t, "[default]\nauth_mode = user\nrefresh_token = rt\niam_env = dev\n")
	if err := contextSwitchCmd.RunE(contextSwitchCmd, []string{"prod"}); err == nil {
		t.Fatal("context switch on a user profile succeeded, want error")
	} else if !strings.Contains(err.Error(), "grn login --iam-env") {
		t.Errorf("err=%q, want guidance to re-login with 'grn login --iam-env'", err)
	}
	// iam_env must NOT have been overwritten on refusal.
	creds, err := ini.Load(filepath.Join(coreconfig.DefaultConfigDir(), "credentials"))
	if err != nil {
		t.Fatalf("load creds: %v", err)
	}
	if got := creds.Section("default").Key("iam_env").String(); got != "dev" {
		t.Errorf("iam_env=%q after refused switch, want dev (untouched)", got)
	}
}

// A machine profile switches freely: context switch writes iam_env to the
// shared profile, and all three services resolve env from it.
func TestContextSwitch_MachineProfileWritesIamEnv(t *testing.T) {
	seedCreds(t, "[default]\nclient_id = cid\nclient_secret = cs\niam_env = prod\n")
	if err := contextSwitchCmd.RunE(contextSwitchCmd, []string{"dev"}); err != nil {
		t.Fatalf("context switch on a machine profile: %v", err)
	}
	cfg, err := coreconfig.LoadConfig("default")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if cfg.IamEnv != "dev" {
		t.Errorf("iam_env=%q after switch, want dev", cfg.IamEnv)
	}
	// The agentbase env follows iam_env.
	if env := envFromIamEnv(cfg.IamEnv); env != agentbaseconfig.EnvDev {
		t.Errorf("envFromIamEnv=%q, want dev", env)
	}
}

// context switch rejects a bogus env before touching the profile.
func TestContextSwitch_InvalidEnvRejected(t *testing.T) {
	seedCreds(t, "[default]\nclient_id = cid\nclient_secret = cs\n")
	if err := contextSwitchCmd.RunE(contextSwitchCmd, []string{"staging"}); err == nil {
		t.Fatal("context switch accepted 'staging', want error")
	}
}
