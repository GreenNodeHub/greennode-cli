package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/greennodehub/greennode-cli/internal/redact"
)

const (
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	defaultTimeout = 30 * time.Second

	// maxAttempts includes the initial request, shared transient retries, and one token refresh.
	maxAttempts = maxRetries + 2

	// retryAfterCap limits server-directed waits.
	retryAfterCap = 30 * time.Second
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

// retryableStatusCodes use backoff; HTTP 429 separately honors Retry-After.
var retryableStatusCodes = map[int]bool{
	500: true, 502: true, 503: true, 504: true,
}

// UserAgent identifies API requests; cmd sets the versioned value.
var UserAgent = "grn-cli"

// GreennodeClient is an HTTP client for Greennode APIs with retry and auto token refresh.
type GreennodeClient struct {
	baseURL       string
	tokenProvider TokenProvider
	httpClient    *http.Client
	headers       http.Header
	debug         bool
	sleep         func(time.Duration)

	// baseCtx applies to one invocation. Set it before requests; setup is not concurrency-safe.
	baseCtx context.Context

	// randFloat overrides jitter for tests; it must return [0, 1). Nil uses rand.Float64.
	randFloat func() float64

	// maskedPathValues holds secret path values; configure before requests.
	maskedPathValues []string
}

// TokenProvider accepts machine and user-login token providers.
type TokenProvider interface {
	GetToken() (string, error)
	RefreshToken() (string, error)
}

// HTTPResponse retains status and empty-body metadata alongside decoded data.
type HTTPResponse struct {
	StatusCode int
	Data       any
	Empty      bool
}

// BytesResponse retains raw bytes, status, and media type.
type BytesResponse struct {
	StatusCode  int
	Data        []byte
	ContentType string
}

type requestOptions struct {
	allowRetries        bool
	redactDebugResponse bool
}

var (
	retryingRequestOptions         = requestOptions{allowRetries: true}
	noRetryRequestOptions          = requestOptions{}
	noRetrySensitiveRequestOptions = requestOptions{redactDebugResponse: true}
)

// NewGreennodeClient sets TCP/TLS and total request timeouts.
// Zero read timeout uses the default; zero connect timeout has no explicit bound.
func NewGreennodeClient(baseURL string, tokenProvider TokenProvider, connectTimeout, readTimeout time.Duration, verifySSL bool, debug bool) *GreennodeClient {
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
		baseURL:       baseURL,
		tokenProvider: tokenProvider,
		headers:       make(http.Header),
		httpClient: &http.Client{
			Timeout:       readTimeout,
			Transport:     transport,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		debug:   debug,
		sleep:   time.Sleep,
		baseCtx: context.Background(),
	}
}

// SetBaseContext supplies cancellation for requests and retry waits; nil uses Background.
func (c *GreennodeClient) SetBaseContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.baseCtx = ctx
}

func (c *GreennodeClient) context() context.Context {
	if c.baseCtx == nil {
		return context.Background()
	}
	return c.baseCtx
}

// SetHeader adds service headers but cannot override Authorization or Content-Type.
func (c *GreennodeClient) SetHeader(name, value string) {
	if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Content-Type") {
		return
	}
	if value == "" {
		c.headers.Del(name)
		return
	}
	c.headers.Set(name, value)
}

// Get performs a GET request.
func (c *GreennodeClient) Get(path string, params map[string]string) (any, error) {
	return c.request("GET", path, params, nil)
}

// Post performs a POST request with a JSON body.
func (c *GreennodeClient) Post(path string, body any) (any, error) {
	return c.request("POST", path, nil, body)
}

// PostWithQuery sends JSON with query parameters and never retries.
func (c *GreennodeClient) PostWithQuery(path string, params map[string]string, body any) (any, error) {
	return c.request("POST", path, params, body)
}

