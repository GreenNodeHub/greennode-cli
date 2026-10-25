// Package gateway is the typed client for the agentbase gateway service
// (`grn agentbase gateway`). The entity model mirrors the agent-core-gateway
// REST contract (POST/GET/PATCH/DELETE /api/v1/gateways), which the agentbase
// /gateway endpoint fronts.
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
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

// ----------------------------------------------------------------------------
// Sub-resources: flavors / access-logs / inbound-auth / private-network (Slice 5)
// ----------------------------------------------------------------------------

// FlavorResponse is one gateway placement flavor (GET /api/v1/flavors). Distinct
// from the runtime compute-flavor catalog (cpu/ram vs memoryGi/networkModes).
type FlavorResponse struct {
	Availability  string   `json:"availability"`
	CPU           int      `json:"cpu"`
	Description   string   `json:"description"`
	DisplayName   string   `json:"displayName"`
	ID            string   `json:"id"`
	MemoryGi      int      `json:"memoryGi"`
	NetworkModes  []string `json:"networkModes"`
	ResourceTypes []string `json:"resourceTypes"`
	SortOrder     int      `json:"sortOrder"`
}

// FlavorListResponse is the body of GET /api/v1/flavors.
type FlavorListResponse struct {
	Items []FlavorResponse `json:"items"`
}

// AccessLogCaller is the caller block on an access-log entry.
type AccessLogCaller struct {
	AuthMode string `json:"authMode"`
	ID       string `json:"id"`
}

// AccessLogMCP is the MCP-method block on an access-log entry.
type AccessLogMCP struct {
	JSONRPCID string `json:"jsonRpcId"`
	Method    string `json:"method"`
	ToolName  string `json:"toolName"`
}

