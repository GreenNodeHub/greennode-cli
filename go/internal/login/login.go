package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

// openBrowser is the browser-open seam used by Login's back-compat path. Tests
// replace it with a no-op. The cmd-layer Prepare/Complete flow does NOT use
// this seam — it calls the exported OpenBrowser directly (so tests of the cmd
// control opening without touching this var).
var openBrowser = defaultOpenBrowser

func defaultOpenBrowser(rawurl string) error {
	return browserRunner()(rawurl) // platform-specific, defined in opener.go
}

// OpenBrowser opens rawurl in the user's default browser, best-effort. It is the
// exported form of the platform opener (darwin `open` / windows `start` / linux
// `xdg-open`), exposed so the cmd-layer Prepare/Complete flow can open the
// browser itself without duplicating the platform logic. A non-nil error means
// no opener was available or the launch command failed — the caller should have
// already printed the URL so a manual open is still possible.
func OpenBrowser(rawurl string) error { return defaultOpenBrowser(rawurl) }

// Flow is a prepared PKCE login: a live loopback listener armed with a nonce,
// the PKCE pair, and the resolved config. The browser is NOT opened by Prepare —
// opening is the CALLER's job (the cmd prints the URL, confirms, then opens).
// Call Complete to wait for the callback and exchange the code; call Close
// (usually deferred) to release the loopback port if Complete is never reached
// (e.g. the user cancels at a confirm prompt).
type Flow struct {
	listener *Listener
	pair     Pair
	nonce    string
	cfg      Config
}

// Prepare generates the PKCE pair + nonce, binds the loopback listener, arms the
// state-mismatch check, and returns a *Flow. It does NOT build the authorize URL
// yet and does NOT open a browser — the caller may still override
// cfg.AuthorizeURL (e.g. for SSO strip-and-replace) via SetAuthorizeURL before
// calling AuthURL/Complete. Listener is bound and armed; Complete starts Serve.
func Prepare(ctx context.Context, cfg Config) (*Flow, error) {
	pair, err := Generate()
	if err != nil {
		return nil, fmt.Errorf("login: pkce generate: %w", err)
	}
	nonce, err := randomNonce()
	if err != nil {
		return nil, fmt.Errorf("login: nonce: %w", err)
	}
	l, err := NewListener()
	if err != nil {
		return nil, fmt.Errorf("login: listener: %w", err)
	}
	l.setExpectedState(nonce) // arm the nonce check so a mismatch serves a failure page
	return &Flow{listener: l, pair: pair, nonce: nonce, cfg: cfg}, nil
}

// RedirectURI returns the loopback URI the listener is bound to, so the caller
// can force a user-supplied SSO authorize URL's redirectUri param onto the
// listener (strip-and-replace) before AuthURL is built.
func (f *Flow) RedirectURI() string { return f.listener.RedirectURI() }

// SetAuthorizeURL overrides the authorize endpoint (e.g. a user-supplied SSO
// link after strip-and-replace). Must be called before AuthURL/Complete. A
// clientId present in the supplied URL is adopted into cfg.ClientID so
// BuildAuthorizeURL (which force-sets clientId from cfg.ClientID) preserves it
// rather than stomping it back to the default — that is the strip-and-replace
// the caller relies on. Best-effort: an unparseable URL is stored as-is.
func (f *Flow) SetAuthorizeURL(u string) {
	if pu, perr := url.Parse(u); perr == nil && pu != nil {
		if cid := pu.Query().Get("clientId"); cid != "" {
			f.cfg.ClientID = cid
		}
	}
	f.cfg.AuthorizeURL = u
}

// AuthURL builds the authorize URL from the flow's config, loopback redirect,
// PKCE challenge, and nonce. Rebuilt each call (cheap) so a SetAuthorizeURL
// override takes effect immediately.
func (f *Flow) AuthURL() string {
	return f.cfg.BuildAuthorizeURL(f.listener.RedirectURI(), f.pair.Challenge, f.nonce)
}