// Put performs a PUT request with a JSON body.
func (c *GreennodeClient) Put(path string, body any) (any, error) {
	return c.request("PUT", path, nil, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *GreennodeClient) Patch(path string, body any) (any, error) {
	return c.request("PATCH", path, nil, body)
}

// Delete performs a DELETE request.
func (c *GreennodeClient) Delete(path string, params map[string]string) (any, error) {
	return c.request("DELETE", path, params, nil)
}

// DeleteWithBody performs a DELETE request with the documented JSON body.
func (c *GreennodeClient) DeleteWithBody(path string, body any) (any, error) {
	return c.request("DELETE", path, nil, body)
}

// GetRaw returns the raw GET response body.
func (c *GreennodeClient) GetRaw(path string, params map[string]string) (string, error) {
	rawBody, _, err := c.requestRaw("GET", path, params, nil)
	return rawBody, err
}

// RequestWithStatus retains successful status and decoded response metadata.
func (c *GreennodeClient) RequestWithStatus(method, path string, params map[string]string, body any) (HTTPResponse, error) {
	return c.requestWithStatus(method, path, params, body, retryingRequestOptions)
}

// RequestWithStatusNoRetry prevents replay, including mutations modeled as GET.
func (c *GreennodeClient) RequestWithStatusNoRetry(method, path string, params map[string]string, body any) (HTTPResponse, error) {
	return c.requestWithStatus(method, path, params, body, noRetryRequestOptions)
}

// RequestWithStatusNoRetrySensitive also suppresses response bodies in debug output and displayed errors.
func (c *GreennodeClient) RequestWithStatusNoRetrySensitive(method, path string, params map[string]string, body any) (HTTPResponse, error) {
	return c.requestWithStatus(method, path, params, body, noRetrySensitiveRequestOptions)
}

// RequestWithStatusNoRetryRaw accepts JSON strings or plain text without retrying.
func (c *GreennodeClient) RequestWithStatusNoRetryRaw(method, path string, params map[string]string, body any) (HTTPResponse, error) {
	return c.requestWithStatusRaw(method, path, params, body, noRetryRequestOptions)
}

// RequestWithStatusNoRetrySensitiveRaw also suppresses response bodies in debug output and displayed errors.
func (c *GreennodeClient) RequestWithStatusNoRetrySensitiveRaw(method, path string, params map[string]string, body any) (HTTPResponse, error) {
	return c.requestWithStatusRaw(method, path, params, body, noRetrySensitiveRequestOptions)
}

func (c *GreennodeClient) requestWithStatus(method, path string, params map[string]string, body any, opts requestOptions) (HTTPResponse, error) {
	rawBody, statusCode, err := c.requestRawWithOptions(method, path, params, body, opts)
	if err != nil {
		return HTTPResponse{}, err
	}

	response := HTTPResponse{StatusCode: statusCode, Empty: strings.TrimSpace(rawBody) == ""}
	if response.Empty {
		response.Data = map[string]any{}
		return response, nil
	}

	if err := json.Unmarshal([]byte(rawBody), &response.Data); err != nil {
		return HTTPResponse{}, fmt.Errorf("failed to parse response JSON: %w", err)
	}
	return response, nil
}

func (c *GreennodeClient) requestWithStatusRaw(method, path string, params map[string]string, body any, opts requestOptions) (HTTPResponse, error) {
	rawBody, statusCode, err := c.requestRawWithOptions(method, path, params, body, opts)
	if err != nil {
		return HTTPResponse{}, err
	}

	response := HTTPResponse{StatusCode: statusCode, Empty: strings.TrimSpace(rawBody) == ""}
	if response.Empty {
		response.Data = map[string]any{}
		return response, nil
	}

	var decoded string
	if err := json.Unmarshal([]byte(rawBody), &decoded); err == nil {
		response.Data = decoded
	} else {
		response.Data = rawBody
	}
	return response, nil
}

// RequestBytes sends an encoded body and returns raw bytes. Writes are never retried.
func (c *GreennodeClient) RequestBytes(method, path string, params map[string]string, body []byte, contentType string) (BytesResponse, error) {
	if strings.TrimSpace(contentType) == "" {
		return BytesResponse{}, errors.New("request content type must not be empty")
	}
	return c.requestBytes(method, path, params, body, contentType)
}

// RequestStream sends a streaming body without retries; readers may not be replayable.
func (c *GreennodeClient) RequestStream(method, path string, params map[string]string, body io.Reader, contentType string) (BytesResponse, error) {
	if strings.TrimSpace(contentType) == "" {
		return BytesResponse{}, errors.New("request content type must not be empty")
	}
	return c.requestStream(method, path, params, body, contentType)
}

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

func (c *GreennodeClient) request(method, path string, params map[string]string, body any) (any, error) {
	response, err := c.RequestWithStatus(method, path, params, body)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *GreennodeClient) requestRaw(method, path string, params map[string]string, body any) (string, int, error) {
	return c.requestRawWithOptions(method, path, params, body, retryingRequestOptions)
}

func (c *GreennodeClient) requestRawWithOptions(method, path string, params map[string]string, body any, opts requestOptions) (string, int, error) {
	var jsonBody []byte
	var err error
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return "", 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}
	response, err := c.requestBytesWithOptions(method, path, params, jsonBody, "application/json", opts)
	if err != nil {
		return "", 0, err
	}
	return string(response.Data), response.StatusCode, nil
}

