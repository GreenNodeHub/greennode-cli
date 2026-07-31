// Package login wires the `grn login` / `grn logout` cobra commands around the
// internal/login PKCE library. It resolves a login.Config from flags + env plus
// a baked-in per-env public client_id (dev and prod both real; the
// client_secret is never baked in), runs the browser PKCE flow, and folds the
// refresh token (0600) + non-secret refresh context into the per-profile
// ~/.greennode/credentials INI (auth-only merge — one identity file per
// profile). The access token is held in memory only — nothing the command
// prints or writes leaks it.
package login

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/config"
	loginpkg "github.com/vngcloud/greennode-cli/internal/login"
)

// The IAM endpoint presets per environment live in loginpkg.IamEndpoints
// (internal/login/iamenv.go), from the PKCE login design spec §7 (dev signin
// host confirmed 2026-07-30). AuthorizeURL is the browser signin page; the prod
// entry 301→signin.greennode.ai (a rebrand, not an env signal). Override either
// endpoint piecewise with --authorize-url / --token-url when a portal points
// elsewhere.
//
// The per-env IAM endpoints (authorize/token/client_id) and the baked-in public
// client_id constants live in internal/login — loginpkg.IamEndpoints,
// loginpkg.DevClientID, loginpkg.ProdClientID, loginpkg.DefaultIamEnv
// — shared with the subcommand refresh-token path so a profile's iam_env
// resolves to the SAME /v2 token URL at login and at refresh. See
// internal/login/iamenv.go for the rationale (public client_ids are non-secret;
// the client_secret is never baked in).

const (
	defaultTimeout = 5 * time.Minute
)

var (
	flagClientID     string
	flagClientSecret string
	flagIamEnv       string
	flagAuthorizeURL string
	flagTokenURL     string
	flagScope        string
	flagTimeout      time.Duration
)

// LoginCmd is the `grn login` command.
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to GreenNode via IAM (interactive PKCE)",
	Long: `Log in to GreenNode by authenticating against VNG IAM with a browser-based
PKCE authorization-code flow. The resulting refresh token is folded into the
per-profile ~/.greennode/credentials INI (0600), alongside any machine
credentials; the access token is held in memory only. Use --profile to target a
specific profile (e.g. dev vs prod) so each holds its own login token.

A default public client_id is baked in per --iam-env (dev's and prod's real
ids), so "grn login" needs no --client-id. Override with --client-id or
GRN_LOGIN_CLIENT_ID. Pick the env with --iam-env or GRN_IAM_ENV (default prod).
The client_secret is never baked in; omit --client-secret for a PKCE-only
public client.`,
	RunE: runLogin,
}

// LogoutCmd is the `grn logout` command.
var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Forget the cached GreenNode login refresh token",
	Long: `Removes the login keys (refresh_token, expiry, auth_mode, login_client_id,
iam_env) from the current profile's credentials section, leaving any machine
client_id/client_secret intact. Idempotent: running logout when not logged in
is not an error.`,
	RunE: runLogout,
}

func init() {
	LoginCmd.Flags().StringVar(&flagClientID, "client-id", "", "IAM OAuth client id (default: baked-in per --iam-env; env: GRN_LOGIN_CLIENT_ID)")
	LoginCmd.Flags().StringVar(&flagClientSecret, "client-secret", "", "IAM OAuth client secret; omit for a PKCE-only public client (env: GRN_LOGIN_CLIENT_SECRET)")
	LoginCmd.Flags().StringVar(&flagIamEnv, "iam-env", "", "IAM environment preset: prod|dev (default prod; env: GRN_IAM_ENV)")
	LoginCmd.Flags().StringVar(&flagAuthorizeURL, "authorize-url", "", "Override the IAM signin/authorize URL")
	LoginCmd.Flags().StringVar(&flagTokenURL, "token-url", "", "Override the IAM /v2 token URL")
	LoginCmd.Flags().StringVar(&flagScope, "scope", "", "OAuth scopes (space-separated); default openid")
	LoginCmd.Flags().DurationVar(&flagTimeout, "timeout", defaultTimeout, "Maximum time to wait for the browser login to complete")
}

// resolveConfig builds a loginpkg.Config from the flag/env inputs plus the
// iam-env preset. Precedence: explicit flag > env var; --authorize-url/
// --token-url > preset; client-id flag > GRN_LOGIN_CLIENT_ID > the per-env
// baked-in default (see resolveClientID). It is pure w.r.t. the cobra flag
// state (callers pass flag values in) so it table-tests cleanly. Returns an
// error only when ClientID resolves empty AND the preset has no baked default
// — unreachable with the shipped presets, kept as a defensive guard.
func resolveConfig(clientID, clientSecret, iamEnv, authorizeURL, tokenURL, scope string, timeout time.Duration) (loginpkg.Config, time.Duration, error) {
	env := iamEnv
	if env == "" {
		env = os.Getenv("GRN_IAM_ENV")
	}
	if env == "" {
		env = loginpkg.DefaultIamEnv
	}
	preset, ok := loginpkg.IamEndpoints[env]
	if !ok {
		return loginpkg.Config{}, 0, fmt.Errorf("invalid --iam-env %q (valid: prod, dev)", env)
	}

	if authorizeURL == "" {
		authorizeURL = preset.Authorize
	}
	if tokenURL == "" {
		tokenURL = preset.Token
	}

	clientID, err := resolveClientID(clientID, preset.ClientID)
	if err != nil {
		return loginpkg.Config{}, 0, err
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("GRN_LOGIN_CLIENT_SECRET")
	}

	var scopes []string
	if s := strings.TrimSpace(scope); s != "" {
		scopes = strings.Fields(s)
	} else {
		scopes = []string{"openid"}
	}

	if timeout <= 0 {
		timeout = defaultTimeout
	}

	cfg := loginpkg.Config{
		AuthorizeURL: authorizeURL,
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       scopes,
	}
	return cfg, timeout, nil
}

