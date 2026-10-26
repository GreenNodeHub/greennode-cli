// Package runtime is the typed client for the agent-core-runtime service
// (`grn agentbase runtime`). The entity model mirrors the service's REST
// contract (POST/GET/PATCH/DELETE /v1/agent-runtimes), which the agentbase
// /runtime endpoint fronts.
//
// Note: agent-core-runtime is a Java/Spring Boot service (not Go), but the
// wire contract is what matters here. Spring serializes fields by name
// (camelCase); all paths are versioned under /v1 (the no-prefix routes are
// legacy aliases); paging uses {listData, page, pageSize, totalPage,
// totalItem} with ?page=&size= (1-based, defaults 1/10).
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
package runtime

import (
	"encoding/json"
	"time"
)

// ----------------------------------------------------------------------------
// Shared sub-DTOs (create + update)
// ----------------------------------------------------------------------------

// ImageAuth is the optional private-registry credentials. Password is
// send-only: it appears on create/update but is never present on a response
// (AgentRuntime has no imageAuth field), so it never round-trips back.
// UseAgentBaseRegistryCredentials (pointer: only sent when set) lets the
// platform pull the image using its own registry credentials instead of
// username/password.
type ImageAuth struct {
	Enabled                         bool   `json:"enabled"`
	UseAgentBaseRegistryCredentials *bool  `json:"useAgentBaseRegistryCredentials,omitempty"`
	Username                        string `json:"username"`
	Password                        string `json:"password"`
}

// Autoscaling is the HPA config. Bounds mirror the service's validation:
// maxReplicas 1-10, utilizations 10-90. minReplicas was removed from the
// contract — the platform owns the floor (always 1), so only maxReplicas and
// the utilization targets are configurable.
type Autoscaling struct {
	MaxReplicas       int `json:"maxReplicas"`
	CPUUtilization    int `json:"cpuUtilization"`
	MemoryUtilization int `json:"memoryUtilization"`
}

// ----------------------------------------------------------------------------
// Request types. name is immutable (create-only); every other field is
// @NotNull on both create and update, so update is a full-spec replacement,
// not a merge-patch.
// ----------------------------------------------------------------------------

// NetworkConfig is the create/update network placement. mode PUBLIC exposes
// the runtime's endpoints on the public internet (default); mode VPC attaches
// them to a VPC/subnet (vpcId and subnetId are then required server-side;
// routeCidrs adds egress routes, max 20).
type NetworkConfig struct {
	Mode       string   `json:"mode"` // PUBLIC | VPC
	VpcID      string   `json:"vpcId,omitempty"`
	SubnetID   string   `json:"subnetId,omitempty"`
	RouteCidrs []string `json:"routeCidrs,omitempty"`
}

// InboundAuthJWT is the JWT validation config (only read when
// InboundAuth.Mode is JWT). source is required: DISCOVERY pulls keys from a
// .well-known OIDC document (discoveryUrl, then required), JWKS inlines the
// key set (jwks, then required, <=32KiB). Allowed* lists cap at 20/20/50.
type InboundAuthJWT struct {
	Source           string          `json:"source"` // DISCOVERY | JWKS
	JWKS             json.RawMessage `json:"jwks,omitempty"`
	DiscoveryURL     string          `json:"discoveryUrl,omitempty"`
	AllowedAudiences []string        `json:"allowedAudiences,omitempty"`
	AllowedClients   []string        `json:"allowedClients,omitempty"`
	AllowedScopes    []string        `json:"allowedScopes,omitempty"`
	PrincipalClaim   string          `json:"principalClaim,omitempty"`
}

// InboundAuth is the inbound authentication mode for a runtime's endpoints.
// mode is required: NONE is open, IAM authenticates via the platform IAM
// (Bearer token), JWT validates caller JWTs per the jwt config (then
// required).
type InboundAuth struct {
	Mode string          `json:"mode"` // NONE | IAM | JWT
	JWT  *InboundAuthJWT `json:"jwt,omitempty"`
}

