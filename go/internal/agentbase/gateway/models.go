// Package gateway is the typed client for the agentbase gateway service
// (`grn agentbase gateway`). The entity model mirrors the agent-core-gateway
// REST contract (POST/GET/PATCH/DELETE /api/v1/gateways), which the agentbase
// /gateway endpoint fronts.
//
// Compiled in ONLY with `-tags agentbase`.
package gateway

import "time"

// ----------------------------------------------------------------------------
// Request types (POST /api/v1/gateways). Mirrors agent-core-gateway's
// CreateGatewayRequest. Required fields are plain values; optional fields use
// pointers so the wire body can omit them. Slice fields use the package's
// jsonslice.Array so an absent slice never serializes as null.
// ----------------------------------------------------------------------------

// CreateGatewayRequest is the body for create. name/networkMode/flavorId/
// replicas/inboundAuth are required; the rest is optional.
type CreateGatewayRequest struct {
	Name           string               `json:"name"`
	DisplayName    string               `json:"displayName,omitempty"`
	Description    string               `json:"description,omitempty"`
	NetworkMode    string               `json:"networkMode"`
	PrivateNetwork *PrivateNetworkInput `json:"privateNetwork,omitempty"`
	FlavorID       string               `json:"flavorId"`
	Replicas       int                  `json:"replicas"`
	InboundAuth    InboundAuthRequest   `json:"inboundAuth"`
	PolicyGroupID  string               `json:"policyGroupId,omitempty"`
	Targets        []CreateTargetInput  `json:"targets,omitempty"`
	AllowedCIDRs   *[]string            `json:"allowedCidrs,omitempty"`
	HostAliases    []HostAliasInput     `json:"hostAliases,omitempty"`
}

// PrivateNetworkInput is the PRIVATE-mode VPC reference. vpcId/subnetId are
// required (and sealed at create); routes + publicEndpointEnabled are optional.
type PrivateNetworkInput struct {
	VPCID                 string   `json:"vpcId"`
	SubnetID              string   `json:"subnetId"`
	Routes                []string `json:"routes,omitempty"`
	PublicEndpointEnabled bool     `json:"publicEndpointEnabled,omitempty"`
}

// InboundAuthRequest is the gateway's inbound (caller) authentication config.
type InboundAuthRequest struct {
	Mode               string        `json:"mode"` // NONE | IAM | JWT
	ClientRedirectURIs []string      `json:"clientRedirectUris,omitempty"`
	JWT                *JWTConfigReq `json:"jwt,omitempty"`
	IAMRequireOwner    *bool         `json:"iamRequireOwner,omitempty"`
}

// JWTConfigReq is the JWT-mode inbound config. source is DISCOVERY or JWKS.
type JWTConfigReq struct {
	Source           string              `json:"source,omitempty"`
	DiscoveryURL     string              `json:"discoveryUrl,omitempty"`
	JWKS             string              `json:"jwks,omitempty"`
	AllowedAudiences []string            `json:"allowedAudiences,omitempty"`
	AllowedClients   []string            `json:"allowedClients,omitempty"`
	AllowedScopes    []string            `json:"allowedScopes,omitempty"`
	CustomClaims     []map[string]string `json:"customClaims,omitempty"`
	PrincipalClaim   string              `json:"principalClaim,omitempty"`
}

// CreateTargetInput is one upstream MCP target.
type CreateTargetInput struct {
	Name         string              `json:"name"`
	Type         string              `json:"type"` // MCP
	Endpoint     string              `json:"endpoint"`
	OutboundAuth OutboundAuthRequest `json:"outboundAuth"`
}

// OutboundAuthRequest is a target's outbound (upstream) auth. type is
// NONE | APIKEY | OAUTH | INBOUND_FORWARD.
type OutboundAuthRequest struct {
	Type              string            `json:"type"`
	ProviderSource    *string           `json:"providerSource,omitempty"` // CUSTOM | MANAGED (OAUTH only)
	Flow              string            `json:"flow,omitempty"`           // 2LO | 3LO
	ProviderName      string            `json:"providerName,omitempty"`
	Scopes            []string          `json:"scopes,omitempty"`
	ReturnURL         string            `json:"returnUrl,omitempty"`
	HeaderName        string            `json:"headerName,omitempty"`
	HeaderValuePrefix string            `json:"headerValuePrefix,omitempty"`
	CustomParameters  map[string]string `json:"customParameters,omitempty"`
}

// HostAliasInput is one /etc/hosts override entry.
type HostAliasInput struct {
	IP        string   `json:"ip"`
	Hostnames []string `json:"hostnames"`
}

// ----------------------------------------------------------------------------
// Response types (GET /api/v1/gateways). Always-present scalars are value
// types; genuinely optional fields are pointers / omitempty.
// ----------------------------------------------------------------------------

