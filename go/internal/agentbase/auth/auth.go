// Package auth provides OAuth2 client-credentials token acquisition for the
// GreenNode AgentBase platform.
package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Token holds a cached access token with its expiry.
type Token struct {
	AccessToken string
	Expiry      time.Time
}

// IsExpired reports whether the token is expired or about to expire (within 30s).
func (t *Token) IsExpired() bool {
	return time.Now().After(t.Expiry.Add(-30 * time.Second))
}

// Provider fetches and caches an OAuth2 access token.
type Provider struct {
	mu           sync.Mutex
	cached       *Token
	clientID     string
	clientSecret string
	tokenURL     string
}

// NewProvider creates a new token Provider for the given credentials and token URL.
func NewProvider(clientID, clientSecret, tokenURL string) *Provider {
	return &Provider{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
	}
}

// AccessToken returns a valid access token, fetching a new one if needed.
func (p *Provider) AccessToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cached != nil && !p.cached.IsExpired() {
		return p.cached.AccessToken, nil
	}

	cfg := &clientcredentials.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		TokenURL:     p.tokenURL,
	}

	tok, err := cfg.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain access token: %w", err)
	}

	p.cached = &Token{
		AccessToken: tok.AccessToken,
		Expiry:      tok.Expiry,
	}

	return tok.AccessToken, nil
}

// TokenSource returns an oauth2.TokenSource compatible with standard Go OAuth2 libraries.
func (p *Provider) TokenSource(ctx context.Context) oauth2.TokenSource {
	cfg := &clientcredentials.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		TokenURL:     p.tokenURL,
	}
	return cfg.TokenSource(ctx)
}
