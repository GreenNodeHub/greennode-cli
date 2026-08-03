package runtime

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the agent-core-runtime service. It wraps the
// shared base client (same auth seam as identity/gateway/vks/vserver).
type Client struct {
	http *client.Client
}

// NewClient creates a runtime Client backed by the shared
// coreclient.TokenProvider. The runtime service authenticates inbound via the
// upstream IAM ingress (Bearer → portal-user-id header), so the same provider
// the rest of agentbase uses works here unchanged. Pass nil only in
// construction tests that never issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// List returns a page of agent runtimes. page is 1-based (the runtime service
// uses 1-based page/size, like gateway but unlike identity's 0-based); page/size
// default to 1/10 server-side when omitted.
func (c *Client) List(ctx context.Context, page, size int) (*ListAgentRuntimesResponse, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	var out ListAgentRuntimesResponse
	if err := c.http.Get(ctx, "/agent-runtimes", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create submits a new agent runtime. Returns the runtime in its initial
// (CREATING) state; converge with `wait`.
func (c *Client) Create(ctx context.Context, req *CreateAgentRuntimeRequest) (*AgentRuntime, error) {
	var out AgentRuntime
	if err := c.http.Post(ctx, "/agent-runtimes", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves an agent runtime by id (the runtime API keys on id, not name;
// name is immutable but not the path key).
func (c *Client) Get(ctx context.Context, id string) (*AgentRuntime, error) {
	var out AgentRuntime
	if err := c.http.Get(ctx, fmt.Sprintf("/agent-runtimes/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update applies a full-spec replacement (NOT a merge-patch): every field of
// req is @NotNull server-side, so the caller must supply the complete desired
// spec (the create spec minus name). Updating creates a new version and rolls
// the default endpoint forward.
func (c *Client) Update(ctx context.Context, id string, req *UpdateAgentRuntimeRequest) (*AgentRuntime, error) {
	var out AgentRuntime
	if err := c.http.Patch(ctx, fmt.Sprintf("/agent-runtimes/%s", id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes an agent runtime by id. The service returns 200 with an empty
// body; converge with `wait` (status DELETED).
func (c *Client) Delete(ctx context.Context, id string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/agent-runtimes/%s", id), nil)
}
