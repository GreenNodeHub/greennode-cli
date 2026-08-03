package runtime

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

// ----------------------------------------------------------------------------
// Sub-resources: endpoints (Slice 4)
// ----------------------------------------------------------------------------

// ListEndpoints returns a page of endpoints for a runtime.
func (c *Client) ListEndpoints(ctx context.Context, id string, page, size int) (*ListResponseAgentRuntimeEndpointDto, error) {
	var out ListResponseAgentRuntimeEndpointDto
	if err := c.http.Get(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints", id), pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEndpoint creates a new endpoint on a runtime.
func (c *Client) CreateEndpoint(ctx context.Context, id string, req *AgentRuntimeEndpointCreateRequest) (*AgentRuntimeEndpointDto, error) {
	var out AgentRuntimeEndpointDto
	if err := c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints", id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEndpoint rolls an endpoint to a target version. version is a required
// query param; the request body is empty. The service also exposes a deprecated
// PATCH variant; this PUT is the canonical one (the QC PATCH row maps to it).
// Uses Do directly (Put has no query seam).
func (c *Client) UpdateEndpoint(ctx context.Context, id, endpointID string, version int) (*AgentRuntimeEndpointDto, error) {
	q := url.Values{}
	q.Set("version", strconv.Itoa(version))
	var out AgentRuntimeEndpointDto
	if err := c.http.Do(ctx, http.MethodPut, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s", id, endpointID), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEndpoint deletes an endpoint. The service returns 200 with the deleted
// endpoint (or an empty body); the caller renders the dto or falls back to
// PrintDeletedID when the body is empty.
func (c *Client) DeleteEndpoint(ctx context.Context, id, endpointID string) (*AgentRuntimeEndpointDto, error) {
	var out AgentRuntimeEndpointDto
	if err := c.http.Delete(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s", id, endpointID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StartEndpoint starts an endpoint. No body; 200 OK.
func (c *Client) StartEndpoint(ctx context.Context, id, endpointID string) error {
	return c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s/start", id, endpointID), nil, nil)
}

// StopEndpoint stops an endpoint. No body; 200 OK.
func (c *Client) StopEndpoint(ctx context.Context, id, endpointID string) error {
	return c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s/stop", id, endpointID), nil, nil)
}

// EndpointLogs searches an endpoint's logs. 200 → LogSearchResult.
func (c *Client) EndpointLogs(ctx context.Context, id, endpointID string, req *LogSearchRequest) (*LogSearchResult, error) {
	var out LogSearchResult
	if err := c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s/logs", id, endpointID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EndpointMetrics fetches an endpoint's CPU/memory metrics over a time range.
func (c *Client) EndpointMetrics(ctx context.Context, id, endpointID, from, to string) (*AgentRuntimeEndpointMetrics, error) {
	q := url.Values{}
	if from != "" {
		q.Set("fromTimestamp", from)
	}
	if to != "" {
		q.Set("toTimestamp", to)
	}
	var out AgentRuntimeEndpointMetrics
	if err := c.http.Get(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s/metrics", id, endpointID), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EndpointEvents lists kubernetes events for an endpoint.
func (c *Client) EndpointEvents(ctx context.Context, id, endpointID string) ([]KubeEventDto, error) {
	var out []KubeEventDto
	if err := c.http.Get(ctx, fmt.Sprintf("/agent-runtimes/%s/endpoints/%s/events", id, endpointID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ----------------------------------------------------------------------------
// Sub-resources: runtime-level logs / service account / versions (Slice 4)
// ----------------------------------------------------------------------------

// Logs searches a runtime's logs (runtime-level, not per-endpoint). 200 →
// LogSearchResult.
func (c *Client) Logs(ctx context.Context, id string, req *LogSearchRequest) (*LogSearchResult, error) {
	var out LogSearchResult
	if err := c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/logs", id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetServiceAccount rotates the runtime's IAM service account. No body; 200
// OK. The service also exposes a deprecated PATCH variant; this POST is the
// canonical one (the QC PATCH row maps to it).
func (c *Client) ResetServiceAccount(ctx context.Context, id string) error {
	return c.http.Post(ctx, fmt.Sprintf("/agent-runtimes/%s/reset-service-account", id), nil, nil)
}

// ListVersions returns a page of a runtime's versions (full spec per version).
func (c *Client) ListVersions(ctx context.Context, id string, page, size int) (*ListResponseAgentRuntimeVersionDto, error) {
	var out ListResponseAgentRuntimeVersionDto
	if err := c.http.Get(ctx, fmt.Sprintf("/agent-runtimes/%s/versions", id), pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ----------------------------------------------------------------------------
// Sub-resources: tracing (Slice 4)
//
// The trace endpoints are Google-AIP custom verbs on the agent-runtimes
// collection (/agent-runtimes:get-trace, :search-traces,
// :trace-search-tag-values). They forward arbitrary query params (allParams) to
// the tracing backend and return that backend's raw JSON, so the response is
// decoded as json.RawMessage and printed verbatim.
// ----------------------------------------------------------------------------

// GetTrace fetches a single trace by id. traceID is the required traceId query
// param; params are passthrough query params forwarded to the tracing backend
// (the --param flag).
func (c *Client) GetTrace(ctx context.Context, traceID string, params url.Values) (json.RawMessage, error) {
	q := params
	if q == nil {
		q = url.Values{}
	}
	q.Set("traceId", traceID)
	var out json.RawMessage
	if err := c.http.Get(ctx, "/agent-runtimes:get-trace", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SearchTraces searches traces. params are passthrough query params forwarded to
// the tracing backend (the --param flag).
func (c *Client) SearchTraces(ctx context.Context, params url.Values) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.http.Get(ctx, "/agent-runtimes:search-traces", params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TraceTagValues lists distinct values for a trace tag. tagKey is the required
// tagKey query param; params are passthrough query params forwarded to the
// tracing backend (the --param flag).
func (c *Client) TraceTagValues(ctx context.Context, tagKey string, params url.Values) (json.RawMessage, error) {
	q := params
	if q == nil {
		q = url.Values{}
	}
	q.Set("tagKey", tagKey)
	var out json.RawMessage
	if err := c.http.Get(ctx, "/agent-runtimes:trace-search-tag-values", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}