// Complete waits for the callback (Serve) and exchanges the code for a token.
// It does NOT open a browser. It emits the same redacted --debug trace Login
// does, and surfaces the same errors (ErrStateMismatch, *ErrAuthzDenied,
// transport/exchange errors, ctx errors). Returns the minted Token; persistence
// is the CALLER's job.
func (f *Flow) Complete(ctx context.Context) (Token, error) {
	authURL := f.AuthURL()
	dbg("endpoints authorize=%s token=%s", f.cfg.AuthorizeURL, f.cfg.TokenURL)
	dbg("client client_id=%s redirect_uri=%s has_basic=%t secret_len=%d", f.cfg.ClientID, f.listener.RedirectURI(), f.cfg.ClientSecret != "", len(f.cfg.ClientSecret))
	dbg("authorize_url %s", authURL) // carries challenge+appState only; the verifier never leaves the token POST

	code, _, serr := f.listener.Serve(ctx)
	if serr != nil {
		return Token{}, serr // ErrStateMismatch / *ErrAuthzDenied / ctx error — listener owns the nonce check
	}
	dbg("callback ok code_len=%d state_len=%d", len(code), len(f.nonce))
	dbg("exchange grant_type=authorization_code client_id=%s redirect_uri=%s code_len=%d verifier_len=%d has_basic=%t",
		f.cfg.ClientID, f.listener.RedirectURI(), len(code), len(f.pair.Verifier), f.cfg.ClientSecret != "")
	tc := New(30 * time.Second)
	resp, errE, err := tc.ExchangeCode(ctx, f.cfg.TokenURL, ExchangeParams{
		Code:         code,
		RedirectURI:  f.listener.RedirectURI(),
		ClientID:     f.cfg.ClientID,
		ClientSecret: f.cfg.ClientSecret,
		CodeVerifier: f.pair.Verifier,
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

	accessToken, err := f.cfg.AccessTokenFrom(resp.Raw)
	if err != nil {
		return Token{}, fmt.Errorf("login: %w", err)
	}
	tok, refreshToken, err := decodeTokenBody(resp.Raw, accessToken)
	if err != nil {
		dbg("decode failed err=%v", err)
		return Token{AccessToken: accessToken, TokenType: "Bearer"}, fmt.Errorf("login: token body decode: %w", err)
	}
	dbg("decoded access_token_len=%d token_type=%s refresh_present=%t expires_at=%s",
		len(tok.AccessToken), tok.TokenType, refreshToken != "", tok.ExpiresAt.Format(time.RFC3339))
	if refreshToken == "" {
		fmt.Fprintf(noisyStderr, "Warning: IAM returned no refresh_token — re-login needed after expiry.\n")
	}
	return tok, nil
}

// Close releases the loopback port. Safe to call multiple times (idempotent).
// Call it (usually deferred) even when the user cancels before Complete.
func (f *Flow) Close() error { return f.listener.Close() }

// Login runs the full PKCE authorization-code flow against IAM and returns the
// minted token — the back-compat wrapper over Prepare/Complete. It DOES open the
// browser best-effort (the original behavior), printing the URL on failure.
// Existing callers (and the login_test.go suite) keep working unchanged. The
// staged Prepare/Complete path is what the `grn login` cmd now uses so it can
// own the UX (show link → confirm → open → spinner → wait).
func Login(ctx context.Context, cfg Config) (Token, error) {
	f, err := Prepare(ctx, cfg)
	if err != nil {
		return Token{}, err
	}
	defer f.Close()
	authURL := f.AuthURL()
	if oerr := openBrowser(authURL); oerr != nil {
		fmt.Fprintf(noisyStderr, "Could not open a browser automatically.\nOpen this URL to log in:\n  %s\n", authURL)
	}
	return f.Complete(ctx)
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

// DecodeTokenBody decodes a /v2 token response (snake_case) into a Token,
// fail-loud validating that an access_token is present (the same guard
// Config.AccessTokenFrom applies, so a silent decode-to-empty is impossible).
// Exported so a refresh-token-grant caller outside this package — which has a
// raw response body but no Config — can decode access_token / token_type /
// refresh_token / expires_in in one call. AccessTokenFrom is a method on Config
// but reads no Config state, so a zero-value Config reaches the exact validation
// path Login uses; the decoded Token.RefreshToken lets the caller detect (and
// persist) a rotated refresh token, and ExpiresAt drives cache expiry.
func DecodeTokenBody(raw json.RawMessage) (Token, error) {
	accessToken, err := Config{}.AccessTokenFrom(raw)
	if err != nil {
		return Token{}, err
	}
	tok, _, err := decodeTokenBody(raw, accessToken)
	return tok, err
}

// randomNonce returns a 256-bit random base64url string for the appState.
func randomNonce() (string, error) {
	p, err := Generate() // reuse the 256-bit base64url generator
	if err != nil {
		return "", err
	}
	return p.Verifier, nil
}
