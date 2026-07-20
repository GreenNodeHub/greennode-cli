// Package identity contains models and the API client for the GreenNode AgentBase Identity service.
package identity

import (
	"time"

	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
)

// --- Agent Identity ---

// CreateAgentIdentityRequest is the request body for creating an agent identity.
// name is required (3-50 chars, pattern: ^[a-zA-Z0-9_-]+$).
type CreateAgentIdentityRequest struct {
	Name              string                  `json:"name"`
	Description       *string                 `json:"description"`
	AllowedReturnURLs jsonslice.Array[string] `json:"allowedReturnUrls"`
}

// UpdateAgentIdentityRequest is the request body for updating an agent identity.
// All fields are optional.
type UpdateAgentIdentityRequest struct {
	Description       *string                 `json:"description"`
	AllowedReturnURLs jsonslice.Array[string] `json:"allowedReturnUrls"`
}

// AgentIdentityResponse is the response model for an agent identity.
// All fields are optional per the OpenAPI schema; pointer types are used accordingly.
type AgentIdentityResponse struct {
	ID                *string                 `json:"id"`
	Name              *string                 `json:"name"`
	Description       *string                 `json:"description"`
	AllowedReturnURLs jsonslice.Array[string] `json:"allowedReturnUrls"`
	CreatedAt         *time.Time              `json:"createdAt"`
	UpdatedAt         *time.Time              `json:"updatedAt"`
}

// PagedResponseAgentIdentityResponse is a paginated list of agent identities.
// All fields are optional per the OpenAPI schema.
type PagedResponseAgentIdentityResponse struct {
	Content       jsonslice.Array[AgentIdentityResponse] `json:"content"`
	Page          *int                                   `json:"page"`
	Size          *int                                   `json:"size"`
	TotalElements *int64                                 `json:"totalElements"`
	TotalPages    *int                                   `json:"totalPages"`
	First         *bool                                  `json:"first"`
	Last          *bool                                  `json:"last"`
	HasNext       *bool                                  `json:"hasNext"`
	HasPrevious   *bool                                  `json:"hasPrevious"`
}

// --- OAuth2 Provider ---

