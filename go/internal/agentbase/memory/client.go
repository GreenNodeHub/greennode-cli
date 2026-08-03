package memory

import (
	"context"
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
