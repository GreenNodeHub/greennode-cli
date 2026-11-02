// Package tools is the typed client for the agent-core-built-in-tools service
// (`grn agentbase tools`): platform-managed headless browsers and code
// interpreters that agents invoke as built-in tools. The service lives under
// its own /v1 prefix (/v1/browsers, /v1/code-interpreters) with the shared
// paging envelope ({listData, page, pageSize, totalItem, totalPage}), so the
// client is built with ab.endpoints.BuiltInTools and the same shared base
// client/auth seam as the other agentbase services.
//
// Session-scoped endpoints (everything under /v1/browsers/{id}/sessions and
// /v1/code-interpreters/{id}/sessions) accept an optional X-End-User-ID
// header identifying the end user on whose behalf the session acts; the
// client methods take it as an endUserID string and omit the header when
// empty.
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
package tools

import "time"

// Browser is a platform-managed headless browser. Create is asynchronous:
// the API returns CREATING and the instance becomes ACTIVE (or FAILED) later —
// poll with GET /v1/browsers/{id} (the `browser wait` command).
type Browser struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// BrowserPage is the body of GET /v1/browsers. Same envelope as
// runtime/memory/marketplace lists.
type BrowserPage struct {
	ListData  []Browser `json:"listData"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
	TotalItem int       `json:"totalItem"`
	TotalPage int       `json:"totalPage"`
}

// CreateBrowserRequest is the body of POST /v1/browsers. Name is required.
type CreateBrowserRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// BrowserSession is a browser session (an element of the session list). Its
// lifecycle is bound to the browser: deleting the browser deletes its
// sessions, and only then is the session recording downloadable.
type BrowserSession struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	EndUserID string    `json:"endUserId"`
	CreatedAt time.Time `json:"createdAt"`
}

// BrowserSessionPage is the body of GET /v1/browsers/{id}/sessions.
type BrowserSessionPage struct {
	ListData  []BrowserSession `json:"listData"`
	Page      int              `json:"page"`
	PageSize  int              `json:"pageSize"`
	TotalItem int              `json:"totalItem"`
	TotalPage int              `json:"totalPage"`
}

// CreateBrowserSessionRequest is the body of POST /v1/browsers/{id}/sessions.
// Both fields are optional; the server defaults the name when omitted.
type CreateBrowserSessionRequest struct {
	Name string `json:"name,omitempty"`
}

// BrowserSessionDetail is the GET /v1/browsers/{id}/sessions/{sessionId}
// response: a browser session plus its stream endpoints.
type BrowserSessionDetail struct {
	BrowserSession
	Streams *BrowserSessionStreams `json:"streams,omitempty"`
}

// BrowserSessionStreams is the stream-endpoint block of a browser session
// detail: the CDP automation stream (toggleable) and the live-view stream.
type BrowserSessionStreams struct {
	AutomationStream *BrowserAutomationStream `json:"automationStream,omitempty"`
	LiveViewStream   *BrowserLiveViewStream   `json:"liveViewStream,omitempty"`
}

// BrowserAutomationStream is the CDP relay endpoint and its status. The
// endpoint URL is consumed by CDP clients (the browser-mcp relay), not the CLI.
type BrowserAutomationStream struct {
	StreamEndpoint string `json:"streamEndpoint"`
	StreamStatus   string `json:"streamStatus"`
}

// BrowserLiveViewStream is the live-view (screenshot) relay endpoint. Access
// requires a signed URL from POST .../live-view-stream-sign.
type BrowserLiveViewStream struct {
	StreamEndpoint string `json:"streamEndpoint"`
}

// BrowserLiveViewStreamSignature is the response of POST
// /v1/browsers/{id}/sessions/{sessionId}/live-view-stream-sign: a
// short-lived signed URL for the live-view stream plus its expiry in seconds.
type BrowserLiveViewStreamSignature struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expiresIn"`
}

