package cr

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
	coreclient "github.com/vngcloud/greennode-cli/internal/client"
)

// Client is the API client for the agent-core-container-registry service. It
// wraps the shared base client (same auth seam as identity/gateway/runtime/
// memory/policy).
type Client struct {
	http *client.Client
}

// NewClient creates a cr Client backed by the shared coreclient.TokenProvider.
// The container-registry service authenticates inbound via the upstream IAM
// ingress (Bearer → portal-user-id header); the /api/v1 group also runs a
// provisioning middleware that lazily creates the user's repository + robot
// account on first access, so GET /repository may return a fresh repo. Pass nil
// only in construction tests that never issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// listQuery builds the ?page=&size=&name= query. page/size default server-side
// when omitted (sent only when > 0); name is sent only when non-empty.
func listQuery(page, size int, name string) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	if name != "" {
		q.Set("name", name)
	}
	return q
}

// ----------------------------------------------------------------------------
// Repository
// ----------------------------------------------------------------------------

// GetRepository returns the user's auto-provisioned repository info. First
// access may provision it.
func (c *Client) GetRepository(ctx context.Context) (*Repository, error) {
	var out Repository
	if err := c.http.Get(ctx, "/api/v1/repository", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListImages returns a page of images in the user's namespace.
func (c *Client) ListImages(ctx context.Context, name string, page, size int) (*ListImagesResponse, error) {
	var out ListImagesResponse
	if err := c.http.Get(ctx, "/api/v1/repository/images", listQuery(page, size, name), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteImage deletes an image. The target is identified by the imageName query
// parameter (not a path segment), and the server returns 204 No Content — so
// the request must use Do directly (the Delete helper cannot carry a query).
func (c *Client) DeleteImage(ctx context.Context, imageName string) error {
	q := url.Values{}
	q.Set("imageName", imageName)
	// 204 → empty body; pass nil out so Do does not attempt to decode nothing.
	return c.http.Do(ctx, http.MethodDelete, "/api/v1/repository/images", q, nil, nil)
}

// ListArtifacts returns a page of artifacts within an image. imageName is
// required (the collection is scoped to one image).
func (c *Client) ListArtifacts(ctx context.Context, imageName, name string, page, size int) (*ListArtifactsResponse, error) {
	q := listQuery(page, size, name)
	q.Set("imageName", imageName)
	var out ListArtifactsResponse
	if err := c.http.Get(ctx, "/api/v1/repository/artifacts", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteArtifact deletes a single artifact. The target is identified by the
// imageName + digest query parameters; server returns 204 No Content — use Do
// directly (Delete cannot carry a query).
func (c *Client) DeleteArtifact(ctx context.Context, imageName, digest string) error {
	q := url.Values{}
	q.Set("imageName", imageName)
	q.Set("digest", digest)
	return c.http.Do(ctx, http.MethodDelete, "/api/v1/repository/artifacts", q, nil, nil)
}

// ----------------------------------------------------------------------------
// Registry credential (robot account)
// ----------------------------------------------------------------------------

// GetRegistryCredential returns the user's robot account (username + secret).
// The secret is real and used for `docker login` — handle with care.
func (c *Client) GetRegistryCredential(ctx context.Context) (*RegistryCredential, error) {
	var out RegistryCredential
	if err := c.http.Get(ctx, "/api/v1/registry-credential", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetSecret rotates the robot-account secret. No request body. Returns the
// refreshed credential (username unchanged, new secret).
func (c *Client) ResetSecret(ctx context.Context) (*RegistryCredential, error) {
	var out RegistryCredential
	if err := c.http.Patch(ctx, "/api/v1/registry-credential/secret", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
