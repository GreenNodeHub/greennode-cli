package gateway

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
	coreclient "github.com/vngcloud/greennode-cli/internal/client"
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
