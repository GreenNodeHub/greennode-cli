// Package client implements authenticated AgentBase requests.
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
	"github.com/greennodehub/greennode-cli/internal/redact"
)

// Client is the authenticated HTTP client for a single API base URL.
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       coreclient.TokenProvider
}

// New requires a token provider for requests.
func New(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
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
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, redact.Value)
}

type displayedError struct {
	message string
	cause   error
}

func (e *displayedError) Error() string { return e.message }
func (e *displayedError) Unwrap() error { return e.cause }

// Do decodes responses when out is non-nil.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out interface{}) error {
	return c.doReq(ctx, method, path, query, nil, body, out, true)
}

// DoWithHeaders preserves reserved authentication and JSON headers.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, query url.Values, headers map[string]string, body, out interface{}) error {
	return c.doReq(ctx, method, path, query, headers, body, out, true)
}

// GetOnce prevents replay of credential or provisioning reads.
func (c *Client) GetOnce(ctx context.Context, path string, query url.Values, out interface{}) error {
	return c.doReq(ctx, http.MethodGet, path, query, nil, nil, out, false)
}

func (c *Client) doReq(ctx context.Context, method, path string, query url.Values, headers map[string]string, body, out interface{}, retryRead bool) error {
	token, err := c.auth.GetToken()
	if err != nil {
		return &displayedError{message: "authentication failed", cause: err}
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
		return &displayedError{message: "request failed; outcome may be unknown", cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &displayedError{message: "failed to read response body", cause: err}
	}

	// Only reads may refresh and replay.
	if retryRead && resp.StatusCode == http.StatusUnauthorized && (method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions) {
		token, err = c.auth.RefreshToken()
		if err != nil {
			return &displayedError{message: "authentication refresh failed", cause: err}
		}
		req2, err := c.buildRequest(ctx, method, fullURL, data, headers, token)
		if err != nil {
			return err
		}
		resp2, err := c.httpClient.Do(req2)
		if err != nil {
			return &displayedError{message: "request failed", cause: err}
		}
		resp2Body, err := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		if err != nil {
			return &displayedError{message: "failed to read response body", cause: err}
		}
		resp = resp2
		respBody = resp2Body
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return &displayedError{message: "failed to decode response", cause: err}
		}
	}

	return nil
}

// buildRequest preserves headers across read retries.
func (c *Client) buildRequest(ctx context.Context, method, fullURL string, data []byte, headers map[string]string, token string) (*http.Request, error) {
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &displayedError{message: "failed to create request", cause: err}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Content-Type") || strings.EqualFold(k, "Accept") {
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
