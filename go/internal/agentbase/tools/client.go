package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the agent-core-built-in-tools service
// (/v1/browsers, /v1/code-interpreters). It wraps the shared base client
// (same auth seam as runtime/gateway/memory) and is built with
// ab.endpoints.BuiltInTools.
type Client struct {
	http *client.Client
}

// NewClient creates a tools Client backed by the shared
// coreclient.TokenProvider. Pass nil only in construction tests that never
// issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// pageQuery builds the ?page=&size= query (both omitted when <= 0, falling back
// to the server-side defaults of 1/10).
func pageQuery(page, size int) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	return q
}

// endUserHeaders builds the optional X-End-User-ID header map for the
// session-scoped endpoints. It returns nil when endUserID is empty so the
// header is omitted entirely (the API treats it as optional).
func endUserHeaders(endUserID string) map[string]string {
	if endUserID == "" {
		return nil
	}
	return map[string]string{"X-End-User-ID": endUserID}
}

// browserSessionPath builds the session-scoped path
// /v1/browsers/{id}/sessions/{sessionId}{suffix} (suffix may be empty).
func browserSessionPath(browserID, sessionID, suffix string) string {
	return fmt.Sprintf("/v1/browsers/%s/sessions/%s%s", browserID, sessionID, suffix)
}

// ciSessionPath builds the session-scoped path
// /v1/code-interpreters/{id}/sessions/{sessionId}{suffix} (suffix may be empty).
func ciSessionPath(ciID, sessionID, suffix string) string {
	return fmt.Sprintf("/v1/code-interpreters/%s/sessions/%s%s", ciID, sessionID, suffix)
}

// --- Browsers ---

