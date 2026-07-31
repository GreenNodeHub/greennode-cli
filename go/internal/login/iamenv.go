package login

import "fmt"

// IamEndpoint is one IAM environment's OAuth endpoints plus the baked-in
// public client_id for that env. These are PUBLIC OAuth client identifiers
// (RFC 6749) — not secrets — so baking the client_id is safe; the client_secret
// is NEVER baked in (the dev client is public/no-secret). This map is the
// single source of truth shared by the `grn login` authorize flow and the
// subcommand refresh-token path (which needs the /v2 token URL given an env).
// Mirrors the endpoints previously inlined (unexported) in cmd/login/login.go.
type IamEndpoint struct {
	Authorize string
	Token     string
	ClientID  string
}

// IamEndpoints holds the prod/dev IAM presets. The Token URL is the /v2 PKCE
// endpoint (distinct from the machine client_credentials v1 endpoint in
// internal/auth/token.go). Authorize is the IAM signin URL the browser opens.
var IamEndpoints = map[string]IamEndpoint{
	"prod": {
		Authorize: "https://signin.vngcloud.vn/ap/auth",
		Token:     "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token",
		ClientID:  ProdClientID,
	},
	"dev": {
		Authorize: "https://dev-signin.vngcloud.tech/ap/auth",
		Token:     "https://pub-iamapis.api-dev.vngcloud.tech/accounts-api/v2/auth/token",
		ClientID:  DevClientID,
	},
}

// DevClientID is the public (no-secret, redirect-*) OAuth client registered on
// the IAM dev portal. A non-secret identifier; safe to bake in. Used as the
// default client_id for `grn login --iam-env dev` and for the refresh path on a
// dev login profile.
const DevClientID = "70a17ade-b887-4354-9ecb-cfcfd06150b0"

// ProdClientID is the public (no-secret, redirect-*) OAuth client registered on
// the IAM prod portal for `grn login`. A non-secret identifier; safe to bake in
// (only the client_secret is secret, and it is never baked). Used as the default
// client_id for `grn login --iam-env prod` (and the prod default) and for the
// refresh path on a prod login profile.
const ProdClientID = "09b427e9-3b45-437b-a3a6-bf9f7fd24185"

// DefaultIamEnv is the iam-env assumed when neither a flag nor GRN_IAM_ENV
// selects one. Prod is the safe default (matches the console).
const DefaultIamEnv = "prod"

// TokenURLForEnv returns the /v2 PKCE token endpoint for the given env. A
// refresh path uses this to mint fresh access tokens from the persisted refresh
// token. Empty or unknown env → error (a profile without a valid iam_env cannot
// be refreshed safely).
func TokenURLForEnv(env string) (string, error) {
	if env == "" {
		return "", fmt.Errorf("iam_env is not set for this profile — run `grn login` again")
	}
	ep, ok := IamEndpoints[env]
	if !ok {
		return "", fmt.Errorf("unknown iam_env %q (valid: %s)", env, validIamEnvs())
	}
	return ep.Token, nil
}

// ClientIDForEnv returns the baked-in public client_id for the given env — the
// no-secret OAuth client `grn login` uses for that env, and the client the
// refresh-token grant must present (the refresh token is bound to it at login).
// The subcommand refresh path resolves the client_id from a profile's iam_env
// via this helper rather than reading a persisted login_client_id — the value
// is a public identifier already baked into source (iamenv.go), so storing it
// again in the credentials INI is redundant. Empty/unknown env → error.
func ClientIDForEnv(env string) (string, error) {
	if env == "" {
		return "", fmt.Errorf("iam_env is not set for this profile — run `grn login` again")
	}
	ep, ok := IamEndpoints[env]
	if !ok {
		return "", fmt.Errorf("unknown iam_env %q (valid: %s)", env, validIamEnvs())
	}
	return ep.ClientID, nil
}

// EndpointForEnv returns the full preset for an env (authorize/token/client_id),
// used by the `grn login` authorize flow. Unknown env → error.
func EndpointForEnv(env string) (IamEndpoint, error) {
	if env == "" {
		return IamEndpoint{}, fmt.Errorf("iam_env is empty (valid: %s)", validIamEnvs())
	}
	ep, ok := IamEndpoints[env]
	if !ok {
		return IamEndpoint{}, fmt.Errorf("invalid iam_env %q (valid: %s)", env, validIamEnvs())
	}
	return ep, nil
}

func validIamEnvs() string {
	return "prod, dev"
}