func (c *GreennodeClient) requestBytes(method, path string, params map[string]string, body []byte, contentType string) (BytesResponse, error) {
	return c.requestBytesWithOptions(method, path, params, body, contentType, retryingRequestOptions)
}

func (c *GreennodeClient) requestBytesWithOptions(method, path string, params map[string]string, body []byte, contentType string, opts requestOptions) (BytesResponse, error) {
	return c.requestReaderWithOptions(method, path, params, func() io.Reader {
		if body == nil {
			return nil
		}
		return bytes.NewReader(body)
	}, contentType, opts, body)
}

func (c *GreennodeClient) requestStream(method, path string, params map[string]string, body io.Reader, contentType string) (BytesResponse, error) {
	return c.requestReaderWithOptions(method, path, params, func() io.Reader { return body }, contentType, noRetryRequestOptions, nil)
}

// requestReaderWithOptions handles transport, read-only retries, and response logging.
func (c *GreennodeClient) requestReaderWithOptions(method, path string, params map[string]string, newBody func() io.Reader, contentType string, opts requestOptions, debugBody []byte) (BytesResponse, error) {
	fullURL, err := buildRequestURL(c.baseURL, path, params)
	if err != nil {
		return BytesResponse{}, err
	}

	ctx := c.context()

	token, err := c.tokenProvider.GetToken()
	if err != nil {
		return BytesResponse{}, err
	}

	refreshedAfterUnauthorized := false
	// Transport failures, 5xx, and 429 share one retry budget.
	transientRetries := 0
	requestAttempts := 0

	for {
		requestAttempts++
		reqBody := newBody()

		req, err := c.newRequest(ctx, method, fullURL, reqBody, token, contentType)
		if err != nil {
			return BytesResponse{}, fmt.Errorf("failed to create request: %w", err)
		}

		if c.debug {
			c.logAttempt(method, fullURL, requestAttempts, debugBody, contentType, reqBody != nil)
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return BytesResponse{}, fmt.Errorf("%s request canceled: %w", method, ctxErr)
			}
			if opts.allowRetries && isRetryableMethod(method) && transientRetries < maxRetries {
				delay := jitteredBackoff(transientRetries, c.randFloat)
				transientRetries++
				if waitErr := c.waitBackoff(ctx, delay); waitErr != nil {
					return BytesResponse{}, fmt.Errorf("%s request canceled while waiting to retry: %w", method, waitErr)
				}
				continue
			}
			if !opts.allowRetries || !isRetryableMethod(method) {
				return BytesResponse{}, fmt.Errorf("%s request failed; outcome may be unknown and the request was not retried: %w", method, c.safeTransportError(err))
			}
			return BytesResponse{}, fmt.Errorf("request failed after %d attempts: %w", requestAttempts, c.safeTransportError(err))
		}

		respBody, readErr := readResponseBody(resp)
		if readErr != nil {
			return BytesResponse{}, readErr
		}

		if c.debug {
			c.logResponse(resp.StatusCode, time.Since(start), respBody, resp.Header.Get("Content-Type"), opts.redactDebugResponse)
		}

		// Refresh once for reads only; a failed write response does not prove the operation was unapplied.
		if resp.StatusCode == http.StatusUnauthorized {
			if opts.allowRetries && isRetryableMethod(method) && !refreshedAfterUnauthorized {
				token, err = c.tokenProvider.RefreshToken()
				if err != nil {
					return BytesResponse{}, err
				}
				refreshedAfterUnauthorized = true
				continue
			}
		}

		// 5xx and 429 share the transport retry budget; 429 honors Retry-After.
		if opts.allowRetries && isRetryableMethod(method) && transientRetries < maxRetries {
			switch {
			case resp.StatusCode == http.StatusTooManyRequests:
				delay := retryAfterDelay(resp.Header.Get("Retry-After"), jitteredBackoff(transientRetries, c.randFloat))
				transientRetries++
				if waitErr := c.waitBackoff(ctx, delay); waitErr != nil {
					return BytesResponse{}, fmt.Errorf("%s request canceled while waiting to retry: %w", method, waitErr)
				}
				continue
			case retryableStatusCodes[resp.StatusCode]:
				delay := jitteredBackoff(transientRetries, c.randFloat)
				transientRetries++
				if waitErr := c.waitBackoff(ctx, delay); waitErr != nil {
					return BytesResponse{}, fmt.Errorf("%s request canceled while waiting to retry: %w", method, waitErr)
				}
				continue
			}
		}

		// Non-retryable errors
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			message := formatError(resp.StatusCode, respBody)
			if opts.redactDebugResponse {
				message = formatError(resp.StatusCode, []byte(RedactedValue))
			}
			return BytesResponse{}, &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
				message:    message,
			}
		}

		return BytesResponse{StatusCode: resp.StatusCode, Data: respBody, ContentType: resp.Header.Get("Content-Type")}, nil
	}
}

