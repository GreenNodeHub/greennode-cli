package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vngcloud/greennode-cli/internal/auth"
)

const (
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	defaultTimeout = 30 * time.Second
)

var statusMessages = map[int]string{
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	409: "Conflict",
	429: "Too Many Requests",
	500: "Internal Server Error",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
}

var retryableStatusCodes = map[int]bool{
	500: true, 502: true, 503: true, 504: true,
}

// GreennodeClient is an HTTP client for Greennode APIs with retry and auto token refresh.
type GreennodeClient struct {
	baseURL      string
	tokenManager *auth.TokenManager
	httpClient   *http.Client
	debug        bool
}

// NewGreennodeClient creates a new API client. connectTimeout bounds the TCP
// connect and TLS handshake (the --cli-connect-timeout flag); readTimeout bounds
// the overall request (the --cli-read-timeout flag). A zero readTimeout falls
// back to the default; a zero connectTimeout means no explicit connect bound.
func NewGreennodeClient(baseURL string, tokenManager *auth.TokenManager, connectTimeout, readTimeout time.Duration, verifySSL bool, debug bool) *GreennodeClient {
	if readTimeout == 0 {
		readTimeout = defaultTimeout
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: connectTimeout}).DialContext,
	}
	if connectTimeout > 0 {
		transport.TLSHandshakeTimeout = connectTimeout
	}
	if !verifySSL {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &GreennodeClient{
		baseURL:      baseURL,
		tokenManager: tokenManager,
		httpClient: &http.Client{
			Timeout:   readTimeout,
			Transport: transport,
		},
		debug: debug,
	}
}

// Get performs a GET request.
func (c *GreennodeClient) Get(path string, params map[string]string) (interface{}, error) {
	return c.request("GET", path, params, nil)
}

// Post performs a POST request with a JSON body.
func (c *GreennodeClient) Post(path string, body interface{}) (interface{}, error) {
	return c.request("POST", path, nil, body)
}

