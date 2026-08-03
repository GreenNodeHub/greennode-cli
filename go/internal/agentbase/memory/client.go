package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the agent-core-memory service. It wraps the
// shared base client (same auth seam as identity/gateway/runtime).
type Client struct {
	http *client.Client
}

// NewClient creates a memory Client backed by the shared coreclient.TokenProvider.
// The memory service authenticates inbound via the upstream IAM ingress
// (Bearer → portal-user-id header), so the same provider the rest of agentbase
// uses works here unchanged. Pass nil only in construction tests that never
// issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// List returns a page of memories. page is 1-based (same envelope/paging as the
// runtime service); page/size default to 1/10 server-side when omitted.
func (c *Client) List(ctx context.Context, page, size int) (*ListMemoriesResponse, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	var out ListMemoriesResponse
	if err := c.http.Get(ctx, "/memories", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create submits a new memory. Returns the created memory synchronously (state
// ACTIVE) — there is no async FSM, so no `wait` is needed.
func (c *Client) Create(ctx context.Context, req *CreateMemoryRequest) (*Memory, error) {
	var out Memory
	if err := c.http.Post(ctx, "/memories", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves a memory by id.
func (c *Client) Get(ctx context.Context, id string) (*Memory, error) {
	var out Memory
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete soft-deletes a memory by id (ACTIVE → DELETED). The service returns 200
// with an empty body.
func (c *Client) Delete(ctx context.Context, id string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/memories/%s", id), nil)
}

// Search runs a semantic search over a memory's long-term records (backed by the
// external Mem0 vector store). namespace is the resolved namespace string
// (required); returns a ranked []MemoryRecord with Score populated. Uses the
// Google-AIP :search custom verb on the memory-records collection.
func (c *Client) Search(ctx context.Context, id, namespace string, req *SearchMemoryRecordsRequest) ([]MemoryRecord, error) {
	q := url.Values{}
	q.Set("namespace", namespace)
	var out []MemoryRecord
	// Post has no query-param seam, so go through Do directly to attach namespace.
	if err := c.http.Do(ctx, http.MethodPost,
		fmt.Sprintf("/memories/%s/memory-records:search", id), q, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ----------------------------------------------------------------------------
// Sub-resources: actors / sessions / events (Slice 3)
// ----------------------------------------------------------------------------

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

// ListActors returns a page of actors for a memory.
func (c *Client) ListActors(ctx context.Context, id string, page, size int) (*ListResponseActorDto, error) {
	var out ListResponseActorDto
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s/actors", id), pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSessions returns a page of sessions for an actor within a memory.
func (c *Client) ListSessions(ctx context.Context, id, actorID string, page, size int) (*ListResponseSessionDto, error) {
	var out ListResponseSessionDto
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s/actors/%s/sessions", id, actorID), pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSessionEvents returns the events for a session. The memory service
// publishes no response schema for this endpoint, so events are returned as raw
// JSON messages — use -o json to see the verbatim server response (the table
// view is a best-effort parse). size is clamped to 100.
func (c *Client) ListSessionEvents(ctx context.Context, id, actorID, sessionID, from, to string, page, size int) ([]json.RawMessage, error) {
	if size <= 0 || size > 100 {
		size = 100
	}
	q := pageQuery(page, size)
	if from != "" {
		q.Set("fromTimestamp", from)
	}
	if to != "" {
		q.Set("toTimestamp", to)
	}
	var out []json.RawMessage
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s/actors/%s/sessions/%s/events", id, actorID, sessionID), q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateSessionEvent appends an event to a session. 200 with no response body.
func (c *Client) CreateSessionEvent(ctx context.Context, id, actorID, sessionID string, req *EventCreateRequest) error {
	return c.http.Post(ctx, fmt.Sprintf("/memories/%s/actors/%s/sessions/%s/events", id, actorID, sessionID), req, nil)
}

// DeleteSessionEvent deletes an event. 200 with no response body.
func (c *Client) DeleteSessionEvent(ctx context.Context, id, actorID, sessionID, eventID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/memories/%s/actors/%s/sessions/%s/events/%s", id, actorID, sessionID, eventID), nil)
}

// ----------------------------------------------------------------------------
// Sub-resources: long-term-memory strategies / memory-records (Slice 3)
// ----------------------------------------------------------------------------

// ListStrategies returns the long-term-memory strategies for a memory.
func (c *Client) ListStrategies(ctx context.Context, id string) ([]LongTermMemoryStrategyEntity, error) {
	var out []LongTermMemoryStrategyEntity
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s/long-term-memory-strategies", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRecords lists a memory's long-term records under a namespace. namespace
// is required; limit defaults to 100 server-side when 0.
func (c *Client) ListRecords(ctx context.Context, id, namespace string, limit int) ([]MemoryRecord, error) {
	q := url.Values{}
	q.Set("namespace", namespace)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out []MemoryRecord
	if err := c.http.Get(ctx, fmt.Sprintf("/memories/%s/memory-records", id), q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteRecord deletes a memory record by id. 200 with no response body.
func (c *Client) DeleteRecord(ctx context.Context, id, recordID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/memories/%s/memory-records/%s", id, recordID), nil)
}

// InsertRecords inserts records directly under a namespace (bypasses
// extraction). namespace is a required query param; the body is the records.
// 200 with no response body. Uses Do directly (Post has no query seam).
func (c *Client) InsertRecords(ctx context.Context, id, namespace string, req *MemoryRecordInsertDirectlyRequest) error {
	q := url.Values{}
	q.Set("namespace", namespace)
	return c.http.Do(ctx, http.MethodPost, fmt.Sprintf("/memories/%s/memory-records:insert-directly", id), q, req, nil)
}

// GenerateRecordsFromSession generates memory records from a session. actorID,
// sessionID, and strategyID are all required query params; no request body. 200
// with no response body.
func (c *Client) GenerateRecordsFromSession(ctx context.Context, id, actorID, sessionID, strategyID string) error {
	q := url.Values{}
	q.Set("actorId", actorID)
	q.Set("sessionId", sessionID)
	q.Set("longTermMemoryStrategyId", strategyID)
	return c.http.Do(ctx, http.MethodPost, fmt.Sprintf("/memories/%s/memory-records:generate-from-session", id), q, nil, nil)
}

// GenerateRecordsFromContent generates memory records from chat content.
// strategyID is a required query param; actorID/sessionID are optional (default
// empty). The body is the chat messages. 200 with no response body.
func (c *Client) GenerateRecordsFromContent(ctx context.Context, id, strategyID, actorID, sessionID string, req *MemoryRecordGenerateFromContentRequest) error {
	q := url.Values{}
	q.Set("longTermMemoryStrategyId", strategyID)
	if actorID != "" {
		q.Set("actorId", actorID)
	}
	if sessionID != "" {
		q.Set("sessionId", sessionID)
	}
	return c.http.Do(ctx, http.MethodPost, fmt.Sprintf("/memories/%s/memory-records:generate-from-content", id), q, req, nil)
}
