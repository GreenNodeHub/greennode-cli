package agentbase

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	agentbaseconfig "github.com/greennodehub/greennode-cli/internal/agentbase/config"
	"github.com/greennodehub/greennode-cli/internal/cli"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
	coreconfig "github.com/greennodehub/greennode-cli/internal/config"
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
//
// The root --endpoint-url override is honored here, the same flag vks/vserver
// consume via cli.NewClient: when set, cli.CheckEndpoint enforces the endpoint
// safety policy (trusted-host/TLS/allow-untrusted) and overrideEndpointHosts
// repoints the agentbase service endpoints at it. Without this, agentbase alone
// ignored --endpoint-url and always used the iam_env-derived endpoints.
func mustLoadAgentbaseCtx(cmd *cobra.Command) *agentbaseCtx {
	shared, err := coreconfig.LoadConfig(resolveProfile(cmd))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	env := envFromIamEnv(shared.IamEnv)
	endpoints := agentbaseconfig.EndpointsForEnv(env)

	if endpointURL := flagString(cmd, "endpoint-url"); endpointURL != "" {
		if err := cli.CheckEndpoint(endpointURL, flagBool(cmd, "no-verify-ssl"), flagBool(cmd, "allow-untrusted-endpoint")); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if err := overrideEndpointHosts(&endpoints, endpointURL); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}

	return &agentbaseCtx{
		shared:    shared,
		env:       env,
		endpoints: endpoints,
	}
}

// flagString/flagBool are nil-safe persistent-flag readers. agentbase's helpers
// accept a zero *cobra.Command from white-box tests (see resolveProfile's nil
// guard); these keep that contract for the endpoint-url/no-verify-ssl flags.
func flagString(cmd *cobra.Command, name string) string {
	if cmd != nil {
		if v, err := cmd.Flags().GetString(name); err == nil {
			return v
		}
	}
	return ""
}

func flagBool(cmd *cobra.Command, name string) bool {
	if cmd != nil {
		if v, err := cmd.Flags().GetBool(name); err == nil {
			return v
		}
	}
	return false
}

// resolveOutputFormat picks the effective agentbase output format: an explicit
// --output/-o wins; otherwise the profile's config-file output key; otherwise
// flagValue (the flag's own default, "table"). This is the pure core of
// effectiveOutputFormat — kept IO-free so it is unit-testable without a
// ~/.greennode on disk. agentbase keeps its own --output/-o flag (shorthand +
// table default) rather than inheriting the root --output, so this fallback is
// what makes ~/.greennode/config's output key govern agentbase like the rest of
// the CLI (cli.Output applies the same fallback for vks/vserver).
func resolveOutputFormat(cmd *cobra.Command, flagValue, cfgOutput string) string {
	if cmd != nil && cmd.Flags().Changed("output") {
		return flagValue
	}
	if cfgOutput != "" {
		return cfgOutput
	}
	return flagValue
}

// effectiveOutputFormat is the PersistentPreRun wiring for resolveOutputFormat:
// it loads the resolved profile's config-file output key (missing config is
// non-fatal here — the command's RunE will surface a real LoadConfig error via
// mustLoadAgentbaseCtx) and delegates to resolveOutputFormat.
func effectiveOutputFormat(cmd *cobra.Command, flagValue string) string {
	cfgOutput := ""
	if cfg, err := coreconfig.LoadConfig(resolveProfile(cmd)); err == nil {
		cfgOutput = cfg.Output
	}
	return resolveOutputFormat(cmd, flagValue, cfgOutput)
}

// overrideEndpointHosts repoints every agentbase service endpoint at
// endpointURL by swapping its scheme+host[:port], preserving the per-service
// path (e.g. /identity, /runtime, /agent-core-runtime). This lets
// --endpoint-url point agentbase at a different deployment while keeping
// service routing intact — a bare host ("https://staging.example.com") reaches
// every service at its real path, and even a full service URL pasted from
// `agentbase context current` works (its path is ignored, only the host is
// swapped in). OAuth2Token is untouched: it is the IAM token URL the auth
// provider resolves from iam_env, not a service call, so overriding it would
// lie in `agentbase context current` about where tokens are minted.
func overrideEndpointHosts(eps *agentbaseconfig.Endpoints, endpointURL string) error {
	override, err := url.Parse(endpointURL)
	if err != nil {
		return fmt.Errorf("invalid --endpoint-url %q: %w", endpointURL, err)
	}
	if override.Scheme == "" || override.Host == "" {
		return fmt.Errorf("invalid --endpoint-url %q: want scheme://host[:port]", endpointURL)
	}
	for _, p := range []*string{&eps.Identity, &eps.Runtime, &eps.Memory, &eps.Gateway, &eps.Policy, &eps.Cr} {
		u, err := url.Parse(*p)
		if err != nil {
			return fmt.Errorf("invalid agentbase endpoint %q: %w", *p, err)
		}
		u.Scheme = override.Scheme
		u.Host = override.Host
		*p = u.String()
	}
	return nil
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