// buildRequestURL joins the exact base/path and encodes query parameters when present.
func buildRequestURL(baseURL, path string, params map[string]string) (string, error) {
	fullURL := baseURL + path
	if len(params) == 0 {
		return fullURL, nil
	}

	parsed, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("invalid request URL %q: %w", fullURL, err)
	}
	q := parsed.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func (c *GreennodeClient) newRequest(ctx context.Context, method, fullURL string, body io.Reader, token, contentType string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}
	c.setRequestHeaders(req, token, contentType)
	return req, nil
}

// readResponseBody closes the body and rejects truncated reads.
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response body read failed; response discarded: %w", err)
	}
	return body, nil
}

// MaskPathValues registers secret path values that query-name redaction cannot identify.
func (c *GreennodeClient) MaskPathValues(values []string) {
	c.maskedPathValues = append(c.maskedPathValues, values...)
}

type displayedError struct {
	message string
	cause   error
}

func (e *displayedError) Error() string { return e.message }
func (e *displayedError) Unwrap() error { return e.cause }
func (c *GreennodeClient) safeTransportError(err error) error {
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		return err
	}
	copy := *uerr
	copy.URL = redact.PathValues(redact.URL(uerr.URL), c.maskedPathValues)
	return &displayedError{message: copy.Error(), cause: err}
}

func (c *GreennodeClient) logAttempt(method, fullURL string, attempt int, debugBody []byte, contentType string, hasBody bool) {
	loggedURL := redact.PathValues(redact.URL(fullURL), c.maskedPathValues)
	fmt.Fprintf(os.Stderr, "[debug] %s %s (attempt %d/%d)\n", method, loggedURL, attempt, maxAttempts)
	if debugBody != nil {
		if strings.Contains(strings.ToLower(contentType), "json") {
			fmt.Fprintf(os.Stderr, "[debug] request body: %s\n", redact.Body(string(debugBody)))
		} else {
			fmt.Fprintf(os.Stderr, "[debug] request body: <%d bytes; %s>\n", len(debugBody), contentType)
		}
	} else if hasBody {
		fmt.Fprintf(os.Stderr, "[debug] request body: <stream; %s>\n", contentType)
	}
}

