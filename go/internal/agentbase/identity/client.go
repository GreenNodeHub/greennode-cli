package identity

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
)

// Client is the API client for the Identity service.
type Client struct {
	http *client.Client
}

// NewClient creates a new identity Client.
func NewClient(baseURL string, authProvider *auth.Provider) *Client {
	return &Client{http: client.New(baseURL, authProvider)}
}

// --- Agent Identities ---

// ListAgentIdentities returns a paginated list of agent identities.
func (c *Client) ListAgentIdentities(ctx context.Context, page, size int) (*PagedResponseAgentIdentityResponse, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	var out PagedResponseAgentIdentityResponse
	if err := c.http.Get(ctx, "/api/v1/agent-identities", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateAgentIdentity creates a new agent identity.
func (c *Client) CreateAgentIdentity(ctx context.Context, req *CreateAgentIdentityRequest) (*AgentIdentityResponse, error) {
	var out AgentIdentityResponse
	if err := c.http.Post(ctx, "/api/v1/agent-identities", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAgentIdentity retrieves an agent identity by name.
func (c *Client) GetAgentIdentity(ctx context.Context, name string) (*AgentIdentityResponse, error) {
	var out AgentIdentityResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/agent-identities/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateAgentIdentity updates an existing agent identity.
func (c *Client) UpdateAgentIdentity(ctx context.Context, name string, req *UpdateAgentIdentityRequest) (*AgentIdentityResponse, error) {
	var out AgentIdentityResponse
	if err := c.http.Put(ctx, fmt.Sprintf("/api/v1/agent-identities/%s", name), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAgentIdentity deletes an agent identity by name.
func (c *Client) DeleteAgentIdentity(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/agent-identities/%s", name), nil)
}

// --- OAuth2 Providers ---

// ListOauth2Providers returns a paginated list of OAuth2 providers.
func (c *Client) ListOauth2Providers(ctx context.Context, page, size int) (*PagedResponseOauth2ProviderResponse, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	var out PagedResponseOauth2ProviderResponse
	if err := c.http.Get(ctx, "/api/v1/outbound-auth/oauth2-providers", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateOauth2Provider creates a new OAuth2 provider.
func (c *Client) CreateOauth2Provider(ctx context.Context, req *CreateOauth2ProviderRequest) (*Oauth2ProviderResponse, error) {
	var out Oauth2ProviderResponse
	if err := c.http.Post(ctx, "/api/v1/outbound-auth/oauth2-providers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOauth2Provider retrieves an OAuth2 provider by name.
func (c *Client) GetOauth2Provider(ctx context.Context, name string) (*Oauth2ProviderResponse, error) {
	var out Oauth2ProviderResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/outbound-auth/oauth2-providers/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateOauth2Provider updates an existing OAuth2 provider.
func (c *Client) UpdateOauth2Provider(ctx context.Context, name string, req *UpdateOauth2ProviderRequest) error {
	return c.http.Put(ctx, fmt.Sprintf("/api/v1/outbound-auth/oauth2-providers/%s", name), req, nil)
}

// DeleteOauth2Provider deletes an OAuth2 provider by name.
func (c *Client) DeleteOauth2Provider(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/outbound-auth/oauth2-providers/%s", name), nil)
}

// GetM2MToken retrieves an M2M token for an agent identity via an OAuth2 provider.
func (c *Client) GetM2MToken(ctx context.Context, providerName, agentIdentityName string, req *GetM2mTokenRequest) (*TokenResponse, error) {
	var out TokenResponse
	path := fmt.Sprintf("/api/v1/outbound-auth/oauth2-providers/%s/agent-identities/%s/tokens/m2m", providerName, agentIdentityName)
	if err := c.http.Post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get3LOToken retrieves a 3-legged OAuth2 token for an agent identity.
func (c *Client) Get3LOToken(ctx context.Context, providerName, agentIdentityName string, req *ThreeLoTokenRequest) (*ThreeLoTokenResponse, error) {
	var out ThreeLoTokenResponse
	path := fmt.Sprintf("/api/v1/outbound-auth/oauth2-providers/%s/agent-identities/%s/tokens/3lo", providerName, agentIdentityName)
	if err := c.http.Post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Static API Key Providers ---

// ListApikeyProviders returns a paginated list of static API key providers.
func (c *Client) ListApikeyProviders(ctx context.Context, page, size int) (*PagedResponseApikeyProviderResponse, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	var out PagedResponseApikeyProviderResponse
	if err := c.http.Get(ctx, "/api/v1/outbound-auth/api-key-providers", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateApikeyProvider creates a new static API key provider.
func (c *Client) CreateApikeyProvider(ctx context.Context, req *CreateApikeyProviderRequest) (*ApikeyProviderResponse, error) {
	var out ApikeyProviderResponse
	if err := c.http.Post(ctx, "/api/v1/outbound-auth/api-key-providers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetApikeyProvider retrieves a static API key provider by name.
func (c *Client) GetApikeyProvider(ctx context.Context, name string) (*ApikeyProviderResponse, error) {
	var out ApikeyProviderResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/outbound-auth/api-key-providers/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateApikeyProvider updates a static API key provider.
func (c *Client) UpdateApikeyProvider(ctx context.Context, name string, req *UpdateApikeyProviderRequest) error {
	return c.http.Put(ctx, fmt.Sprintf("/api/v1/outbound-auth/api-key-providers/%s", name), req, nil)
}

// DeleteApikeyProvider deletes a static API key provider by name.
func (c *Client) DeleteApikeyProvider(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/outbound-auth/api-key-providers/%s", name), nil)
}

// GetApikeyForAgentIdentity retrieves the API key for an agent identity from a provider.
func (c *Client) GetApikeyForAgentIdentity(ctx context.Context, providerName, agentIdentityName string) (*ApikeyResponse, error) {
	var out ApikeyResponse
	path := fmt.Sprintf("/api/v1/outbound-auth/api-key-providers/%s/agent-identities/%s/api-key", providerName, agentIdentityName)
	if err := c.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Delegated API Key Providers ---

// ListDelegatedProviders returns a paginated list of delegated API key providers.
func (c *Client) ListDelegatedProviders(ctx context.Context, page, size int) (*PagedResponseDelegatedApiKeyProviderResponse, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	var out PagedResponseDelegatedApiKeyProviderResponse
	if err := c.http.Get(ctx, "/api/v1/outbound-auth/delegated-api-key-providers", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateDelegatedProvider creates a new delegated API key provider.
func (c *Client) CreateDelegatedProvider(ctx context.Context, req *CreateDelegatedApiKeyProviderRequest) (*DelegatedApiKeyProviderResponse, error) {
	var out DelegatedApiKeyProviderResponse
	if err := c.http.Post(ctx, "/api/v1/outbound-auth/delegated-api-key-providers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDelegatedProvider retrieves a delegated API key provider by name.
func (c *Client) GetDelegatedProvider(ctx context.Context, name string) (*DelegatedApiKeyProviderResponse, error) {
	var out DelegatedApiKeyProviderResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/outbound-auth/delegated-api-key-providers/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDelegatedProvider deletes a delegated API key provider by name.
func (c *Client) DeleteDelegatedProvider(ctx context.Context, name string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/api/v1/outbound-auth/delegated-api-key-providers/%s", name), nil)
}

// GetDelegatedApiKey retrieves a delegated API key for an agent identity.
func (c *Client) GetDelegatedApiKey(ctx context.Context, providerName, agentIdentityName string, req *GetDelegatedApiKeyRequest) (*GetDelegatedApiKeyResponse, error) {
	var out GetDelegatedApiKeyResponse
	path := fmt.Sprintf("/api/v1/outbound-auth/delegated-api-key-providers/%s/agent-identities/%s/api-key", providerName, agentIdentityName)
	if err := c.http.Post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
