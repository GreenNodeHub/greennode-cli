// Package runtime is the typed client for the agent-core-runtime service
// (`grn agentbase runtime`). The entity model mirrors the service's REST
// contract (POST/GET/PATCH/DELETE /agent-runtimes), which the agentbase
// /runtime endpoint fronts.
//
// Note: agent-core-runtime is a Java/Spring Boot service (not Go), but the
// wire contract is what matters here. Spring serializes fields by name
// (camelCase); no /api/v1 version prefix; paging uses {listData, page,
// pageSize, totalPage, totalItem} with ?page=&size= (1-based, defaults 1/10).
//
// Compiled in ONLY with `-tags agentbase`.
package runtime

import "time"

// ----------------------------------------------------------------------------
// Shared sub-DTOs (create + update)
// ----------------------------------------------------------------------------

// ImageAuth is the optional private-registry credentials. Password is
// send-only: it appears on create/update but is never present on a response
// (AgentRuntime has no imageAuth field), so it never round-trips back.
type ImageAuth struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Autoscaling is the HPA config. Bounds mirror the service's validation:
// replicas 1-10, utilizations 10-90.
type Autoscaling struct {
	MinReplicas       int `json:"minReplicas"`
	MaxReplicas       int `json:"maxReplicas"`
	CPUUtilization    int `json:"cpuUtilization"`
	MemoryUtilization int `json:"memoryUtilization"`
}

// ----------------------------------------------------------------------------
// Request types. name is immutable (create-only); every other field is
// @NotNull on both create and update, so update is a full-spec replacement,
// not a merge-patch.
// ----------------------------------------------------------------------------

// CreateAgentRuntimeRequest is the body for POST /agent-runtimes. name is
// required and sealed; the rest is the mutable spec.
type CreateAgentRuntimeRequest struct {
	Name                 string            `json:"name"`
	Description          string            `json:"description"`
	ImageURL             string            `json:"imageUrl"`
	ImageAuth            *ImageAuth        `json:"imageAuth,omitempty"`
	Command              []string          `json:"command"`
	Args                 []string          `json:"args"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	FlavorID             string            `json:"flavorId"`
	Autoscaling          Autoscaling       `json:"autoscaling"`
}

// UpdateAgentRuntimeRequest is the body for PATCH /agent-runtimes/{id}. Same as
// create minus name (name is immutable). Every field is required (full-spec
// replacement — updating creates a new version and rolls the default endpoint
// forward), so this is NOT JSON Merge Patch semantics.
type UpdateAgentRuntimeRequest struct {
	Description          string            `json:"description"`
	ImageURL             string            `json:"imageUrl"`
	ImageAuth            *ImageAuth        `json:"imageAuth,omitempty"`
	Command              []string          `json:"command"`
	Args                 []string          `json:"args"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	FlavorID             string            `json:"flavorId"`
	Autoscaling          Autoscaling       `json:"autoscaling"`
}

// ----------------------------------------------------------------------------
// Response types.
// ----------------------------------------------------------------------------

// AgentRuntime is the body returned by get/create and one element of list.
// description/statusReason are declared but only populated transiently by get()
// (statusReason when the IAM service account is unhealthy); list/create/update
// return them empty. Model them anyway for forward compatibility.
type AgentRuntime struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	StatusReason string    `json:"statusReason"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ListAgentRuntimesResponse is the body of GET /agent-runtimes. Distinct from
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
