package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the agentbase gateway service. It wraps the
// shared base client (same auth seam as identity/vks/vserver).
type Client struct {
	http *client.Client
}

// NewClient creates a gateway Client backed by the shared
// coreclient.TokenProvider. Pass nil only in construction tests that never
// issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// List returns a page of gateways. page is 1-based (the gateway service uses
// 1-based paging, unlike identity's 0-based); pageSize defaults to 50 server-side.
func (c *Client) List(ctx context.Context, page, pageSize int) (*ListGatewaysResponse, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pageSize", strconv.Itoa(pageSize))
	}
	var out ListGatewaysResponse
	if err := c.http.Get(ctx, "/api/v1/gateways", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create submits a new gateway. The service returns 202 + the gateway in its
// initial (WAITING_CREATING) state; converge with `wait`.
func (c *Client) Create(ctx context.Context, req *CreateGatewayRequest) (*GatewayResponse, error) {
	var out GatewayResponse
	if err := c.http.Post(ctx, "/api/v1/gateways", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves a gateway by name.
func (c *Client) Get(ctx context.Context, name string) (*GatewayResponse, error) {
	var out GatewayResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/gateways/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update applies a JSON Merge Patch (RFC 7396) to a gateway's mutable fields.
// patch is sent verbatim — only keys the caller includes are applied (absent =
// leave alone, null = clear, value = replace). Sealed-at-create fields are
// ignored by the service.
func (c *Client) Update(ctx context.Context, name string, patch map[string]interface{}) (*GatewayResponse, error) {
	var out GatewayResponse
	if err := c.http.Patch(ctx, fmt.Sprintf("/api/v1/gateways/%s", name), nil, patch, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a gateway by name. The service returns 202; converge with `wait`.
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/gateways/%s", name), nil)
}

// ----------------------------------------------------------------------------
// Sub-resources: flavors / access-logs / inbound-auth / private-network /
// service-account (Slice 5)
// ----------------------------------------------------------------------------

// ListFlavors returns gateway placement flavors. resourceType/networkMode/zoneId
// are all optional filters.
func (c *Client) ListFlavors(ctx context.Context, resourceType, networkMode, zoneID string) (*FlavorListResponse, error) {
	q := url.Values{}
	if resourceType != "" {
		q.Set("resourceType", resourceType)
	}
	if networkMode != "" {
		q.Set("networkMode", networkMode)
	}
	if zoneID != "" {
		q.Set("zoneId", zoneID)
	}
	var out FlavorListResponse
	if err := c.http.Get(ctx, "/api/v1/flavors", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListAccessLogs returns a gateway's access-log entries. Filters and paging
// come from AccessLogQuery; omitted/zero filters are not sent.
func (c *Client) ListAccessLogs(ctx context.Context, name string, qy AccessLogQuery) (*AccessLogListResponse, error) {
	q := accessLogQueryValues(qy, false)
	var out AccessLogListResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/gateways/%s/access-logs", name), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AccessLogStats returns aggregate access-log stats for a gateway. Filters
// come from AccessLogQuery (interval/topN are honored, page/pageSize ignored).
func (c *Client) AccessLogStats(ctx context.Context, name string, qy AccessLogQuery) (*AccessLogStatsResponse, error) {
	q := accessLogQueryValues(qy, true)
	var out AccessLogStatsResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/gateways/%s/access-logs/stats", name), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PutIdpApp sets (PUT) the inbound-auth JWT IdP app credentials for a gateway.
// 204 No Content on success. clientSecret==nil preserves the existing secret;
// a non-empty value replaces it.
func (c *Client) PutIdpApp(ctx context.Context, name string, req *PutIdpAppRequest) error {
	return c.http.Do(ctx, http.MethodPut, fmt.Sprintf("/api/v1/gateways/%s/inbound-auth/jwt/idp-app", name), nil, req, nil)
}

// ClearIdpApp deletes (DELETE) the inbound-auth JWT IdP app credentials. 204.
func (c *Client) ClearIdpApp(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/gateways/%s/inbound-auth/jwt/idp-app", name), nil)
}

// GetPrivateRoutes returns the PRIVATE-mode gateway's private-network routes
// (IPv4 CIDR list). A PUBLIC-mode gateway 404s with private_network_not_applicable
// — surfaced as *client.APIError by the caller.
func (c *Client) GetPrivateRoutes(ctx context.Context, name string) (*PrivateRoutesResponse, error) {
	var out PrivateRoutesResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/gateways/%s/private-network/routes", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplacePrivateRoutes replaces the PRIVATE-mode gateway's private-network
// routes (PUT, full replacement). ifMatch is the optional If-Match header (an
// ETag from a prior GET) for optimistic concurrency. 200/202 → PrivateRoutesResponse.
func (c *Client) ReplacePrivateRoutes(ctx context.Context, name, ifMatch string, req *ReplacePrivateRoutesRequest) (*PrivateRoutesResponse, error) {
	path := fmt.Sprintf("/api/v1/gateways/%s/private-network/routes", name)
	headers := map[string]string{}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	var out PrivateRoutesResponse
	if err := c.http.DoWithHeaders(ctx, http.MethodPut, path, nil, headers, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RepairServiceAccount triggers an IAM service-account repair for a gateway.
// No body; returns the refreshed gateway.
func (c *Client) RepairServiceAccount(ctx context.Context, name string) (*GatewayResponse, error) {
	var out GatewayResponse
	if err := c.http.Post(ctx, fmt.Sprintf("/api/v1/gateways/%s/service-account/repair", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// accessLogQueryValues builds the query for access-log list/stats. stats=true
// emits interval/topN and skips page/pageSize; list emits page/pageSize and
// skips interval/topN. Empty filters are omitted.
func accessLogQueryValues(qy AccessLogQuery, stats bool) url.Values {
	q := url.Values{}
	if qy.From != "" {
		q.Set("from", qy.From)
	}
	if qy.To != "" {
		q.Set("to", qy.To)
	}
	if qy.MCPMethod != "" {
		q.Set("mcpMethod", qy.MCPMethod)
	}
	if qy.ToolName != "" {
		q.Set("toolName", qy.ToolName)
	}
	if qy.TargetName != "" {
		q.Set("targetName", qy.TargetName)
	}
	if qy.HTTPStatus != "" {
		q.Set("httpStatus", qy.HTTPStatus)
	}
	if qy.ClientIP != "" {
		q.Set("clientIp", qy.ClientIP)
	}
	if stats {
		if qy.Interval != "" {
			q.Set("interval", qy.Interval)
		}
		if qy.TopN > 0 {
			q.Set("topN", strconv.Itoa(qy.TopN))
		}
	} else {
		if qy.Page > 0 {
			q.Set("page", strconv.Itoa(qy.Page))
		}
		if qy.PageSize > 0 {
			q.Set("pageSize", strconv.Itoa(qy.PageSize))
		}
	}
	return q
}
