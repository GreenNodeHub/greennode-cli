package agentbase

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	agentbaseconfig "github.com/greennodehub/greennode-cli/internal/agentbase/config"
	"github.com/greennodehub/greennode-cli/internal/auth"
	coreconfig "github.com/greennodehub/greennode-cli/internal/config"
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

// resolveOutputFormat is the Fix A core: an explicit --output/-o wins, else the
// config-file output key, else the flag default. Pure (no ~/.greennode IO) so it
// is unit-testable without touching disk; effectiveOutputFormat is the thin
// wiring that loads the config key and calls this.

func TestResolveOutputFormat(t *testing.T) {
	t.Parallel()
	newCmd := func(setExplicit bool) *cobra.Command {
		c := &cobra.Command{}
		c.Flags().StringP("output", "o", "table", "")
		if setExplicit {
			_ = c.Flags().Set("output", "id")
		}
		return c
	}
	cases := []struct {
		name      string
		cmd       *cobra.Command
		flagValue string
		cfgOutput string
		want      string
	}{
		{"explicit flag wins over config", newCmd(true), "id", "json", "id"},
		{"config fallback when flag unchanged", newCmd(false), "table", "json", "json"},
		{"flag default when config empty", newCmd(false), "table", "", "table"},
		{"nil cmd falls back to config", nil, "table", "json", "json"},
		{"nil cmd empty config uses flag default", nil, "table", "", "table"},
	}
	for _, tc := range cases {
		if got := resolveOutputFormat(tc.cmd, tc.flagValue, tc.cfgOutput); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// overrideEndpointHosts is the Fix B core: --endpoint-url swaps scheme+host of
// every agentbase service endpoint, preserving each service's path so routing
// still works against a real deployment. OAuth2Token (the IAM token URL) is
// untouched. Pure, so unit-testable without a profile on disk.

func TestOverrideEndpointHosts(t *testing.T) {
	t.Parallel()
	prod := agentbaseconfig.EndpointsForEnv(agentbaseconfig.EnvProd)

	// Bare-host swap: every service keeps its path, host becomes the override.
	eps := prod
	if err := overrideEndpointHosts(&eps, "https://staging.example.com"); err != nil {
		t.Fatalf("override: %v", err)
	}
	if eps.Identity != "https://staging.example.com/identity" {
		t.Errorf("Identity=%q, want https://staging.example.com/identity", eps.Identity)
	}
	if eps.Runtime != "https://staging.example.com/runtime" {
		t.Errorf("Runtime=%q, want https://staging.example.com/runtime", eps.Runtime)
	}
	if eps.OAuth2Token != prod.OAuth2Token {
		t.Errorf("OAuth2Token=%q, want %q (IAM token URL must be untouched)", eps.OAuth2Token, prod.OAuth2Token)
	}

	// Full-URL override (e.g. pasted from `agentbase context current`): the
	// override's own path is ignored — only scheme+host are swapped in — so
	// Runtime still gets /runtime, not /identity.
	eps2 := prod
	if err := overrideEndpointHosts(&eps2, "https://staging.example.com/identity"); err != nil {
		t.Fatalf("override: %v", err)
	}
	if eps2.Identity != "https://staging.example.com/identity" {
		t.Errorf("Identity=%q, want https://staging.example.com/identity", eps2.Identity)
	}
	if eps2.Runtime != "https://staging.example.com/runtime" {
		t.Errorf("Runtime=%q, want https://staging.example.com/runtime (override path ignored)", eps2.Runtime)
	}

	// Dev split-host topology: services live on different hosts with different
	// path segments; the swap repoints every host while keeping each path.
	dev := agentbaseconfig.EndpointsForEnv(agentbaseconfig.EnvDev)
	if err := overrideEndpointHosts(&dev, "https://proxy.example.com"); err != nil {
		t.Fatalf("override: %v", err)
	}
	if dev.Identity != "https://proxy.example.com/identity" {
		t.Errorf("dev Identity=%q, want https://proxy.example.com/identity", dev.Identity)
	}
	if dev.Runtime != "https://proxy.example.com/agent-core-runtime" {
		t.Errorf("dev Runtime=%q, want https://proxy.example.com/agent-core-runtime", dev.Runtime)
	}

	// Invalid override (missing scheme/host) is rejected before any endpoint is
	// mutated, so a bad --endpoint-url never leaves a half-overwritten ctx.
	bad := prod
	if err := overrideEndpointHosts(&bad, "not-a-url"); err == nil {
		t.Errorf("expected error for invalid --endpoint-url, got nil")
	} else if bad.Identity != prod.Identity {
		t.Errorf("endpoints mutated on error: Identity=%q, want %q", bad.Identity, prod.Identity)
	}
}
