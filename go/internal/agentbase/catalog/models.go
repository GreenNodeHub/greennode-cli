// Package catalog is the typed client for the catalog surface of the
// agent-core-runtime service (`grn agentbase marketplace`). The catalog group lives
// under the runtime service's /v1 prefix (/v1/flavors, /v1/openclaw-versions,
// /v1/openclaws) and shares the runtime base URL + paging envelope
// ({listData, page, pageSize, totalPage, totalItem}), so the client is built
// with the same ab.endpoints.Runtime as the runtime client — it is a separate
// package only to keep the runtime client focused on agent-runtimes.
//
// Note: /v1/flavors returns the compute-flavor catalog (cpu/ram/
// supportedResourceTypes), which is distinct from the gateway's
// /api/v1/flavors gateway-flavor list (availability/memoryGi/networkModes).
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
package catalog

import "time"

// FlavorEntity is one row of GET /v1/flavors (compute flavors).
type FlavorEntity struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	CPU                    int      `json:"cpu"`
	RAM                    int      `json:"ram"`
	SupportedResourceTypes []string `json:"supportedResourceTypes"`
	Enabled                bool     `json:"enabled"`
	Deleted                bool     `json:"deleted"`
}

// OpenClawVersionDto is one row of GET /v1/openclaw-versions.
type OpenClawVersionDto struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DefaultVersion bool   `json:"defaultVersion"`
}

// OpenClawDto is the response body of get/create and one element of
// GET /v1/openclaws.
type OpenClawDto struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	VersionID           string    `json:"versionId"`
	URL                 string    `json:"url"`
	GatewayToken        string    `json:"gatewayToken"`
	GreenNodeApiKeyName string    `json:"greenNodeApiKeyName"`
	FlavorID            string    `json:"flavorId"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
	POC                 bool      `json:"poc"`
}

// ListResponseOpenClawDto is the body of GET /v1/openclaws. Same envelope as
// runtime/memory lists.
type ListResponseOpenClawDto struct {
	ListData  []OpenClawDto `json:"listData"`
	Page      int           `json:"page"`
	PageSize  int           `json:"pageSize"`
	TotalPage int           `json:"totalPage"`
	TotalItem int           `json:"totalItem"`
}

// GreenNodeModelProvider is the optional LLM-provider wiring on an openclaw.
type GreenNodeModelProvider struct {
	Enabled    bool   `json:"enabled"`
	ApiKeyName string `json:"apiKeyName"`
}

// OpenClawChannel is a messaging channel (telegram or zalo) on an openclaw.
type OpenClawChannel struct {
	BotToken         string   `json:"botToken"`
	DmPolicy         string   `json:"dmPolicy"`
	DmAllowedUserIds []string `json:"dmAllowedUserIds"`
}

// OpenClawChannelList holds the optional telegram/zalo channels.
type OpenClawChannelList struct {
	Telegram *OpenClawChannel `json:"telegram,omitempty"`
	Zalo     *OpenClawChannel `json:"zalo,omitempty"`
}

// OpenClawCreateRequest is the body for POST /v1/openclaws. Nothing is required
// server-side, but a useful openclaw needs name + versionId + flavorId; the
// channels/greenNodeModelProvider/environmentVariables/poc fields are optional.
// POC is a value type (no omitempty) so a --file spec round-trips faithfully
// (false is preserved, not dropped).
type OpenClawCreateRequest struct {
	Name                   string                  `json:"name,omitempty"`
	VersionID              string                  `json:"versionId,omitempty"`
	GreenNodeModelProvider *GreenNodeModelProvider `json:"greenNodeModelProvider,omitempty"`
	EnvironmentVariables   map[string]string       `json:"environmentVariables,omitempty"`
	Channels               *OpenClawChannelList    `json:"channels,omitempty"`
	FlavorID               string                  `json:"flavorId,omitempty"`
	POC                    bool                    `json:"poc"`
}
