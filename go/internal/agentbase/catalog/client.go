package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the catalog surface of the agent-core-runtime
// service (/v1/flavors, /v1/openclaw-versions, /v1/openclaws). It wraps the
// shared base client (same auth seam as runtime/gateway/memory). The catalog
// group is served by the runtime service, so the client is built with
// ab.endpoints.Runtime — the same base URL the runtime client uses.
type Client struct {
	http *client.Client
}

// NewClient creates a catalog Client backed by the shared
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

// ListFlavors returns the compute-flavor catalog. resourceType is optional
// (filters by supported resource type). These are the runtime/compute flavors
// (cpu/ram/supportedResourceTypes), distinct from gateway flavors.
func (c *Client) ListFlavors(ctx context.Context, resourceType string) ([]FlavorEntity, error) {
	q := url.Values{}
	if resourceType != "" {
		q.Set("resourceType", resourceType)
	}
	var out []FlavorEntity
	if err := c.http.Get(ctx, "/v1/flavors", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListOpenClawVersions returns the available openclaw versions.
func (c *Client) ListOpenClawVersions(ctx context.Context) ([]OpenClawVersionDto, error) {
	var out []OpenClawVersionDto
	if err := c.http.Get(ctx, "/v1/openclaw-versions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListOpenClaws returns a page of openclaws. page/size default server-side when
// omitted (sent only when > 0).
func (c *Client) ListOpenClaws(ctx context.Context, page, size int) (*ListResponseOpenClawDto, error) {
	var out ListResponseOpenClawDto
	if err := c.http.Get(ctx, "/v1/openclaws", pageQuery(page, size), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateOpenClaw submits a new openclaw.
func (c *Client) CreateOpenClaw(ctx context.Context, req *OpenClawCreateRequest) (*OpenClawDto, error) {
	var out OpenClawDto
	if err := c.http.Post(ctx, "/v1/openclaws", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOpenClaw retrieves an openclaw by id.
func (c *Client) GetOpenClaw(ctx context.Context, id string) (*OpenClawDto, error) {
	var out OpenClawDto
	if err := c.http.Get(ctx, fmt.Sprintf("/v1/openclaws/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOpenClaw deletes an openclaw by id. The server returns 200 with no body.
func (c *Client) DeleteOpenClaw(ctx context.Context, id string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/v1/openclaws/%s", id), nil)
}

// StartOpenClaw starts an openclaw. No request body; 200 OK.
func (c *Client) StartOpenClaw(ctx context.Context, id string) error {
	return c.http.Post(ctx, fmt.Sprintf("/v1/openclaws/%s/start", id), nil, nil)
}

// StopOpenClaw stops an openclaw. No request body; 200 OK.
func (c *Client) StopOpenClaw(ctx context.Context, id string) error {
	return c.http.Post(ctx, fmt.Sprintf("/v1/openclaws/%s/stop", id), nil, nil)
}

// UpdateOpenClawVersion switches an openclaw to a different version. versionID
// is a required query parameter; the request body is empty. The service also
// exposes a deprecated PATCH variant of this op; this PUT is the canonical one
// (the QC report's PATCH row maps to it).
func (c *Client) UpdateOpenClawVersion(ctx context.Context, id, versionID string) (*OpenClawDto, error) {
	q := url.Values{}
	q.Set("versionId", versionID)
	// The Put helper has no query-param seam, so go through Do directly.
	var out OpenClawDto
	if err := c.http.Do(ctx, http.MethodPut, fmt.Sprintf("/v1/openclaws/%s/version", id), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
