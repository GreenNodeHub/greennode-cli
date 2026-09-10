package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/greennodehub/greennode-cli/internal/login"
)

// refreshExpirySkew refreshes tokens before expiry.
const refreshExpirySkew = 60 * time.Second

// noExpiryFallback bounds tokens lacking expires_in.
const noExpiryFallback = 30 * time.Minute

// ErrLoginTokenRefreshFailed requires login; never fall back to machine auth.
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
	baseCtx      context.Context

	tc      *login.Client
	persist func(refreshToken string, expiresAt time.Time) error // optional; best-effort rotation write

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewLoginTokenProvider uses a public client ID and optional rotation persistence.
func NewLoginTokenProvider(refreshToken, clientID, clientSecret, tokenURL string, persist func(string, time.Time) error) *LoginTokenProvider {
	return &LoginTokenProvider{
		refreshToken: refreshToken,
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		baseCtx:      context.Background(),
		tc:           login.New(30 * time.Second),
		persist:      persist,
	}
}

// GetToken returns a cached token or refreshes it before expiry.
func (p *LoginTokenProvider) GetToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.expiresAt) {
		return p.accessToken, nil
	}
	return p.refresh()
}

// RefreshToken refreshes regardless of cache state.
func (p *LoginTokenProvider) RefreshToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refresh()
}

// refresh requires p.mu; persistence failures warn without invalidating access.
func (p *LoginTokenProvider) refresh() (string, error) {
	resp, errResp, err := p.tc.Refresh(p.baseCtx, p.tokenURL, login.RefreshParams{
		RefreshToken: p.refreshToken,
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Scope:        "openid",
	})
	if err != nil || errResp != nil {
		if p.baseCtx.Err() != nil {
			return "", p.baseCtx.Err()
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		// Return login guidance without exposing the IAM response.
		return "", ErrLoginTokenRefreshFailed
	}
	tok, err := login.DecodeTokenBody(resp.Raw)
	if err != nil {
		return "", errors.New("login token response is invalid")
	}

	p.accessToken = tok.AccessToken
	exp := tok.ExpiresAt
	if exp.IsZero() {
		exp = time.Now().Add(noExpiryFallback)
	}
	p.expiresAt = exp.Add(-refreshExpirySkew)

	// Adopt and persist rotated refresh tokens.
	if tok.RefreshToken != "" && tok.RefreshToken != p.refreshToken {
		if p.persist != nil {
			if perr := p.persist(tok.RefreshToken, exp); perr != nil {
				fmt.Fprintln(os.Stderr, "grn: warning: failed to persist rotated login refresh token; the next command may require login")
			}
		}
		p.refreshToken = tok.RefreshToken
	}
	return p.accessToken, nil
}

// SetBaseContext supplies cancellation for subsequent IAM requests.
func (p *LoginTokenProvider) SetBaseContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	p.baseCtx = ctx
}

// SetHTTPTimeout sets the total IAM request timeout; zero disables it.
func (p *LoginTokenProvider) SetHTTPTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tc = login.New(timeout)
}
