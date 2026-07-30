package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client posts to an OAuth /token endpoint. A thin wrapper around *http.Client
// with a timeout; no retry. Mirrors agent-core-gateway
// internal/clients/idpoauth/client.go.
type Client struct {
	http *http.Client
}

// New wraps an *http.Client with the supplied timeout.
func New(timeout time.Duration) *Client {
	return &Client{http: &http.Client{Timeout: timeout}}
}

// ExchangeParams collects authorization_code grant inputs.
type ExchangeParams struct {
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string // "" → public client (PKCE-only); non-empty → Basic auth
	CodeVerifier string // PKCE; always sent
}

// RefreshParams collects refresh_token grant inputs.
type RefreshParams struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scope        string
}

// TokenResponse preserves the full raw /token body so the caller can extract
// access_token (via Config.AccessTokenFrom) and refresh_token/expires_in itself.
type TokenResponse struct {
	Raw json.RawMessage
}

// Error captures a non-2xx /token response. RawBody is bounded to 1 MiB.
type Error struct {
	Status  int
	RawBody []byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("login: status=%d body=%s", e.Status, string(e.RawBody))
}

// ExchangeCode posts grant_type=authorization_code + code_verifier (+ Basic
// when ClientSecret != ""). Mirrors idpoauth/client.go:76-87.
func (c *Client) ExchangeCode(ctx context.Context, tokenURL string, p ExchangeParams) (*TokenResponse, *Error, error) {
	v := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {p.Code},
		"redirect_uri": {p.RedirectURI},
		"client_id":    {p.ClientID},
	}
	if p.CodeVerifier != "" {
		v.Set("code_verifier", p.CodeVerifier)
	}
	return c.post(ctx, tokenURL, v, p.ClientID, p.ClientSecret)
}

// Refresh posts grant_type=refresh_token (+ Basic when secret set).
func (c *Client) Refresh(ctx context.Context, tokenURL string, p RefreshParams) (*TokenResponse, *Error, error) {
	v := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {p.RefreshToken},
		"client_id":     {p.ClientID},
	}
	if p.Scope != "" {
		v.Set("scope", p.Scope)
	}
	return c.post(ctx, tokenURL, v, p.ClientID, p.ClientSecret)
}

func (c *Client) post(ctx context.Context, tokenURL string, v url.Values, clientID, clientSecret string) (*TokenResponse, *Error, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	// client_secret_basic per RFC 6749 §2.3.1 — sent on EVERY token POST. VNG
	// IAM's /v2 endpoint requires Basic for all clients, including "public"
	// (no-secret) ones: for those the password is empty and only the client_id
	// (the Basic username) is validated. Verified against dev IAM: a correct
	// PKCE POST without Basic is rejected as AUTHENTICATION_FAILED, while the
	// same POST with Basic(client_id:) advances to grant validation
	// (AUTH_CODE_INVALID for a bogus code). Body secrets are NOT honored by IAM
	// — the credential travels only in this header. Mirrors agent-core-gateway
	// internal/clients/idpoauth/client.go:108-113, extended to the public-client
	// case the CLI needs.
	req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{Status: resp.StatusCode, RawBody: body}, nil
	}
	return &TokenResponse{Raw: json.RawMessage(body)}, nil, nil
}