// CreateOauth2ProviderRequest is the request body for creating an OAuth2 provider.
// name, clientId, clientSecret, authorizationUrl, and tokenUrl are required.
type CreateOauth2ProviderRequest struct {
	Name             string `json:"name"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	AuthorizationURL string `json:"authorizationUrl"`
	TokenURL         string `json:"tokenUrl"`
}

// UpdateOauth2ProviderRequest is the request body for updating an OAuth2 provider.
// clientId, clientSecret, authorizationUrl, and tokenUrl are required.
type UpdateOauth2ProviderRequest struct {
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	AuthorizationURL string `json:"authorizationUrl"`
	TokenURL         string `json:"tokenUrl"`
}

// Oauth2ProviderResponse is the response model for an OAuth2 provider.
// All fields are optional per the OpenAPI schema.
type Oauth2ProviderResponse struct {
	ID               *string    `json:"id"`
	Name             *string    `json:"name"`
	ClientID         *string    `json:"clientId"`
	AuthorizationURL *string    `json:"authorizationUrl"`
	TokenURL         *string    `json:"tokenUrl"`
	Status           *string    `json:"status"`
	CallbackURL      *string    `json:"callbackUrl"`
	CreatedAt        *time.Time `json:"createdAt"`
	UpdatedAt        *time.Time `json:"updatedAt"`
}

// PagedResponseOauth2ProviderResponse is a paginated list of OAuth2 providers.
// All fields are optional per the OpenAPI schema.
type PagedResponseOauth2ProviderResponse struct {
	Content       jsonslice.Array[Oauth2ProviderResponse] `json:"content"`
	Page          *int                                    `json:"page"`
	Size          *int                                    `json:"size"`
	TotalElements *int64                                  `json:"totalElements"`
	TotalPages    *int                                    `json:"totalPages"`
	First         *bool                                   `json:"first"`
	Last          *bool                                   `json:"last"`
	HasNext       *bool                                   `json:"hasNext"`
	HasPrevious   *bool                                   `json:"hasPrevious"`
}

// GetM2mTokenRequest requests an M2M (client credentials) OAuth2 token.
// scopes is required (minItems: 1).
type GetM2mTokenRequest struct {
	Scopes jsonslice.Array[string] `json:"scopes"`
}

// TokenResponse holds a plain access token.
// All fields are optional per the OpenAPI schema.
type TokenResponse struct {
	AccessToken *string `json:"accessToken"`
	TokenType   *string `json:"tokenType"`
}

// ThreeLoTokenRequest requests a 3-legged OAuth2 token.
// agentUserId, returnUrl, and scopes are required.
type ThreeLoTokenRequest struct {
	AgentUserID         string                  `json:"agentUserId"`
	Scopes              jsonslice.Array[string] `json:"scopes"`
	ReturnURL           string                  `json:"returnUrl"`
	SessionID           *string                 `json:"sessionId"`
	CustomParameters    *map[string]string      `json:"customParameters"`
	CustomState         *string                 `json:"customState"`
	ForceAuthentication *bool                   `json:"forceAuthentication"`
}

// ThreeLoTokenResponse holds the 3LO token response including authorization URL.
// All fields are optional per the OpenAPI schema.
type ThreeLoTokenResponse struct {
	AccessToken      *string `json:"accessToken"`
	TokenType        *string `json:"tokenType"`
	AuthorizationURL *string `json:"authorizationUrl"`
	SessionID        *string `json:"sessionId"`
	Status           *string `json:"status"`
}

// --- API Key Provider (static) ---

// CreateApikeyProviderRequest is the request body for creating a static API key provider.
// name and apikey are required.
type CreateApikeyProviderRequest struct {
	Name   string `json:"name"`
	Apikey string `json:"apikey"`
}

// UpdateApikeyProviderRequest is the request body for updating an API key provider.
// apikey is required.
type UpdateApikeyProviderRequest struct {
	Apikey string `json:"apikey"`
}

// ApikeyProviderResponse is the response model for a static API key provider.
// All fields are optional per the OpenAPI schema.
type ApikeyProviderResponse struct {
	ID        *string    `json:"id"`
	Name      *string    `json:"name"`
	Status    *string    `json:"status"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// PagedResponseApikeyProviderResponse is a paginated list of API key providers.
// All fields are optional per the OpenAPI schema.
type PagedResponseApikeyProviderResponse struct {
	Content       jsonslice.Array[ApikeyProviderResponse] `json:"content"`
	Page          *int                                    `json:"page"`
	Size          *int                                    `json:"size"`
	TotalElements *int64                                  `json:"totalElements"`
	TotalPages    *int                                    `json:"totalPages"`
	First         *bool                                   `json:"first"`
	Last          *bool                                   `json:"last"`
	HasNext       *bool                                   `json:"hasNext"`
	HasPrevious   *bool                                   `json:"hasPrevious"`
}

// ApikeyResponse holds the API key value for a specific agent identity.
// All fields are optional per the OpenAPI schema.
type ApikeyResponse struct {
	Apikey *string `json:"apikey"`
}

// --- Delegated API Key Provider ---

// CreateDelegatedApiKeyProviderRequest is the request body for creating a delegated API key provider.
// name is required.
type CreateDelegatedApiKeyProviderRequest struct {
	Name string `json:"name"`
}

// DelegatedApiKeyProviderResponse is the response model for a delegated API key provider.
// All fields are optional per the OpenAPI schema.
type DelegatedApiKeyProviderResponse struct {
	ID        *string    `json:"id"`
	Name      *string    `json:"name"`
	Status    *string    `json:"status"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// PagedResponseDelegatedApiKeyProviderResponse is a paginated list of delegated API key providers.
// All fields are optional per the OpenAPI schema.
type PagedResponseDelegatedApiKeyProviderResponse struct {
	Content       jsonslice.Array[DelegatedApiKeyProviderResponse] `json:"content"`
	Page          *int                                             `json:"page"`
	Size          *int                                             `json:"size"`
	TotalElements *int64                                           `json:"totalElements"`
	TotalPages    *int                                             `json:"totalPages"`
	First         *bool                                            `json:"first"`
	Last          *bool                                            `json:"last"`
	HasNext       *bool                                            `json:"hasNext"`
	HasPrevious   *bool                                            `json:"hasPrevious"`
}

// GetDelegatedApiKeyRequest is the request body for obtaining a delegated API key.
// agentUserId and returnUrl are required.
type GetDelegatedApiKeyRequest struct {
	AgentUserID     string  `json:"agentUserId"`
	ReturnURL       string  `json:"returnUrl"`
	CustomState     *string `json:"customState"`
	SessionID       *string `json:"sessionId"`
	ForceDelegation *bool   `json:"forceDelegation"`
}

// GetDelegatedApiKeyResponse holds the result of a delegated API key request.
// All fields are optional per the OpenAPI schema.
type GetDelegatedApiKeyResponse struct {
	Apikey           *string `json:"apikey"`
	AuthorizationURL *string `json:"authorizationUrl"`
	SessionID        *string `json:"sessionId"`
	Status           *string `json:"status"`
}
