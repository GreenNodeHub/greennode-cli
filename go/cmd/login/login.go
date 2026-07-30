// Package login wires the `grn login` / `grn logout` cobra commands around the
// internal/login PKCE library. It resolves a login.Config from flags + env
// (client identity is never baked into the binary), runs the browser PKCE flow,
// and folds the refresh token (0600) + non-secret refresh context into the
// per-profile ~/.greennode/credentials INI (auth-only merge — one identity file
// per profile). The access token is held in memory only — nothing the command
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

// iamEndpoints are the IAM endpoint presets per environment, from the PKCE
// login design spec §7 (dev signin host confirmed 2026-07-30). AuthorizeURL is
// the browser signin page; the prod entry 301→signin.greennode.ai (a rebrand,
// not an env signal). Override either endpoint piecewise with --authorize-url /
// --token-url when a portal points elsewhere.
var iamEndpoints = map[string]struct{ Authorize, Token string }{
	"prod": {
		Authorize: "https://signin.vngcloud.vn/ap/auth",
		Token:     "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token",
	},
	"dev": {
		Authorize: "https://dev-signin.vngcloud.tech/ap/auth",
		Token:     "https://pub-iamapis.api-dev.vngcloud.tech/accounts-api/v2/auth/token",
	},
}

const (
	defaultIamEnv  = "prod"
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

Client identity is supplied via --client-id/--client-secret (or the
GRN_LOGIN_CLIENT_ID/GRN_LOGIN_CLIENT_SECRET env vars) and is never baked into
the binary. Omit --client-secret for a PKCE-only public client.`,
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
	LoginCmd.Flags().StringVar(&flagClientID, "client-id", "", "IAM OAuth client id (env: GRN_LOGIN_CLIENT_ID)")
	LoginCmd.Flags().StringVar(&flagClientSecret, "client-secret", "", "IAM OAuth client secret; omit for a PKCE-only public client (env: GRN_LOGIN_CLIENT_SECRET)")
	LoginCmd.Flags().StringVar(&flagIamEnv, "iam-env", defaultIamEnv, "IAM environment preset: prod|dev")
	LoginCmd.Flags().StringVar(&flagAuthorizeURL, "authorize-url", "", "Override the IAM signin/authorize URL")
	LoginCmd.Flags().StringVar(&flagTokenURL, "token-url", "", "Override the IAM /v2 token URL")
	LoginCmd.Flags().StringVar(&flagScope, "scope", "", "OAuth scopes (space-separated); default openid")
	LoginCmd.Flags().DurationVar(&flagTimeout, "timeout", defaultTimeout, "Maximum time to wait for the browser login to complete")
}

// resolveConfig builds a loginpkg.Config from the flag/env inputs plus the
// iam-env preset. Precedence: explicit flag > env var (for client identity);
// --authorize-url/--token-url > preset. It is pure w.r.t. the cobra flag state
// (callers pass flag values in) so it table-tests cleanly. Returns an error
// when ClientID resolves empty — login cannot proceed without a client.
func resolveConfig(clientID, clientSecret, iamEnv, authorizeURL, tokenURL, scope string, timeout time.Duration) (loginpkg.Config, time.Duration, error) {
	env := iamEnv
	if env == "" {
		env = defaultIamEnv
	}
	preset, ok := iamEndpoints[env]
	if !ok {
		return loginpkg.Config{}, 0, fmt.Errorf("invalid --iam-env %q (valid: prod, dev)", env)
	}

	if authorizeURL == "" {
		authorizeURL = preset.Authorize
	}
	if tokenURL == "" {
		tokenURL = preset.Token
	}

	if clientID == "" {
		clientID = os.Getenv("GRN_LOGIN_CLIENT_ID")
	}
	if clientID == "" {
		return loginpkg.Config{}, 0, fmt.Errorf("--client-id (or GRN_LOGIN_CLIENT_ID) is required")
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

func runLogin(cmd *cobra.Command, _ []string) error {
	cfg, timeout, err := resolveConfig(flagClientID, flagClientSecret, flagIamEnv, flagAuthorizeURL, flagTokenURL, flagScope, flagTimeout)
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
		iamEnv := flagIamEnv
		if iamEnv == "" {
			iamEnv = defaultIamEnv // flag default is "prod"; guard an explicit --iam-env ""
		}
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
