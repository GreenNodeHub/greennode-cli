package agentbase

import (
	"strings"
	"testing"

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
