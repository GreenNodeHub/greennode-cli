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

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
)

// Client is the authenticated HTTP client for a single API base URL.
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       *auth.Provider
}

// New creates a new Client for the given base URL and auth provider.
func New(baseURL string, authProvider *auth.Provider) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth: authProvider,
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
// Pass out=nil if you do not need the response body.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out interface{}) error {
	token, err := c.auth.AccessToken(ctx)
	if err != nil {
		return err
	}

	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
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