// ListBrowsers returns a page of browsers. page/size default server-side when
// omitted (sent only when > 0).
func (c *Client) ListBrowsers(ctx context.Context, page, size int) (*BrowserPage, error) {
	var out BrowserPage
	if err := c.http.Get(ctx, "/v1/browsers", pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateBrowser submits a new browser. The API responds 201 with the resource
// in CREATING — poll GetBrowser until ACTIVE (the `browser wait` command).
func (c *Client) CreateBrowser(ctx context.Context, req *CreateBrowserRequest) (*Browser, error) {
	var out Browser
	if err := c.http.Post(ctx, "/v1/browsers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBrowser retrieves a browser by id.
func (c *Client) GetBrowser(ctx context.Context, id string) (*Browser, error) {
	var out Browser
	if err := c.http.Get(ctx, fmt.Sprintf("/v1/browsers/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBrowser deletes a browser by id (and, with it, its sessions).
func (c *Client) DeleteBrowser(ctx context.Context, id string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/v1/browsers/%s", id), nil)
}

// --- Browser sessions ---

// ListBrowserSessions returns a page of a browser's sessions. endUserID is
// optional (X-End-User-ID header).
func (c *Client) ListBrowserSessions(ctx context.Context, browserID, endUserID string, page, size int) (*BrowserSessionPage, error) {
	var out BrowserSessionPage
	path := fmt.Sprintf("/v1/browsers/%s/sessions", browserID)
	if err := c.http.DoWithHeaders(ctx, http.MethodGet, path, pageQuery(page, size), endUserHeaders(endUserID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateBrowserSession creates a browser session. Creation is synchronous —
// the response is the session (typically ACTIVE), no wait needed. endUserID
// is optional (X-End-User-ID header).
func (c *Client) CreateBrowserSession(ctx context.Context, browserID string, req *CreateBrowserSessionRequest, endUserID string) (*BrowserSession, error) {
	var out BrowserSession
	path := fmt.Sprintf("/v1/browsers/%s/sessions", browserID)
	if err := c.http.DoWithHeaders(ctx, http.MethodPost, path, nil, endUserHeaders(endUserID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBrowserSession retrieves a browser session detail (session + streams).
// endUserID is optional (X-End-User-ID header).
func (c *Client) GetBrowserSession(ctx context.Context, browserID, sessionID, endUserID string) (*BrowserSessionDetail, error) {
	var out BrowserSessionDetail
	path := browserSessionPath(browserID, sessionID, "")
	if err := c.http.DoWithHeaders(ctx, http.MethodGet, path, nil, endUserHeaders(endUserID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBrowserSession deletes a browser session. The session recording
// becomes downloadable only after the session is deleted (and archived).
// endUserID is optional (X-End-User-ID header).
func (c *Client) DeleteBrowserSession(ctx context.Context, browserID, sessionID, endUserID string) error {
	path := browserSessionPath(browserID, sessionID, "")
	return c.http.DoWithHeaders(ctx, http.MethodDelete, path, nil, endUserHeaders(endUserID), nil, nil)
}

// SignLiveViewStream requests a short-lived signed URL for the session's
// live-view stream. endUserID is optional (X-End-User-ID header).
func (c *Client) SignLiveViewStream(ctx context.Context, browserID, sessionID, endUserID string) (*BrowserLiveViewStreamSignature, error) {
	var out BrowserLiveViewStreamSignature
	path := browserSessionPath(browserID, sessionID, "/live-view-stream-sign")
	if err := c.http.DoWithHeaders(ctx, http.MethodPost, path, nil, endUserHeaders(endUserID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadRecording downloads the session recording as raw NDJSON bytes
// (application/x-ndjson, one JSON event per line). It 404s until the session
// is deleted and the recording archived. endUserID is optional
// (X-End-User-ID header).
func (c *Client) DownloadRecording(ctx context.Context, browserID, sessionID, endUserID string) ([]byte, error) {
	path := browserSessionPath(browserID, sessionID, "/recording")
	return c.http.GetRaw(ctx, path, nil, endUserHeaders(endUserID))
}

// UpdateBrowserStreams toggles the session's automation stream status
// (ENABLED|DISABLED). The API returns no body, so callers re-GET the session
// detail to show the resulting streams. endUserID is optional
// (X-End-User-ID header).
func (c *Client) UpdateBrowserStreams(ctx context.Context, browserID, sessionID, streamStatus, endUserID string) error {
	req := &UpdateBrowserSessionStreamsRequest{
		AutomationStreamUpdate: &BrowserSessionStreamUpdate{StreamStatus: streamStatus},
	}
	path := browserSessionPath(browserID, sessionID, "/streams")
	return c.http.DoWithHeaders(ctx, http.MethodPut, path, nil, endUserHeaders(endUserID), req, nil)
}

// --- Code interpreters ---

// ListCodeInterpreters returns a page of code interpreters. page/size default
// server-side when omitted (sent only when > 0).
func (c *Client) ListCodeInterpreters(ctx context.Context, page, size int) (*CodeInterpreterPage, error) {
	var out CodeInterpreterPage
	if err := c.http.Get(ctx, "/v1/code-interpreters", pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCodeInterpreter submits a new code interpreter. The API responds 201
// with the resource in CREATING — poll GetCodeInterpreter until ACTIVE (the
// `code-interpreter wait` command).
func (c *Client) CreateCodeInterpreter(ctx context.Context, req *CreateCodeInterpreterRequest) (*CodeInterpreter, error) {
	var out CodeInterpreter
	if err := c.http.Post(ctx, "/v1/code-interpreters", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCodeInterpreter retrieves a code interpreter by id.
func (c *Client) GetCodeInterpreter(ctx context.Context, id string) (*CodeInterpreter, error) {
	var out CodeInterpreter
	if err := c.http.Get(ctx, fmt.Sprintf("/v1/code-interpreters/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCodeInterpreter deletes a code interpreter by id (and, with it, its
// sessions).
func (c *Client) DeleteCodeInterpreter(ctx context.Context, id string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/v1/code-interpreters/%s", id), nil)
}

// ListCodeInterpreterSessions returns a page of a code interpreter's
// sessions. endUserID is optional (X-End-User-ID header).
func (c *Client) ListCodeInterpreterSessions(ctx context.Context, ciID, endUserID string, page, size int) (*SessionPage, error) {
	var out SessionPage
	path := fmt.Sprintf("/v1/code-interpreters/%s/sessions", ciID)
	if err := c.http.DoWithHeaders(ctx, http.MethodGet, path, pageQuery(page, size), endUserHeaders(endUserID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCodeInterpreterSession creates a code-interpreter session. Creation
// is synchronous — the response is the session (typically ACTIVE), no wait
// needed. endUserID is optional (X-End-User-ID header).
func (c *Client) CreateCodeInterpreterSession(ctx context.Context, ciID string, req *CreateSessionRequest, endUserID string) (*Session, error) {
	var out Session
	path := fmt.Sprintf("/v1/code-interpreters/%s/sessions", ciID)
	if err := c.http.DoWithHeaders(ctx, http.MethodPost, path, nil, endUserHeaders(endUserID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCodeInterpreterSession deletes a code-interpreter session. endUserID
// is optional (X-End-User-ID header).
func (c *Client) DeleteCodeInterpreterSession(ctx context.Context, ciID, sessionID, endUserID string) error {
	path := ciSessionPath(ciID, sessionID, "")
	return c.http.DoWithHeaders(ctx, http.MethodDelete, path, nil, endUserHeaders(endUserID), nil, nil)
}

// InvokeCodeInterpreter runs a tool in the session and returns its result.
// The server holds the request open until the command finishes; an
// InvokeResponse with isError=true is a tool-level failure on an otherwise
// successful HTTP call. endUserID is optional (X-End-User-ID header).
func (c *Client) InvokeCodeInterpreter(ctx context.Context, ciID, sessionID string, req *InvokeRequest, endUserID string) (*InvokeResponse, error) {
	var out InvokeResponse
	path := ciSessionPath(ciID, sessionID, "/invoke")
	if err := c.http.DoWithHeaders(ctx, http.MethodPost, path, nil, endUserHeaders(endUserID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
