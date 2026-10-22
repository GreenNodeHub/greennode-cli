package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserAgent identifies IAM token requests.
var UserAgent = "grn-cli"

// Client posts token grants without retries or redirects.
type Client struct {
	http *http.Client
}

// New wraps an *http.Client with the supplied timeout.
func New(timeout time.Duration) *Client {
	return &Client{http: &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
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
	return fmt.Sprintf("login: status=%d body=[REDACTED]", e.Status)
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
		return nil, nil, tokenRequestError(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	// IAM requires Basic, including public clients with an empty secret.
	req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, tokenRequestError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, nil, tokenRequestError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{Status: resp.StatusCode, RawBody: body}, nil
	}
	return &TokenResponse{Raw: json.RawMessage(body)}, nil, nil
}

func tokenRequestError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errors.New("login: token request failed")
}