func (c *GreennodeClient) logResponse(statusCode int, duration time.Duration, body []byte, contentType string, redactBody bool) {
	elapsed := duration.Round(time.Millisecond)
	switch {
	case redactBody:
		fmt.Fprintf(os.Stderr, "[debug] response %d in %s: %s\n", statusCode, elapsed, RedactedValue)
	case strings.Contains(strings.ToLower(contentType), "json"):
		fmt.Fprintf(os.Stderr, "[debug] response %d in %s: %s\n", statusCode, elapsed, redact.Body(string(body)))
	default:
		fmt.Fprintf(os.Stderr, "[debug] response %d in %s: <%d bytes; %s>\n", statusCode, elapsed, len(body), contentType)
	}
}

// waitBackoff honors cancellation; the injected sleep finishes independently.
func (c *GreennodeClient) waitBackoff(ctx context.Context, d time.Duration) error {
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	done := make(chan struct{})
	go func() {
		sleep(d)
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// jitteredBackoff applies exponential delay with jitter in [delay/2, delay].
// The optional test source must return [0, 1).
func jitteredBackoff(attempt int, jitter func() float64) time.Duration {
	base := retryBaseDelay * time.Duration(1<<uint(attempt))
	return fullJitter(base, jitter)
}

func fullJitter(base time.Duration, jitter func() float64) time.Duration {
	if jitter == nil {
		jitter = rand.Float64
	}
	half := base / 2
	span := base - half
	if span <= 0 {
		return half
	}
	return half + time.Duration(jitter()*float64(span))
}

// retryAfterDelay accepts seconds or HTTP-date, caps waits, and falls back for invalid headers.
func retryAfterDelay(header string, fallback time.Duration) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return clampDelay(time.Duration(seconds) * time.Second)
	}
	if when, err := http.ParseTime(header); err == nil {
		return clampDelay(time.Until(when))
	}
	return fallback
}

func clampDelay(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > retryAfterCap {
		return retryAfterCap
	}
	return d
}

func (c *GreennodeClient) setRequestHeaders(req *http.Request, token, contentType string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", UserAgent)
	for name, values := range c.headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// APIError retains raw response data for internal handling. Error returns a redacted display message.
type APIError struct {
	StatusCode int
	Body       string
	message    string
}

func (e *APIError) Error() string { return e.message }

// Message returns the redacted display error.
func (e *APIError) Message() string { return e.message }

func formatError(statusCode int, body []byte) string {
	statusText := statusMessages[statusCode]
	if statusText == "" {
		statusText = "Error"
	}

	detail := ""
	var data map[string]any
	if err := json.Unmarshal(body, &data); err == nil {
		redacted, ok := redact.JSON(data).(map[string]any)
		if !ok {
			redacted = data
		}
		if msg, ok := redacted["message"].(string); ok && msg != "" {
			detail = msg
		} else if errMsg, ok := redacted["error"].(string); ok && errMsg != "" {
			detail = errMsg
		} else if detailMsg, ok := redacted["detail"].(string); ok && detailMsg != "" {
			detail = detailMsg
		} else if errList, ok := redacted["errors"].([]any); ok && len(errList) > 0 {
			if errObj, ok := errList[0].(map[string]any); ok {
				if msg, ok := errObj["message"].(string); ok {
					detail = msg
				}
			}
		}
		// Preserve unknown error envelopes without exposing credential fields.
		if detail == "" {
			if out, err := json.Marshal(redacted); err == nil && len(out) > 0 {
				detail = strings.TrimSpace(string(out))
			}
		}
	} else {
		detail = redact.Body(string(body))
	}

	if detail != "" {
		return fmt.Sprintf("API error (HTTP %d %s): %s", statusCode, statusText, detail)
	}
	return fmt.Sprintf("API error (HTTP %d %s)", statusCode, statusText)
}