// CreateAgentRuntimeRequest is the body for POST /v1/agent-runtimes. name is
// required (5-50 chars, [a-zA-Z0-9._-]) and sealed; the rest is the mutable
// spec. poc marks a proof-of-concept runtime (create-only — it cannot be
// changed by update).
type CreateAgentRuntimeRequest struct {
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	ImageURL             string            `json:"imageUrl"`
	ImageAuth            *ImageAuth        `json:"imageAuth,omitempty"`
	Command              []string          `json:"command"`
	Args                 []string          `json:"args"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	NetworkConfig        *NetworkConfig    `json:"networkConfig,omitempty"`
	AllowedCidrs         []string          `json:"allowedCidrs,omitempty"`
	InboundAuth          *InboundAuth      `json:"inboundAuth,omitempty"`
	Protocol             string            `json:"protocol,omitempty"` // HTTP | MCP
	FlavorID             string            `json:"flavorId"`
	POC                  *bool             `json:"poc,omitempty"`
	Autoscaling          Autoscaling       `json:"autoscaling"`
}

// UpdateAgentRuntimeRequest is the body for PATCH /v1/agent-runtimes/{id}.
// Same as create minus name (immutable) and poc (create-only). imageAuth is
// required server-side; the API never returns the stored password, so the CLI
// cannot blindly echo a version's imageAuth back on update. The server applies
// the body as a full-spec replacement (updating creates a new immutable
// version), so the CLI's update command carries fields whose flags are unset
// over from the current version rather than resetting them to the defaults.
type UpdateAgentRuntimeRequest struct {
	Description          string            `json:"description"`
	ImageURL             string            `json:"imageUrl"`
	ImageAuth            *ImageAuth        `json:"imageAuth,omitempty"`
	Command              []string          `json:"command"`
	Args                 []string          `json:"args"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	NetworkConfig        *NetworkConfig    `json:"networkConfig,omitempty"`
	AllowedCidrs         []string          `json:"allowedCidrs,omitempty"`
	InboundAuth          *InboundAuth      `json:"inboundAuth,omitempty"`
	Protocol             string            `json:"protocol,omitempty"` // HTTP | MCP
	FlavorID             string            `json:"flavorId"`
	Autoscaling          Autoscaling       `json:"autoscaling"`
}

// ----------------------------------------------------------------------------
// Response types.
// ----------------------------------------------------------------------------

// AgentRuntime is the body returned by get/create and one element of list
// (AgentRuntimeDto in the spec). description/statusReason are declared but
// only populated transiently by get() (statusReason when the IAM service
// account is unhealthy); list/create/update return them empty. Model them
// anyway for forward compatibility. poc marks a proof-of-concept runtime
// (create-only).
type AgentRuntime struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	DisplayStatus string    `json:"displayStatus"`
	StatusReason  string    `json:"statusReason"`
	POC           *bool     `json:"poc,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ListAgentRuntimesResponse is the body of GET /v1/agent-runtimes. Distinct from
// gateway's {items, pagination} and identity's PagedResponse: this service uses
// {listData, page, pageSize, totalPage, totalItem} with 1-based page/size query
// params (defaults 1/10).
type ListAgentRuntimesResponse struct {
	ListData  []AgentRuntime `json:"listData"`
	Page      int            `json:"page"`
	PageSize  int            `json:"pageSize"`
	TotalPage int            `json:"totalPage"`
	TotalItem int            `json:"totalItem"`
}

// ----------------------------------------------------------------------------
// Sub-resources: endpoints (Slice 4)
// ----------------------------------------------------------------------------

// AgentRuntimeEndpointDto is one endpoint of a runtime (returned by get/create
// and one row of list). version/targetVersion/liveVersion/currentReplicaCount
// are int32; targetVersion/liveVersion track a rolling update.
type AgentRuntimeEndpointDto struct {
	ID                  string    `json:"id"`
	AgentRuntimeID      string    `json:"agentRuntimeId"`
	Name                string    `json:"name"`
	Version             int       `json:"version"`
	TargetVersion       int       `json:"targetVersion"`
	LiveVersion         int       `json:"liveVersion"`
	CurrentReplicaCount int       `json:"currentReplicaCount"`
	URL                 string    `json:"url"`
	OutboundIps         []string  `json:"outboundIps"`
	Status              string    `json:"status"`
	DisplayStatus       string    `json:"displayStatus"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// AgentRuntimeEndpointCreateRequest is the body for POST
// /v1/agent-runtimes/{id}/endpoints. name is required; version is optional
// (defaults server-side, minimum 1).
type AgentRuntimeEndpointCreateRequest struct {
	Name    string `json:"name"`
	Version int    `json:"version,omitempty"`
}

// ListResponseAgentRuntimeEndpointDto is the body of GET .../endpoints.
type ListResponseAgentRuntimeEndpointDto struct {
	ListData  []AgentRuntimeEndpointDto `json:"listData"`
	Page      int                       `json:"page"`
	PageSize  int                       `json:"pageSize"`
	TotalPage int                       `json:"totalPage"`
	TotalItem int                       `json:"totalItem"`
}

// ----------------------------------------------------------------------------
// Sub-resources: logs / metrics / events (Slice 4)
// ----------------------------------------------------------------------------