// GatewayResponse is the body returned by get/create and one element of list.
type GatewayResponse struct {
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	DisplayName            string                  `json:"displayName"`
	Description            string                  `json:"description"`
	NetworkMode            string                  `json:"networkMode"`
	PrivateNetwork         *PrivateNetworkResponse `json:"privateNetwork,omitempty"`
	Flavor                 *FlavorSnapshotResponse `json:"flavor,omitempty"`
	Replicas               int                     `json:"replicas,omitempty"`
	InboundAuth            InboundAuthResponse     `json:"inboundAuth"`
	PolicyGroupID          string                  `json:"policyGroupId,omitempty"`
	AgentIdentityName      string                  `json:"agentIdentityName,omitempty"`
	IAM                    IAMResponse             `json:"iam"`
	Endpoint               string                  `json:"endpoint,omitempty"`
	PrivateEndpoint        string                  `json:"privateEndpoint,omitempty"`
	PublicEndpoint         string                  `json:"publicEndpoint,omitempty"`
	Targets                []TargetResponse        `json:"targets"`
	AllowedCIDRs           []string                `json:"allowedCidrs"`
	HostAliases            []HostAliasResponse     `json:"hostAliases"`
	State                  string                  `json:"state"`
	LastError              *LastErrorResponse      `json:"lastError,omitempty"`
	AppliedResourceVersion string                  `json:"appliedResourceVersion,omitempty"`
	AppliedAt              *time.Time              `json:"appliedAt,omitempty"`
	CreatedAt              time.Time               `json:"createdAt"`
	UpdatedAt              time.Time               `json:"updatedAt"`
}

// PrivateNetworkResponse is the user-facing PRIVATE-mode VPC projection.
type PrivateNetworkResponse struct {
	VPCID                 string   `json:"vpcId"`
	SubnetID              string   `json:"subnetId"`
	Routes                []string `json:"routes,omitempty"`
	PublicEndpointEnabled bool     `json:"publicEndpointEnabled"`
}

// FlavorSnapshotResponse is the (frozen) flavor the gateway was created with.
type FlavorSnapshotResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	CPU         int    `json:"cpu"`
	MemoryGi    int    `json:"memoryGi"`
}

// InboundAuthResponse mirrors the stored inbound-auth config.
type InboundAuthResponse struct {
	Mode               string         `json:"mode"`
	ClientRedirectURIs []string       `json:"clientRedirectUris,omitempty"`
	JWT                *JWTConfigResp `json:"jwt,omitempty"`
	IAMRequireOwner    *bool          `json:"iamRequireOwner,omitempty"`
}

// JWTConfigResp is the response-side JWT config.
type JWTConfigResp struct {
	Source           string   `json:"source,omitempty"`
	DiscoveryURL     string   `json:"discoveryUrl,omitempty"`
	HasJWKS          bool     `json:"-"` // set when inline JWKS is present (not echoed back)
	AllowedAudiences []string `json:"allowedAudiences,omitempty"`
	AllowedClients   []string `json:"allowedClients,omitempty"`
	AllowedScopes    []string `json:"allowedScopes,omitempty"`
	PrincipalClaim   string   `json:"principalClaim,omitempty"`
}

// IAMResponse exposes only the service-account id (the OAuth2 client/secret
// stay server-side). LastAuthFailureAt is set when IAM last rejected the
// exchange — repair-service-account is the remedy.
type IAMResponse struct {
	ServiceAccountID  string     `json:"serviceAccountId"`
	LastAuthFailureAt *time.Time `json:"lastAuthFailureAt,omitempty"`
}

// TargetResponse is one upstream target as returned.
type TargetResponse struct {
	Name         string               `json:"name"`
	Type         string               `json:"type"`
	Endpoint     string               `json:"endpoint"`
	OutboundAuth OutboundAuthResponse `json:"outboundAuth"`
}

// OutboundAuthResponse is the response-side outbound auth.
type OutboundAuthResponse struct {
	Type              string            `json:"type"`
	ProviderSource    string            `json:"providerSource,omitempty"`
	Flow              string            `json:"flow,omitempty"`
	ProviderName      string            `json:"providerName,omitempty"`
	Scopes            []string          `json:"scopes,omitempty"`
	ReturnURL         string            `json:"returnUrl,omitempty"`
	HeaderName        string            `json:"headerName,omitempty"`
	HeaderValuePrefix string            `json:"headerValuePrefix,omitempty"`
	CustomParameters  map[string]string `json:"customParameters,omitempty"`
}

// LastErrorResponse is the user-facing error summary.
type LastErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Stage   string `json:"stage"`
}

// HostAliasResponse is one /etc/hosts override entry (response side).
type HostAliasResponse struct {
	IP        string   `json:"ip"`
	Hostnames []string `json:"hostnames"`
}

// Pagination is the list-response pagination block.
type Pagination struct {
	Page       int64 `json:"page"`
	PageSize   int64 `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	HasMore    bool  `json:"hasMore"`
}

// ListGatewaysResponse is the body of GET /api/v1/gateways. Unlike the identity
// service's PagedResponse envelope, the gateway list returns {items, pagination}
// and uses 1-based page / pageSize query params.
type ListGatewaysResponse struct {
	Items      []GatewayResponse `json:"items"`
	Pagination Pagination        `json:"pagination"`
}
