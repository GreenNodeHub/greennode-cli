package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

// MachineTokenProvider mints short-lived access tokens via the IAM v2
// client_credentials grant (RFC 6749) using golang.org/x/oauth2/clientcredentials.
// It is the machine counterpart to LoginTokenProvider and satisfies
// internal/client.TokenProvider (GetToken/RefreshToken, ctx-less — the fetch uses
// context.Background, matching LoginTokenProvider and the former v1 TokenManager).
//
// v2 is the RFC 6749 facade over the same IAM backend as v1: the same
// client_id/client_secret registry and the same token authority, so vks/vserver's
// existing machine creds are valid at v2 and their backends accept v2-minted
// tokens. The tokenURL is resolved by the caller from the profile's iam_env
// (default prod) so machine mode is env-aware, mirroring the user refresh path.
//
// The access token is held in memory only for the process lifetime (NEVER
// persisted — by design). IAM does not issue a refresh token for the
// client_credentials grant, so there is no rotation/persist path here (unlike the
// user LoginTokenProvider, which rotates and persists the refresh token).
type MachineTokenProvider struct {
	cfg clientcredentials.Config

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewMachineTokenProvider builds a machine client_credentials provider against
// the given IAM v2 tokenURL. tokenURL is resolved by the caller from the
// profile's iam_env (default prod) via internal/login.TokenURLForEnv.
func NewMachineTokenProvider(clientID, clientSecret, tokenURL string) *MachineTokenProvider {
	return &MachineTokenProvider{
		cfg: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
		},
	}
}

// GetToken returns a valid access token, re-minting if the cache is empty or
// within the expiry skew. GreennodeClient calls this once per request.
func (p *MachineTokenProvider) GetToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.expiresAt) {
		return p.accessToken, nil
	}
	return p.mint()
}

// RefreshToken force-mints regardless of cache state. GreennodeClient calls this
// on HTTP 401 to retry once with a fresh token.
func (p *MachineTokenProvider) RefreshToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mint()
}

// SetToken pre-seeds the provider with a static token and expiry.
// Intended for use in tests only (mirrors the former TokenManager.SetToken):
// it lets a GreennodeClient fixture return a captive Bearer without hitting IAM.
func (p *MachineTokenProvider) SetToken(token string, expiresAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accessToken = token
	p.expiresAt = expiresAt
}

// mint fetches a new access token via the client_credentials grant. Caller holds p.mu.
func (p *MachineTokenProvider) mint() (string, error) {
	tok, err := p.cfg.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("IAM machine token request failed: %w", err)
	}
	p.accessToken = tok.AccessToken
	exp := tok.Expiry
	if exp.IsZero() {
		exp = time.Now().Add(noExpiryFallback)
	}
	p.expiresAt = exp.Add(-refreshExpirySkew)
	return p.accessToken, nil
}

// Compile-time assertion that MachineTokenProvider satisfies the consumer-side
// internal/client.TokenProvider contract (GetToken/RefreshToken). Kept local as
// an anonymous interface so internal/auth need not import internal/client.
var _ interface {
	GetToken() (string, error)
	RefreshToken() (string, error)
} = (*MachineTokenProvider)(nil)