// AccessLogRequest is the inbound-request block on an access-log entry.
type AccessLogRequest struct {
	ClientIP    string            `json:"clientIp"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	UserAgent   string            `json:"userAgent"`
}

// AccessLogResponse is the outbound-response block on an access-log entry.
type AccessLogResponse struct {
	ContentType string `json:"contentType"`
	Status      int    `json:"status"`
	Streaming   bool   `json:"streaming"`
}

// AccessLogEntry is one row of GET .../access-logs.
type AccessLogEntry struct {
	Caller       AccessLogCaller   `json:"caller"`
	DurationMs   int               `json:"durationMs"`
	ErrorCode    string            `json:"errorCode"`
	ErrorMessage string            `json:"errorMessage"`
	MCP          AccessLogMCP      `json:"mcp"`
	Request      AccessLogRequest  `json:"request"`
	Response     AccessLogResponse `json:"response"`
	TargetName   string            `json:"targetName"`
	Timestamp    string            `json:"timestamp"`
	UpstreamURL  string            `json:"upstreamUrl"`
}

// AccessLogPagination is the pagination block on the access-log list (note:
// `total`, not `totalItems` — distinct from the gateway-list Pagination).
type AccessLogPagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

// AccessLogListResponse is the body of GET .../access-logs.
type AccessLogListResponse struct {
	Items      []AccessLogEntry    `json:"items"`
	Pagination AccessLogPagination `json:"pagination"`
}

// AccessLogDurationStats is the duration block on the stats response.
type AccessLogDurationStats struct {
	AvgMs float64 `json:"avgMs"`
	MaxMs float64 `json:"maxMs"`
	MinMs float64 `json:"minMs"`
}

// AccessLogStatsRange is the time-range block on the stats response.
type AccessLogStatsRange struct {
	From     string `json:"from"`
	Interval string `json:"interval"`
	To       string `json:"to"`
}

// AccessLogStatusBucket is one status-code bucket in the stats histogram.
type AccessLogStatusBucket struct {
	Count  int `json:"count"`
	Status int `json:"status"`
}

// AccessLogTimeBucket is one time-series bucket in the stats response.
type AccessLogTimeBucket struct {
	Count     int    `json:"count"`
	Error     int    `json:"error"`
	Success   int    `json:"success"`
	Timestamp string `json:"timestamp"`
}

// AccessLogCallerBucket is one caller bucket in the stats top-callers.
type AccessLogCallerBucket struct {
	AuthMode string `json:"authMode"`
	Count    int    `json:"count"`
	ID       string `json:"id"`
}

// AccessLogTermBucket is one name/count bucket (top tools/targets/user-agents).
type AccessLogTermBucket struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
}

// AccessLogStatsResponse is the body of GET .../access-logs/stats. successRate
// and errorRate are fractions (0..1) of totalRequests.
type AccessLogStatsResponse struct {
	Duration        AccessLogDurationStats  `json:"duration"`
	ErrorRate       float64                 `json:"errorRate"`
	Range           AccessLogStatsRange     `json:"range"`
	StatusHistogram []AccessLogStatusBucket `json:"statusHistogram"`
	SuccessRate     float64                 `json:"successRate"`
	TimeSeries      []AccessLogTimeBucket   `json:"timeSeries"`
	TopCallers      []AccessLogCallerBucket `json:"topCallers"`
	TopTargets      []AccessLogTermBucket   `json:"topTargets"`
	TopTools        []AccessLogTermBucket   `json:"topTools"`
	TopUserAgents   []AccessLogTermBucket   `json:"topUserAgents"`
	TotalRequests   int                     `json:"totalRequests"`
}

// AccessLogQuery carries the shared access-log filter params (list + stats).
// List uses the filter fields + page/pageSize; stats uses the filter fields +
// interval/topN.
type AccessLogQuery struct {
	From, To, MCPMethod, ToolName, TargetName, HTTPStatus, ClientIP string
	Page, PageSize                                                  int
	Interval                                                        string
	TopN                                                            int
}

// PutIdpAppRequest is the body for PUT .../inbound-auth/jwt/idp-app. ClientID is
// required; ClientSecret is nullable (absent/null preserves the existing secret,
// a non-empty value replaces it); Scopes replaces the scope list.
type PutIdpAppRequest struct {
	ClientID     string   `json:"clientId"`
	ClientSecret *string  `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// PrivateRoutesResponse is the body of GET/PUT .../private-network/routes
// (routes is a list of IPv4 CIDRs).
type PrivateRoutesResponse struct {
	Routes []string `json:"routes"`
}

// ReplacePrivateRoutesRequest is the body for PUT .../private-network/routes.
// routes is required (omitting it is a 400).
type ReplacePrivateRoutesRequest struct {
	Routes []string `json:"routes"`
}

// ----------------------------------------------------------------------------
// MCP connectors / MCP server templates / target tools sync
// (GET /api/v1/mcp-connectors, GET /api/v1/mcp-server-templates,
// GET|POST /api/v1/gateways/{name}/targets/{target}/tools|sync)
// ----------------------------------------------------------------------------

// MCPConnectorListResponse is the body of GET /api/v1/mcp-connectors: every
// target flattened across the caller's gateways.
type MCPConnectorListResponse struct {
	Items      []MCPConnectorResponse `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

// MCPConnectorResponse is one MCP connector: a single target on a single
// caller-owned gateway, with its ready-to-use connectUrl.
type MCPConnectorResponse struct {
	Name         string                      `json:"name"`
	Type         string                      `json:"type"`
	ConnectURL   string                      `json:"connectUrl"`
	CreatedAt    string                      `json:"createdAt"`
	UpdatedAt    string                      `json:"updatedAt"`
	Gateway      MCPConnectorGatewayRef      `json:"gateway"`
	Template     MCPConnectorTemplateRef     `json:"template"`
	OutboundAuth MCPConnectorOutboundAuthDTO `json:"outboundAuth"`
}

// MCPConnectorGatewayRef is the parent-gateway context on a connector.
type MCPConnectorGatewayRef struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	DisplayName string                     `json:"displayName"`
	Endpoint    string                     `json:"endpoint"`
	State       string                     `json:"state"`
	NetworkMode string                     `json:"networkMode"`
	UpdatedAt   string                     `json:"updatedAt"`
	InboundAuth MCPConnectorInboundAuthDTO `json:"inboundAuth"`
}

// MCPConnectorInboundAuthDTO is the parent gateway's inbound-auth summary.
type MCPConnectorInboundAuthDTO struct {
	Mode         string `json:"mode"`
	Source       string `json:"source"`
	DiscoveryURL string `json:"discoveryUrl"`
}

// MCPConnectorTemplateRef names the catalog template a connector derives from
// (or CUSTOM when it does not).
type MCPConnectorTemplateRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// MCPConnectorOutboundAuthDTO is the upstream outbound-auth descriptor.
type MCPConnectorOutboundAuthDTO struct {
	Type     string   `json:"type"`
	Flow     string   `json:"flow"`
	Provider string   `json:"provider"`
	Scopes   []string `json:"scopes"`
}

// MCPServerTemplateListResponse is the body of GET /api/v1/mcp-server-templates.
type MCPServerTemplateListResponse struct {
	Items      []MCPServerTemplateResponse `json:"items"`
	Pagination Pagination                  `json:"pagination"`
}

// MCPServerTemplateResponse is one enabled template from the MCP server
// catalog. Its authOptions' Recommended* fields are pre-fill hints for a
// target-creation form; edits go through the gateway update path.
type MCPServerTemplateResponse struct {
	ID                string                        `json:"id"`
	DisplayName       string                        `json:"displayName"`
	Vendor            string                        `json:"vendor"`
	AuthVendor        string                        `json:"authVendor"`
	Description       string                        `json:"description"`
	Endpoint          string                        `json:"endpoint"`
	Logo              string                        `json:"logo"`
	SortOrder         int                           `json:"sortOrder"`
	AuthOptions       []MCPServerTemplateAuthOption `json:"authOptions"`
	EndpointVariables []EndpointVariable            `json:"endpointVariables"`
}

// MCPServerTemplateAuthOption is one auth wiring a template recommends.
type MCPServerTemplateAuthOption struct {
	Type                         string            `json:"type"`
	Flow                         string            `json:"flow"`
	Label                        string            `json:"label"`
	RecommendedScopes            []string          `json:"recommendedScopes"`
	RecommendedHeaderName        string            `json:"recommendedHeaderName"`
	RecommendedHeaderValuePrefix string            `json:"recommendedHeaderValuePrefix"`
	RecommendedCustomParameters  map[string]string `json:"recommendedCustomParameters"`
}

// EndpointVariable is one URL-path variable a template's endpoint needs.
type EndpointVariable struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Sample      string `json:"sample"`
}

// SyncRequest is the optional body of POST .../targets/{target}/sync.
// returnTo is where the gateway redirects the browser after interactive
// (3LO) tools/list completion; omit it for a plain done page.
type SyncRequest struct {
	ReturnTo string `json:"returnTo,omitempty"`
}

// SyncResponse is the body of both GET .../targets/{target}/tools (the last
// synced list) and POST .../sync. On interactive auth the sync response comes
// back with status=authorization_required plus an authorizeUrl to open in a
// browser; the gateway then finishes tools/list server-side.
type SyncResponse struct {
	Status        string       `json:"status"`
	Tools         []SyncedTool `json:"tools"`
	ToolCount     int          `json:"toolCount"`
	LastSyncedAt  string       `json:"lastSyncedAt"`
	AuthorizeURL  string       `json:"authorizeUrl"`
	SyncSessionID string       `json:"syncSessionId"`
	Error         string       `json:"error"`
}

// SyncedTool is one tool from a target's tools/list.
type SyncedTool struct {
	Name         string                 `json:"name"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema"`
}