// resolveClientID applies client-id precedence: explicit --client-id flag >
// GRN_LOGIN_CLIENT_ID env > the per-iam-env baked-in public default. The baked
// default is a non-secret public client_id (dev's and prod's real ids);
// only when ALL three are empty (a preset with no default) does login refuse.
// Unreachable with the shipped presets — the error is a defensive guard so a
// future preset without a ClientID fails loudly rather than sending an empty
// client_id to IAM.
func resolveClientID(flagClientID, embeddedDefault string) (string, error) {
	if flagClientID != "" {
		return flagClientID, nil
	}
	if v := os.Getenv("GRN_LOGIN_CLIENT_ID"); v != "" {
		return v, nil
	}
	if embeddedDefault != "" {
		return embeddedDefault, nil
	}
	return "", fmt.Errorf("--client-id (or GRN_LOGIN_CLIENT_ID) is required")
}

// resolveIamEnv applies iam-env precedence: a non-empty --iam-env flag >
// GRN_IAM_ENV env > prod default. The flag has no cobra default (registered as
// "") so an unset flag falls through to the env var, letting
// `GRN_IAM_ENV=dev grn login` select dev without --iam-env. Returns the
// EFFECTIVE env so it can be threaded to both resolveConfig and the persisted
// login context (iam_env must reflect where the user actually logged in, not
// the flag default — otherwise a GRN_IAM_ENV=dev login would persist prod).
func resolveIamEnv(cmd *cobra.Command) string {
	if f := cmd.Flag("iam-env"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	if v := os.Getenv("GRN_IAM_ENV"); v != "" {
		return v
	}
	return loginpkg.DefaultIamEnv
}

func runLogin(cmd *cobra.Command, _ []string) error {
	iamEnv := resolveIamEnv(cmd)
	cfg, timeout, err := resolveConfig(flagClientID, flagClientSecret, iamEnv, flagAuthorizeURL, flagTokenURL, flagScope, flagTimeout)
	if err != nil {
		return err
	}
	// The global --debug flag gates the library's stderr trace. Restored on
	// return so a debug run never leaves the seam armed for later invocations.
	if dbg, _ := cmd.Flags().GetBool("debug"); dbg {
		loginpkg.SetDebug(true)
		defer loginpkg.SetDebug(false)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	// The library mints and returns the Token; persistence is our job. Fold the
	// refresh token + non-secret refresh context (login_client_id, iam_env) into
	// the per-profile credentials INI so a later usage slice can refresh without
	// re-prompting. The access token is NEVER persisted — only the refresh token
	// is (0600). On partial success (no refresh_token) there is nothing to
	// persist; the library already warned on stderr.
	tok, err := loginpkg.Login(ctx, cfg)
	if err != nil {
		return err
	}

	profile := resolveProfile(cmd)
	if tok.RefreshToken != "" {
		// Persist the RESOLVED iam-env (flag > GRN_IAM_ENV > prod) — not the raw
		// flag — so refresh targets the environment the user actually logged in
		// against, even when they selected it via GRN_IAM_ENV.
		if err := config.NewConfigFileWriter().WriteLoginToken(profile, tok.RefreshToken, tok.ExpiresAt, "user", cfg.ClientID, iamEnv); err != nil {
			return fmt.Errorf("persist login token: %w", err)
		}
	}

	// Do not echo the access token (credential hygiene). Report only the profile
	// the refresh token landed in + the access-token expiry.
	fmt.Printf("Logged in. Refresh token saved to profile '%s'.", profile)
	if !tok.ExpiresAt.IsZero() {
		fmt.Printf(" Access token expires %s.", tok.ExpiresAt.Format(time.RFC3339))
	}
	if tok.RefreshToken == "" {
		fmt.Printf(" No refresh token returned — re-login will be needed after expiry.")
	}
	fmt.Println()
	return nil
}

// resolveProfile mirrors cmd/configure's resolution: --profile flag → GRN_PROFILE
// → "default". login/logout act on the resolved profile's credentials section
// so dev/prod profiles each hold their own login token. The nil-flag guard lets
// a zero *cobra.Command (white-box tests, no registered flags) resolve to the
// default/env profile instead of panicking.
func resolveProfile(cmd *cobra.Command) string {
	profile := ""
	if f := cmd.Flag("profile"); f != nil {
		profile = f.Value.String()
	}
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}
	return profile
}

func runLogout(cmd *cobra.Command, _ []string) error {
	profile := resolveProfile(cmd)
	if err := config.NewConfigFileWriter().ClearLoginToken(profile); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	fmt.Printf("Logged out (login token cleared from profile '%s').\n", profile)
	return nil
}
