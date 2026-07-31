package cli

import (
	"fmt"
	"time"

	"github.com/vngcloud/greennode-cli/internal/auth"
	"github.com/vngcloud/greennode-cli/internal/client"
	"github.com/vngcloud/greennode-cli/internal/config"
	"github.com/vngcloud/greennode-cli/internal/login"
)

// tokenURLForEnv resolves the IAM /v2 token URL for a given iam_env. It is a
// package-level var (not a direct call to login.TokenURLForEnv) so tests can
// point it at a httptest server and drive a refresh-grant rotation through
// NewTokenProvider → LoginTokenProvider, asserting the rotated refresh token
// persists to cfg.Profile's section (the "empty section name" regression would
// otherwise be silent — a warning, not a hard failure).
var tokenURLForEnv = login.TokenURLForEnv

// clientIDForEnv resolves the baked-in public client_id for a profile's iam_env.
// A package-level var (mirroring tokenURLForEnv) so tests can drive the refresh
// path without depending on the real IAM presets. The refresh-token grant must
// present the same client_id the refresh token was issued to at login; since
// `grn login` uses the baked-in per-env public client, that id is recovered from
// iam_env rather than persisted in the credentials INI (the value is public and
// already in source).
var clientIDForEnv = login.ClientIDForEnv

// NewTokenProvider picks the auth source for a profile from cfg.AuthMode — one
// auth type per profile, profile-driven (no per-call flag/env override). The
// profile commits to a single auth type:
//
//   - "user" → LoginTokenProvider: mints short-lived access tokens from the
//     persisted refresh token via the IAM /v2 refresh_token grant. Requires the
//     login context `grn login` persisted (refresh_token + iam_env); missing →
//     hard error pointing to `grn login` (no silent fallback to machine creds).
//     The client_id is NOT persisted — it is a public id baked into source and
//     resolved from iam_env (login.ClientIDForEnv). A rotation callback writes
//     any rotated refresh token + new expiry back to the credentials INI
//     (atomic 0600) so later invocations don't see a stale token.
//   - anything else (unset / "machine") → the machine client_credentials
//     TokenManager (today's behavior). Requires machine client_id/client_secret;
//     missing → the existing "run `grn configure`" error.
//
// The profile is read from cfg.Profile — the RESOLVED profile from LoadConfig
// (always non-empty in production: flag → GRN_PROFILE → "default") — NOT the
// raw --profile flag, which is "" when unset. The rotation callback closes over
// this resolved name because WriteLoginToken writes a section named after the
// profile; passing the raw (empty) flag made rotation fail with "empty section
// name" (the persisted section can't be "").
//
// Shared by cli.NewClient (vks) and vserverclient.BuildClient (vserver) so both
// services select auth the same way.
func NewTokenProvider(cfg *config.Config) (client.TokenProvider, error) {
	profile := cfg.Profile
	switch cfg.AuthMode {
	case "user":
		if cfg.RefreshToken == "" {
			return nil, fmt.Errorf("profile %q is auth_mode=user but has no login token — run `grn login`", profile)
		}
		tokenURL, err := tokenURLForEnv(cfg.IamEnv)
		if err != nil {
			return nil, err
		}
		// The client_id is NOT persisted (a public id baked into source); resolve
		// it from the profile's iam_env so the refresh-token grant presents the
		// same client the refresh token was issued to at `grn login`.
		clientID, err := clientIDForEnv(cfg.IamEnv)
		if err != nil {
			return nil, err
		}
		// Best-effort rotation write: if IAM issues a new refresh_token on
		// refresh, persist it + the new expiry under the same profile/login
		// context so the next invocation refreshes against the rotated token.
		persist := func(rt string, exp time.Time) error {
			return config.NewConfigFileWriter().WriteLoginToken(profile, rt, exp, "user", cfg.IamEnv)
		}
		// clientSecret="": `grn login` uses a public/no-secret client (the
		// baked-in dev/prod id); client_secret is never persisted (spec), so the
		// refresh sends Basic(client_id, "") exactly like the public-client
		// authorize-flow token POST (internal/login/tokencx.go).
		return auth.NewLoginTokenProvider(cfg.RefreshToken, clientID, "", tokenURL, persist), nil
	default: // "" or "machine"
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, fmt.Errorf("credentials not configured. Run 'grn configure' to set up credentials")
		}
		return auth.NewTokenManager(cfg.ClientID, cfg.ClientSecret), nil
	}
}