// Put performs a PUT request with a JSON body.
func (c *GreennodeClient) Put(path string, body interface{}) (interface{}, error) {
	return c.request("PUT", path, nil, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *GreennodeClient) Patch(path string, body interface{}) (interface{}, error) {
	return c.request("PATCH", path, nil, body)
}

// Delete performs a DELETE request.
func (c *GreennodeClient) Delete(path string, params map[string]string) (interface{}, error) {
	return c.request("DELETE", path, params, nil)
}

// GetRaw performs a GET request and returns the raw response body.
func (c *GreennodeClient) GetRaw(path string, params map[string]string) (string, error) {
	return c.requestRaw("GET", path, params, nil)
}

// GetAllPages fetches all pages and merges items into a single result.
func (c *GreennodeClient) GetAllPages(path string, pageSize int) (map[string]interface{}, error) {
	if pageSize == 0 {
		pageSize = 50
	}

	var allItems []interface{}
	page := 0

	for {
		params := map[string]string{
			"page":     fmt.Sprintf("%d", page),
			"pageSize": fmt.Sprintf("%d", pageSize),
		}
		result, err := c.Get(path, params)
		if err != nil {
			return nil, err
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			break
		}

		items, _ := resultMap["items"].([]interface{})
		allItems = append(allItems, items...)

		total, _ := resultMap["total"].(float64)
		if len(allItems) >= int(total) || len(items) == 0 {
			break
		}
		page++
	}

	return map[string]interface{}{
		"items": allItems,
		"total": float64(len(allItems)),
	}, nil
}

func (c *GreennodeClient) request(method, path string, params map[string]string, body interface{}) (interface{}, error) {
	rawBody, err := c.requestRaw(method, path, params, body)
	if err != nil {
		return nil, err
	}

	if rawBody == "" {
		return map[string]interface{}{}, nil
	}

	var result interface{}
	if err := json.Unmarshal([]byte(rawBody), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}
	return result, nil
}

func (c *GreennodeClient) requestRaw(method, path string, params map[string]string, body interface{}) (string, error) {
	fullURL := c.baseURL + path

	if len(params) > 0 {
		u, _ := url.Parse(fullURL)
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	token, err := c.tokenManager.GetToken()
	if err != nil {
		return "", err
	}

	var lastErr error

	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequest(method, fullURL, reqBody)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		if c.debug {
			fmt.Fprintf(os.Stderr, "[debug] %s %s\n", method, fullURL)
			if jsonBody != nil {
				fmt.Fprintf(os.Stderr, "[debug] request body: %s\n", redactDebugBody(string(jsonBody)))
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				delay := retryBaseDelay * time.Duration(1<<uint(attempt))
				time.Sleep(delay)
				continue
			}
			return "", fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if c.debug {
			fmt.Fprintf(os.Stderr, "[debug] response %d: %s\n", resp.StatusCode, redactDebugBody(string(respBody)))
		}

		// 401 — refresh token and retry once
		if resp.StatusCode == http.StatusUnauthorized {
			token, err = c.tokenManager.RefreshToken()
			if err != nil {
				return "", err
			}
			// Retry with new token; reqBody is exhausted so reset from jsonBody.
			var retryBody io.Reader
			if jsonBody != nil {
				retryBody = bytes.NewReader(jsonBody)
			}
			req2, _ := http.NewRequest(method, fullURL, retryBody)
			req2.Header.Set("Authorization", "Bearer "+token)
			req2.Header.Set("Content-Type", "application/json")
			resp2, err := c.httpClient.Do(req2)
			if err != nil {
				return "", err
			}
			respBody, _ = io.ReadAll(resp2.Body)
			resp2.Body.Close()
			resp = resp2
		}

		// Retryable server errors (5xx)
		if retryableStatusCodes[resp.StatusCode] {
			if attempt < maxRetries {
				delay := retryBaseDelay * time.Duration(1<<uint(attempt))
				time.Sleep(delay)
				continue
			}
		}

		// Non-retryable errors
		if resp.StatusCode >= 400 {
			return "", &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
				message:    formatError(resp.StatusCode, respBody),
			}
		}

		return string(respBody), nil
	}

	return "", fmt.Errorf("request failed after %d attempts", maxRetries+1)
}

// APIError is returned for non-retryable HTTP errors (status >= 400). It carries
// the status code and raw body so callers (e.g. waiters) can branch on them,
// while Error() preserves the human-readable formatted message.
type APIError struct {
	StatusCode int
	Body       string
	message    string
}

func (e *APIError) Error() string { return e.message }

func formatError(statusCode int, body []byte) string {
	statusText := statusMessages[statusCode]
	if statusText == "" {
		statusText = "Error"
	}

	detail := ""
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		if msg, ok := data["message"].(string); ok && msg != "" {
			detail = msg
		} else if errMsg, ok := data["error"].(string); ok && errMsg != "" {
			detail = errMsg
		} else if detailMsg, ok := data["detail"].(string); ok && detailMsg != "" {
			detail = detailMsg
		} else if errors, ok := data["errors"].([]interface{}); ok && len(errors) > 0 {
			if errObj, ok := errors[0].(map[string]interface{}); ok {
				if msg, ok := errObj["message"].(string); ok {
					detail = msg
				}
			}
		}
		// Fallback: the error payload didn't use a known string field (e.g. the
		// VKS API returns {"error": {...}} as a nested object). Surface the raw
		// JSON body so the server's message isn't silently dropped.
		if detail == "" && len(body) > 0 {
			detail = strings.TrimSpace(string(body))
		}
	} else {
		detail = string(body)
	}

	if detail != "" {
		return fmt.Sprintf("API error (HTTP %d %s): %s", statusCode, statusText, detail)
	}
	return fmt.Sprintf("API error (HTTP %d %s)", statusCode, statusText)
}

func (c *GreennodeClient) DeleteWithBody(path string, body interface{}) (interface{}, error) {
	return c.request("DELETE", path, nil, body)
}