// LogSearchRequest is the body for POST .../logs (runtime- and endpoint-level).
// from max 5000; limit max 500; fromTimestamp/toTimestamp/query/order optional.
type LogSearchRequest struct {
	From          int    `json:"from,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	FromTimestamp string `json:"fromTimestamp,omitempty"`
	ToTimestamp   string `json:"toTimestamp,omitempty"`
	Query         string `json:"query,omitempty"`
	Order         string `json:"order,omitempty"`
}

// LogRecord is one log line.
type LogRecord struct {
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

// LogSearchResult is the response of POST .../logs.
type LogSearchResult struct {
	TotalCount int         `json:"totalCount"`
	Logs       []LogRecord `json:"logs"`
}

// MetricDataPointDouble is one CPU-usage sample (timestamp + double value).
type MetricDataPointDouble struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricDataPointLong is one memory-usage sample (timestamp + int64 bytes).
type MetricDataPointLong struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
}

// AgentRuntimeEndpointMetrics is the response of GET .../metrics.
type AgentRuntimeEndpointMetrics struct {
	CpuCoresUsage    []MetricDataPointDouble `json:"cpuCoresUsage"`
	MemoryBytesUsage []MetricDataPointLong   `json:"memoryBytesUsage"`
}

// KubeEventDto is one kubernetes event (GET .../events).
type KubeEventDto struct {
	Message       string    `json:"message"`
	LastTimestamp time.Time `json:"lastTimestamp"`
}

// ----------------------------------------------------------------------------
// Sub-resources: versions (Slice 4)
// ----------------------------------------------------------------------------

// AgentRuntimeImageAuthDto is the image-auth view on a version (no password —
// distinct from the create/update ImageAuth, which is send-only with password).
type AgentRuntimeImageAuthDto struct {
	Enabled                         bool   `json:"enabled"`
	UseAgentBaseRegistryCredentials bool   `json:"useAgentBaseRegistryCredentials"`
	Username                        string `json:"username"`
}

// AgentRuntimeNetworkConfigEntity is the network config on a version.
type AgentRuntimeNetworkConfigEntity struct {
	Mode       string   `json:"mode"`
	VpcID      string   `json:"vpcId"`
	SubnetID   string   `json:"subnetId"`
	RouteCidrs []string `json:"routeCidrs"`
}

// AgentRuntimeInboundAuthJwtDto is the JWT inbound-auth config on a version.
// JWKS is an arbitrary JSON node (json.RawMessage) — the service models it as a
// raw JsonNode.
type AgentRuntimeInboundAuthJwtDto struct {
	Source           string          `json:"source"`
	JWKS             json.RawMessage `json:"jwks,omitempty"`
	DiscoveryURL     string          `json:"discoveryUrl"`
	AllowedAudiences []string        `json:"allowedAudiences"`
	AllowedClients   []string        `json:"allowedClients"`
	AllowedScopes    []string        `json:"allowedScopes"`
	PrincipalClaim   string          `json:"principalClaim"`
}

// AgentRuntimeInboundAuthDto is the inbound-auth config on a version.
type AgentRuntimeInboundAuthDto struct {
	Mode string                         `json:"mode"`
	JWT  *AgentRuntimeInboundAuthJwtDto `json:"jwt,omitempty"`
}

// AgentRuntimeVersionDto is one row of GET .../versions — the full spec of a
// runtime version (image, command/args/env, network, inbound auth, autoscaling).
// Reuses the existing Autoscaling type (same fields as the create request).
type AgentRuntimeVersionDto struct {
	AgentRuntimeID       string                           `json:"agentRuntimeId"`
	Version              int                              `json:"version"`
	Description          string                           `json:"description"`
	ImageURL             string                           `json:"imageUrl"`
	ImageAuth            *AgentRuntimeImageAuthDto        `json:"imageAuth,omitempty"`
	Command              []string                         `json:"command"`
	Args                 []string                         `json:"args"`
	EnvironmentVariables map[string]string                `json:"environmentVariables"`
	NetworkConfig        *AgentRuntimeNetworkConfigEntity `json:"networkConfig,omitempty"`
	AllowedCidrs         []string                         `json:"allowedCidrs"`
	InboundAuth          *AgentRuntimeInboundAuthDto      `json:"inboundAuth,omitempty"`
	Protocol             string                           `json:"protocol"`
	FlavorID             string                           `json:"flavorId"`
	Autoscaling          Autoscaling                      `json:"autoscaling"`
	CreatedAt            time.Time                        `json:"createdAt"`
}

// ListResponseAgentRuntimeVersionDto is the body of GET .../versions.
type ListResponseAgentRuntimeVersionDto struct {
	ListData  []AgentRuntimeVersionDto `json:"listData"`
	Page      int                      `json:"page"`
	PageSize  int                      `json:"pageSize"`
	TotalPage int                      `json:"totalPage"`
	TotalItem int                      `json:"totalItem"`
}
