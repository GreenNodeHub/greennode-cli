package agentbase

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	agentbaseconfig "github.com/vngcloud/greennode-cli/internal/agentbase/config"
	"github.com/vngcloud/greennode-cli/internal/cli"
	coreclient "github.com/vngcloud/greennode-cli/internal/client"
	coreconfig "github.com/vngcloud/greennode-cli/internal/config"
)

// agentbaseCtx is the resolved per-invocation context for the agentbase
// subtree: the shared ~/.greennode profile (creds, auth_mode, iam_env,
// agent_identity) plus the agentbase env + endpoints derived from iam_env.
// agentbase no longer carries its own .greennode.json — it reads the same
// profile every other service uses, so one `grn configure`/`grn login` feeds
// vks, vserver, AND agentbase.
type agentbaseCtx struct {
	shared    *coreconfig.Config
	env       agentbaseconfig.Env
	endpoints agentbaseconfig.Endpoints
}

// resolveProfile mirrors cmd/login/cmd/configure: --profile flag → GRN_PROFILE
// → "default". The root --profile is a persistent flag inherited by the
// agentbase subtree, so cmd.Flag("profile") resolves it. agentbase acts on the
// resolved profile's credentials section so dev/prod profiles each hold their
// own creds/agent_identity. The nil-flag guard lets a zero *cobra.Command
// (white-box tests) resolve to the default/env profile instead of panicking.
func resolveProfile(cmd *cobra.Command) string {
	profile := ""
	if cmd != nil {
		if f := cmd.Flag("profile"); f != nil {
			profile = f.Value.String()
		}
	}
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}
	return profile
}

// envFromIamEnv maps the profile's iam_env to an agentbase env. "dev" → dev;
// anything else (including "" and "prod") → prod, preserving today's
// prod-default machine behavior when iam_env is unset.
func envFromIamEnv(iamEnv string) agentbaseconfig.Env {
	if iamEnv == string(agentbaseconfig.EnvDev) {
		return agentbaseconfig.EnvDev
	}
	return agentbaseconfig.EnvProd
}

// mustLoadAgentbaseCtx loads the shared profile and resolves the agentbase env
// + endpoints from iam_env. It exits on a LoadConfig error (matching the former
// mustLoadConfig behavior for display commands); credential/token errors are
// NOT fatal here — they surface from newAuthProvider/newIdentityClient so RunE
// can return them.
func mustLoadAgentbaseCtx(cmd *cobra.Command) *agentbaseCtx {
	shared, err := coreconfig.LoadConfig(resolveProfile(cmd))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	env := envFromIamEnv(shared.IamEnv)
	return &agentbaseCtx{
		shared:    shared,
		env:       env,
		endpoints: agentbaseconfig.EndpointsForEnv(env),
	}
}

// newAuthProvider is the single shared auth selector for agentbase — identical
// to vks/vserver. It branches on the profile's auth_mode via cli.NewTokenProvider:
// "user" → LoginTokenProvider (refresh-token grant against the v2 iam_env URL),
// else → MachineTokenProvider (client_credentials against the v2 iam_env URL).
// No agentbase-specific provider, no adapter — agentbase joins vks/vserver
// verbatim. The returned error (e.g. "credentials not configured") is for RunE.
func newAuthProvider(ab *agentbaseCtx) (coreclient.TokenProvider, error) {
	return cli.NewTokenProvider(ab.shared)
}