// UpdateBrowserSessionStreamsRequest is the body of PUT
// /v1/browsers/{id}/sessions/{sessionId}/streams. Only the automation stream
// status is toggleable (ENABLED|DISABLED); the live-view stream is
// always-on. The API returns no body, so the CLI re-GETs the session detail
// to show the resulting streams.
type UpdateBrowserSessionStreamsRequest struct {
	AutomationStreamUpdate *BrowserSessionStreamUpdate `json:"automationStreamUpdate,omitempty"`
}

// BrowserSessionStreamUpdate is the single-field update payload.
type BrowserSessionStreamUpdate struct {
	StreamStatus string `json:"streamStatus"`
}

// CodeInterpreter is a platform-managed code-execution sandbox. Create is
// asynchronous like the browser: CREATING → ACTIVE|FAILED.
type CodeInterpreter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CodeInterpreterPage is the body of GET /v1/code-interpreters.
type CodeInterpreterPage struct {
	ListData  []CodeInterpreter `json:"listData"`
	Page      int               `json:"page"`
	PageSize  int               `json:"pageSize"`
	TotalItem int               `json:"totalItem"`
	TotalPage int               `json:"totalPage"`
}

// CreateCodeInterpreterRequest is the body of POST /v1/code-interpreters.
// Name is required.
type CreateCodeInterpreterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Session is a code-interpreter session (an element of the session list).
// Sessions are created synchronously (ACTIVE immediately, no wait needed).
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	EndUserID string    `json:"endUserId"`
	CreatedAt time.Time `json:"createdAt"`
}

// SessionPage is the body of GET /v1/code-interpreters/{id}/sessions.
type SessionPage struct {
	ListData  []Session `json:"listData"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
	TotalItem int       `json:"totalItem"`
	TotalPage int       `json:"totalPage"`
}

// CreateSessionRequest is the body of POST /v1/code-interpreters/{id}/sessions.
// Both fields are optional; the server defaults the name when omitted.
type CreateSessionRequest struct {
	Name string `json:"name,omitempty"`
}

// InvokeRequest is the body of POST
// /v1/code-interpreters/{id}/sessions/{sessionId}/invoke: the tool name plus
// its arguments object.
type InvokeRequest struct {
	Name      string           `json:"name"`
	Arguments *InvokeArguments `json:"arguments"`
}

// InvokeArguments is the arguments object of an invoke. Exactly which fields
// are required depends on the tool (executeCode wants language+code,
// executeCommand wants command, writeFiles wants content, readFiles wants
// paths, getTask wants taskId, listFiles optionally takes directoryPath);
// the CLI enforces that mapping before sending.
type InvokeArguments struct {
	Language      string        `json:"language,omitempty"`
	Code          string        `json:"code,omitempty"`
	Command       string        `json:"command,omitempty"`
	Content       []FileContent `json:"content,omitempty"`
	DirectoryPath string        `json:"directoryPath,omitempty"`
	Paths         []string      `json:"paths,omitempty"`
	TaskID        string        `json:"taskId,omitempty"`
}

// FileContent is one file of a writeFiles invoke: path plus blob (the file
// content).
type FileContent struct {
	Path string `json:"path"`
	Blob string `json:"blob"`
}

// InvokeResponse is the invoke result, shaped like an LLM tool-result block:
// content items (text and/or structured entries), an optional structured
// content object, and isError marking a tool-level failure (the HTTP call
// still succeeded). The server holds the request open until the command
// finishes, so isError=true covers failed, timed-out, and truncated runs.
type InvokeResponse struct {
	Content           []interface{}           `json:"content,omitempty"`
	IsError           *bool                   `json:"isError,omitempty"`
	StructuredContent *map[string]interface{} `json:"structuredContent,omitempty"`
}

// InvokeTools lists the tool names the invoke command accepts. The mapping is
// CLI-side convenience only — the service is the source of truth and rejects
// unknown tools or bad arguments with a 400 listing the accepted values.
var InvokeTools = []string{
	"executeCode",
	"executeCommand",
	"startCommandExecution",
	"listFiles",
	"writeFiles",
	"readFiles",
	"getTask",
}
