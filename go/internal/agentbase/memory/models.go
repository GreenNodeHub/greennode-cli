// Package memory is the typed client for the agent-core-memory service
// (`grn agentbase memory`). The entity model mirrors the service's REST contract
// (POST/GET/DELETE /memories + POST /memories/{id}/memory-records:search), which
// the agentbase /memory endpoint fronts.
//
// Note: agent-core-memory is a Java/Spring Boot service (not Go); only the wire
// contract matters here. Spring serializes fields by name. Beware the MIXED
// casing on the wire: entities are camelCase (createdAt, eventExpiryDuration),
// but MemoryRecord is deliberately snake_case (created_at, updated_at). No
// /api/v1 prefix; paging uses {listData, page, pageSize, totalPage, totalItem}
// with ?page=&size= (1-based, defaults 1/10) — same envelope as the runtime
// service. Resources are synchronous (no async FSM; soft-delete ACTIVE→DELETED).
//
// Compiled in ONLY with `-tags agentbase`.
package memory

import "time"

// ----------------------------------------------------------------------------
// Long-term memory strategy (embedded in create)
// ----------------------------------------------------------------------------

// LongTermMemoryStrategy is one extraction strategy embedded in a create
// request. type is a built-in strategy key (USER_PREFERENCE, SEMANTIC, CUSTOM,
// …) validated against the built_in_strategies catalog server-side.
// namespaceTemplate resolves to the namespace memory-records are stored under
// (e.g. "/strategies/SEMANTIC/actors/{actorId}"); it is required for search/list.
type LongTermMemoryStrategy struct {
	Name                                  string `json:"name"`
	Type                                  string `json:"type"`
	CustomFactExtractionPrompt            string `json:"customFactExtractionPrompt,omitempty"`
	NamespaceTemplate                     string `json:"namespaceTemplate"`
	EnableAutomaticMemoryRecordGeneration bool   `json:"enableAutomaticMemoryRecordGeneration"`
}

// ----------------------------------------------------------------------------
// Request types
// ----------------------------------------------------------------------------

// CreateMemoryRequest is the body for POST /memories. name + at least one
// longTermMemoryStrategy (with name/type/namespaceTemplate) are required.
type CreateMemoryRequest struct {
	Name                     string                   `json:"name"`
	Description              string                   `json:"description"`
	EventExpiryDuration      int                      `json:"eventExpiryDuration"`
	LongTermMemoryStrategies []LongTermMemoryStrategy `json:"longTermMemoryStrategies"`
}

// SearchMemoryRecordsRequest is the body for POST
// /memories/{id}/memory-records:search?namespace=. The endpoint also takes a
// required namespace query param (the resolved namespace string), set by the
// client. limit defaults to 100 (5-200); scoreThreshold defaults to 0 (0-1).
type SearchMemoryRecordsRequest struct {
	Query          string  `json:"query"`
	Limit          int     `json:"limit"`
	ScoreThreshold float64 `json:"scoreThreshold"`
}

// ----------------------------------------------------------------------------
// Response types.
// ----------------------------------------------------------------------------

// Memory is the top-level container returned by get/create and one element of
// list. portalUserId is set server-side from the portal-user-id header.
type Memory struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	EventExpiryDuration int       `json:"eventExpiryDuration"`
	PortalUserID        int64     `json:"portalUserId"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// ListMemoriesResponse is the body of GET /memories. Same envelope as the
// runtime service ({listData, page, pageSize, totalPage, totalItem}), 1-based.
type ListMemoriesResponse struct {
	ListData  []Memory `json:"listData"`
	Page      int      `json:"page"`
	PageSize  int      `json:"pageSize"`
	TotalPage int      `json:"totalPage"`
	TotalItem int      `json:"totalItem"`
}

// MemoryRecord is one long-term fact (stored in the external Mem0 vector store).
// Wire fields are SNAKE_CASE (created_at, updated_at) — distinct from the
// camelCase entities. Score is populated by Mem0 on search; null on a plain list
// (Slice 2). Search returns a ranked []MemoryRecord.
type MemoryRecord struct {
	ID        string    `json:"id"`
	Memory    string    `json:"memory"`
	Score     *float64  `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
