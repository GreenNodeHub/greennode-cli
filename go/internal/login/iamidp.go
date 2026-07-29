package login

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Config is the resolved IAM login configuration, env-specific. All fields are
// sourced by the caller (the future root grn login command) from greennode-cli's
// per-env config; this package does no config loading itself.
type Config struct {
	// AuthorizeURL is IAM's signin page, e.g.
	// https://signin.vngcloud.vn/ap/auth (301 → signin.greennode.ai).
	AuthorizeURL string
	// TokenURL is IAM's OAuth /v2 token endpoint.
	TokenURL string
	// ClientID is the CLI's IAM OAuth client id.
	ClientID string
	// ClientSecret is "" for a public client (PKCE-only) or non-empty for a
	// confidential client (sent via Basic at the token endpoint).
	ClientSecret string
	// Scopes are optional OAuth scopes, e.g. ["openid"].
	Scopes []string
}

// BuildAuthorizeURL renders IAM's camelCase signin URL: clientId,
// responseType=code, codeChallenge, codeChallengeMethod=S256, appState,
// redirectUri (+ scope when Scopes is non-empty). signedState is the opaque
// appState carrier (the CLI's nonce). Mirrors agent-core-gateway
// internal/oauth/iamidp/adapter.go:51-69.
func (c Config) BuildAuthorizeURL(redirectURI, codeChallenge, signedState string) string {
	u, err := url.Parse(c.AuthorizeURL)
	if err != nil || u == nil {
		// AuthorizeURL is assumed validated upstream; fall back to raw so an
		// error is visible rather than silently dropped.
		return c.AuthorizeURL
	}
	q := u.Query()
	q.Set("clientId", c.ClientID)
	q.Set("responseType", "code")
	q.Set("codeChallenge", codeChallenge)
	q.Set("codeChallengeMethod", "S256")
	q.Set("appState", signedState)
	q.Set("redirectUri", redirectURI)
	if scope := c.ScopeString(); scope != "" {
		q.Set("scope", scope)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// ScopeString joins Scopes with a space (OAuth scope syntax), or "" when none.
func (c Config) ScopeString() string { return strings.Join(c.Scopes, " ") }

// AccessTokenFrom extracts access_token from a /v2 token response body,
// returning an error when the body is not JSON or carries no non-empty
// access_token. This is the fail-loud guard against an unexpected response
// shape — a silent decode-to-empty must be impossible. Mirrors
// agent-core-gateway internal/oauth/iamidp/adapter.go:84-95.
func (c Config) AccessTokenFrom(raw []byte) (string, error) {
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", fmt.Errorf("iamidp: token response is not valid JSON: %w", err)
	}
	if strings.TrimSpace(body.AccessToken) == "" {
		return "", fmt.Errorf("iamidp: token response missing access_token (unexpected /v2 shape)")
	}
	return body.AccessToken, nil
}
