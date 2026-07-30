package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrStateMismatch is returned by Login when the callback's appState does not
// equal the nonce Login generated (CSRF / stale-callback guard).
var ErrStateMismatch = errors.New("login: state mismatch")

// debugMode gates an optional stderr trace (enabled by the cobra layer via
// SetDebug when the global --debug flag is set). Off by default so normal runs
// stay quiet. The trace redacts every secret-shaped field (code, PKCE verifier,
// client_secret, access_token) to a LENGTH only; only non-secret context
// (endpoints, client_id, redirect_uri, has_basic) is printed in full.
var debugMode bool

// SetDebug enables/disables the stderr debug trace.
func SetDebug(b bool) { debugMode = b }

// dbg writes a "login debug:" line to noisyStderr when debugMode is on.
func dbg(format string, args ...any) {
	if debugMode {
		fmt.Fprintf(noisyStderr, "login debug: "+format+"\n", args...)
	}
}

// truncBody caps a response body for the debug trace so a large non-2xx body is
// never dumped in full (the transport already bounds it to 1 MiB; this is a
// further 512 B cap for the human-facing trace).
func truncBody(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "…(truncated)"
}

// Token is the full in-memory result of Login: the live access token plus the
// refresh fields. Login mints and returns it; persisting the refresh token is
// the CALLER's job (the library is stdlib-only and profile/INI-agnostic). The
// access token is caller-held and ephemeral — it is never persisted.
type Token struct {
	AccessToken  string
	TokenType    string // e.g. "Bearer"; from the /v2 body, defaulted if empty
	RefreshToken string // empty if IAM returned none (partial-success)
	ExpiresAt    time.Time
}

// openBrowser is the browser-open seam. Tests replace it with a no-op. The
// default tries os/exec openers and prints the URL on failure (see below).
var openBrowser = defaultOpenBrowser

func defaultOpenBrowser(rawurl string) error {
	return browserRunner()(rawurl) // platform-specific, defined in opener.go
}

// Login runs the full PKCE authorization-code flow against IAM and returns the
// minted token. It does NOT persist anything — the caller decides how/where to
// store the refresh token. ctx bounds the whole browser wait.
func Login(ctx context.Context, cfg Config) (Token, error) {
	pair, err := Generate()
	if err != nil {
		return Token{}, fmt.Errorf("login: pkce generate: %w", err)
	}
	nonce, err := randomNonce()
	if err != nil {
		return Token{}, fmt.Errorf("login: nonce: %w", err)
	}

	l, err := NewListener()
	if err != nil {
		return Token{}, fmt.Errorf("login: listener: %w", err)
	}
	defer l.Close()
	l.setExpectedState(nonce) // arm the nonce check so a mismatch serves a failure page

	authURL := cfg.BuildAuthorizeURL(l.RedirectURI(), pair.Challenge, nonce)
	// Trace the resolved config BEFORE opening the browser so a stale-binary /
	// wrong-env mistake is visible immediately. client_id and redirect_uri are
	// public; the secret is reported as presence+length only, never its value.
	dbg("endpoints authorize=%s token=%s", cfg.AuthorizeURL, cfg.TokenURL)
	dbg("client client_id=%s redirect_uri=%s has_basic=%t secret_len=%d", cfg.ClientID, l.RedirectURI(), cfg.ClientSecret != "", len(cfg.ClientSecret))
	dbg("authorize_url %s", authURL) // carries challenge+appState only; the verifier never leaves the token POST

	// Open the browser best-effort; on failure print the URL and keep waiting
	// (headless/SSH users can complete the flow manually).
	if oerr := openBrowser(authURL); oerr != nil {
		fmt.Fprintf(noisyStderr, "Could not open a browser automatically.\nOpen this URL to log in:\n  %s\n", authURL)
	}

	code, _, serr := l.Serve(ctx)
	if serr != nil {
		return Token{}, serr // ErrStateMismatch / *ErrAuthzDenied / ctx error — listener owns the nonce check
	}

	// Trace the exact grant the POST will carry. code and verifier are
	// secret-shaped (one is a bearer grant, the other the PKCE proof) — lengths
	// only, never the values.
	dbg("callback ok code_len=%d state_len=%d", len(code), len(nonce))
	dbg("exchange grant_type=authorization_code client_id=%s redirect_uri=%s code_len=%d verifier_len=%d has_basic=%t",
		cfg.ClientID, l.RedirectURI(), len(code), len(pair.Verifier), cfg.ClientSecret != "")
	tc := New(30 * time.Second)
	resp, errE, err := tc.ExchangeCode(ctx, cfg.TokenURL, ExchangeParams{
		Code:         code,
		RedirectURI:  l.RedirectURI(),
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		CodeVerifier: pair.Verifier,
	})
	if err != nil {
		dbg("exchange transport_error=%v", err)
		return Token{}, fmt.Errorf("login: token exchange: %w", err)
	}
	if errE != nil {
		dbg("exchange non_2xx status=%d body=%s", errE.Status, truncBody(errE.RawBody))
		return Token{}, fmt.Errorf("login: token exchange: %w", errE)
	}
	dbg("exchange http_ok status=2xx body_len=%d", len(resp.Raw))

	accessToken, err := cfg.AccessTokenFrom(resp.Raw)
	if err != nil {
		return Token{}, fmt.Errorf("login: %w", err)
	}

	tok, refreshToken, err := decodeTokenBody(resp.Raw, accessToken)
	if err != nil {
		// access_token parsed fine but refresh/expires decode failed — partial.
		dbg("decode failed err=%v", err)
		return Token{AccessToken: accessToken, TokenType: "Bearer"}, fmt.Errorf("login: token body decode: %w", err)
	}
	dbg("decoded access_token_len=%d token_type=%s refresh_present=%t expires_at=%s",
		len(tok.AccessToken), tok.TokenType, refreshToken != "", tok.ExpiresAt.Format(time.RFC3339))

	if refreshToken == "" {
		// Partial success: access token is valid but IAM returned no refresh
		// token. Warn the caller; persistence is the caller's call regardless.
		fmt.Fprintf(noisyStderr, "Warning: IAM returned no refresh_token — re-login needed after expiry.\n")
	}
	return tok, nil
}

// tokenBody is the subset of the /v2 response Login decodes (snake_case).
type tokenBody struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func decodeTokenBody(raw json.RawMessage, accessToken string) (Token, string, error) {
	var b tokenBody
	if err := json.Unmarshal(raw, &b); err != nil {
		return Token{}, "", fmt.Errorf("decode token body: %w", err)
	}
	tok := Token{
		AccessToken:  accessToken, // already fail-loud-validated
		TokenType:    b.TokenType,
		RefreshToken: b.RefreshToken,
	}
	if b.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	if b.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(b.ExpiresIn) * time.Second).UTC()
	}
	return tok, b.RefreshToken, nil
}

// randomNonce returns a 256-bit random base64url string for the appState.
func randomNonce() (string, error) {
	p, err := Generate() // reuse the 256-bit base64url generator
	if err != nil {
		return "", err
	}
	return p.Verifier, nil
}
