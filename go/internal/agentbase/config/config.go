// Package config holds the dev/prod endpoint catalog for the GreenNode
// AgentBase platform.
//
// It is a PURE constants/map helper: no file IO, no credentials, no state. The
// active environment is no longer selected here — agentbase now shares grn's
// ~/.greennode profile and resolves its env from the profile's iam_env key
// (default prod) exactly like vks/vserver. Callers (cmd/agentbase) map iam_env
// → Env via envFromIamEnv and resolve endpoints via EndpointsForEnv.
//
// Compiled in ONLY with `-tags agentbase`.
package config

// Env represents a deployment environment.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// Endpoints holds the resolved API base URLs for a given environment. These are
// the agentbase service endpoints (identity/runtime/memory/gateway/policy/cr)
// plus the IAM v2 token URL — identical to login.IamEndpoints[env].Token, kept
// here as the agentbase service-endpoint map so the agentbase subtree need not
// import internal/login for endpoint resolution.
type Endpoints struct {
	Identity    string
	Runtime     string
	Memory      string
	Gateway     string
	Policy      string
	Cr          string
	OAuth2Token string
}

var endpointsByEnv = map[Env]Endpoints{
	EnvDev: {
		Identity:    "https://agentbase.api-dev.vngcloud.tech/identity",
		Runtime:     "https://pub-iamapis.api-dev.vngcloud.tech/agent-core-runtime",
		Memory:      "https://pub-iamapis.api-dev.vngcloud.tech/agent-core-memory",
		Gateway:     "https://agentbase.api-dev.vngcloud.tech/gateway",
		Policy:      "https://agentbase.api-dev.vngcloud.tech/policy",
		Cr:          "https://agentbase.api-dev.vngcloud.tech/cr",
		OAuth2Token: "https://pub-iamapis.api-dev.vngcloud.tech/accounts-api/v2/auth/token",
	},
	EnvProd: {
		Identity:    "https://agentbase.api.vngcloud.vn/identity",
		Runtime:     "https://agentbase.api.vngcloud.vn/runtime",
		Memory:      "https://agentbase.api.vngcloud.vn/memory",
		Gateway:     "https://agentbase.api.vngcloud.vn/gateway",
		Policy:      "https://agentbase.api.vngcloud.vn/policy",
		Cr:          "https://agentbase.api.vngcloud.vn/cr",
		OAuth2Token: "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token",
	},
}

// EndpointsForEnv returns the resolved endpoints for env. An unknown env falls
// back to prod (the default environment), so a malformed iam_env never panics.
func EndpointsForEnv(env Env) Endpoints {
	if eps, ok := endpointsByEnv[env]; ok {
		return eps
	}
	return endpointsByEnv[EnvProd]
}
