// Package client provides the base HTTP client for the GreenNode AgentBase API.
// It handles authentication, JSON serialization, and error mapping.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the authenticated HTTP client for a single API base URL.
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       coreclient.TokenProvider
}

// New creates a new Client for the given base URL and token provider. The
// provider is the shared coreclient.TokenProvider (GetToken/RefreshToken,
// ctx-less) — the same seam vks/vserver use — so agentbase speaks the same auth
// idiom as the rest of the CLI. Pass nil only in construction tests that never
// call Do.
func New(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth: tp,
	}
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// Do executes an authenticated HTTP request and decodes the response into out.
// Pass out=nil if you do not need the response body. The token comes from the
// shared TokenProvider (ctx-less GetToken, matching vks/vserver); ctx still
// drives the HTTP call itself. A nil provider (construction tests) panics on
// Do — never construct a Client with nil for a real request.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out interface{}) error {
	return c.doReq(ctx, method, path, query, nil, body, out)
}

// DoWithHeaders is Do with extra request headers (e.g. If-Match for OCC PUTs).
// headers may be nil. It is additive: Authorization/Content-Type/Accept are
// still applied as in Do, and extra headers never overwrite those three.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, query url.Values, headers map[string]string, body, out interface{}) error {
	return c.doReq(ctx, method, path, query, headers, body, out)
}

// doReq is the single implementation behind Do and DoWithHeaders. Extra headers
// are applied after the standard Auth/Content-Type/Accept set and never
// overwrite those three.
func (c *Client) doReq(ctx context.Context, method, path string, query url.Values, headers map[string]string, body, out interface{}) error {
	token, err := c.auth.GetToken()
	if err != nil {
		return err
	}

	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var data []byte
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := c.buildRequest(ctx, method, fullURL, data, headers, token)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 401 — force-refresh the token and retry once with the new bearer, mirroring
	// internal/client.GreennodeClient so a mid-session access-token expiry (or
	// server-side revocation) self-heals instead of surfacing as a hard APIError.
	// The proactive GetToken above + the provider's pre-expiry skew handle the
	// common case; this is the reactive backstop. A refresh failure, a transport
	// error on the retry, or a second 401 is terminal — no further refresh.
	if resp.StatusCode == http.StatusUnauthorized {
		token, err = c.auth.RefreshToken()
		if err != nil {
			return err
		}
		req2, err := c.buildRequest(ctx, method, fullURL, data, headers, token)
		if err != nil {
			return err
		}
		resp2, err := c.httpClient.Do(req2)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		resp2Body, err := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		resp = resp2
		respBody = resp2Body
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// buildRequest assembles the authenticated *http.Request for a given bearer
// token. Shared by doReq's initial attempt and its 401 retry so the retry
// applies an identical header set — Authorization swapped for the fresh token,
// Content-Type/Accept/extra headers preserved. data is the already-marshaled
// JSON body (nil for bodyless requests); nil data means no Content-Type.
func (c *Client) buildRequest(ctx context.Context, method, fullURL string, data []byte, headers map[string]string, token string) (*http.Request, error) {
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if k == "Authorization" || k == "Content-Type" || k == "Accept" {
			continue
		}
		req.Header.Set(k, v)
	}
	return req, nil
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out interface{}) error {
	return c.Do(ctx, http.MethodGet, path, query, nil, out)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body, out interface{}) error {
	return c.Do(ctx, http.MethodPost, path, nil, body, out)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, query url.Values, body, out interface{}) error {
	return c.Do(ctx, http.MethodPatch, path, query, body, out)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body, out interface{}) error {
	return c.Do(ctx, http.MethodPut, path, nil, body, out)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, out interface{}) error {
	return c.Do(ctx, http.MethodDelete, path, nil, nil, out)
}
