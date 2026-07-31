package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/vngcloud/greennode-cli/internal/login"
)

// refreshExpirySkew is how long before the access token's real expiry the
// LoginTokenProvider considers it stale and refreshes — mirrors the machine
// MachineTokenProvider's 60s skew (machine_token.go).
const refreshExpirySkew = 60 * time.Second

// noExpiryFallback is used when a refresh response carries no expires_in: pin a
// conservative 30 min so a long-lived process still re-refreshes rather than
// trusting a non-expiring token indefinitely.
const noExpiryFallback = 30 * time.Minute

// ErrLoginTokenRefreshFailed wraps a refresh-grant failure so the caller can
// surface "run `grn login`" guidance. A refresh fails when the refresh token is
// expired or revoked, or IAM rejected the grant. This is a hard error — the
// login provider does NOT silently fall back to machine credentials (a profile
// commits to one auth type).
var ErrLoginTokenRefreshFailed = errors.New("login token expired or revoked — run `grn login`")

// LoginTokenProvider is the user-PKCE auth source for GreennodeClient: it mints
// short-lived access tokens from the persisted refresh token via the IAM /v2
// refresh_token grant (login.Client.Refresh). It is the login counterpart to the
// machine client_credentials MachineTokenProvider; both satisfy client.TokenProvider
// (structural — this package does not import internal/client, avoiding a cycle,
// since internal/login is stdlib-only and does not import internal/auth).
//
// The access token is held in memory only for the process lifetime (NEVER
// persisted — by design). IAM may rotate the refresh token on refresh; if so
// and persist is set, the new refresh token + expiry are written back to the
// profile (best-effort) so later invocations don't see a stale token. The
// refresh_token grant always sends Basic(client_id, "") for the public/no-secret
// client `grn login` used — the baked-in per-env id the caller resolves from
// iam_env (login.ClientIDForEnv), exactly the public-client shape the authorize
// flow's token POST uses (tokencx.go:104).
type LoginTokenProvider struct {
	refreshToken string
	clientID     string
	clientSecret string // "" for the public/no-secret dev client login persists
	tokenURL     string

	tc      *login.Client
	persist func(refreshToken string, expiresAt time.Time) error // optional; best-effort rotation write

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewLoginTokenProvider builds a user-token provider. persist is an optional
// callback invoked when IAM rotates the refresh token (issues a new one); the
// caller (the client wiring layer) sets it to write the rotated token + new
// expiry back to the credentials INI via config.WriteLoginToken. A nil persist
// means rotation is not persisted (the in-memory access token is still used for
// the rest of this process). clientID is the baked-in public client resolved
// from iam_env by the caller (login.ClientIDForEnv); clientSecret is "" for
// that public client — client_secret is never persisted by `grn login`.
func NewLoginTokenProvider(refreshToken, clientID, clientSecret, tokenURL string, persist func(string, time.Time) error) *LoginTokenProvider {
	return &LoginTokenProvider{
		refreshToken: refreshToken,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		tc:           login.New(30 * time.Second),
		persist:      persist,
	}
}

// GetToken returns a valid access token, refreshing first if the cache is empty
// or within the expiry skew. GreennodeClient calls this once per request.
func (p *LoginTokenProvider) GetToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.expiresAt) {
		return p.accessToken, nil
	}
	return p.refresh()
}

// RefreshToken force-refreshes regardless of cache state. GreennodeClient calls
// this on HTTP 401 to retry once with a fresh token.
func (p *LoginTokenProvider) RefreshToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refresh()
}

// refresh mints a new access token from the refresh_token grant. Caller holds
// p.mu. On a transport error or a non-2xx IAM response it returns
// ErrLoginTokenRefreshFailed (hard-error — no silent fallback). On success it
// caches the access token with a 60s pre-expiry skew. If IAM returned a NEW
// refresh token (rotation), it best-effort persists it via p.persist and updates
// the in-memory token so subsequent refreshes use the new one; a persist failure
// is reported on stderr, not returned (the access token is still valid for this
// invocation).
func (p *LoginTokenProvider) refresh() (string, error) {
	resp, errResp, err := p.tc.Refresh(context.Background(), p.tokenURL, login.RefreshParams{
		RefreshToken: p.refreshToken,
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Scope:        "openid",
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLoginTokenRefreshFailed, err)
	}
	if errResp != nil {
		return "", fmt.Errorf("%w: iam status=%d body=%s", ErrLoginTokenRefreshFailed, errResp.Status, string(errResp.RawBody))
	}
	tok, err := login.DecodeTokenBody(resp.Raw)
	if err != nil {
		return "", fmt.Errorf("login token decode: %w", err)
	}

	p.accessToken = tok.AccessToken
	exp := tok.ExpiresAt
	if exp.IsZero() {
		exp = time.Now().Add(noExpiryFallback)
	}
	p.expiresAt = exp.Add(-refreshExpirySkew)

	// Rotation: IAM may issue a new refresh token. Persist + adopt it so the next
	// refresh (this process or a later invocation) uses the rotated token.
	if tok.RefreshToken != "" && tok.RefreshToken != p.refreshToken {
		if p.persist != nil {
			if perr := p.persist(tok.RefreshToken, exp); perr != nil {
				fmt.Fprintf(os.Stderr, "grn: warning: failed to persist rotated login refresh token: %v\n", perr)
			}
		}
		p.refreshToken = tok.RefreshToken
	}
	return p.accessToken, nil
}
