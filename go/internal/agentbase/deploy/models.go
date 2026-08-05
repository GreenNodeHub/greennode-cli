// Package deploy holds the manifest model for `grn agentbase deploy` — the
// composite agent-lifecycle orchestrator. deploy has NO backend of its own: it
// drives the already-built identity + memory + runtime + cr clients.
//
// The manifest is a deploy concept (not a passthrough of any one API), so it
// uses ergonomic short keys (image, env, strategies) that the orchestrator maps
// onto each service's request shape. It is parsed YAML→map→JSON→struct (the
// shared bridge), EXCEPT imageAuth, which is a union (`auto` | {username,
// password}) decoded by a custom UnmarshalJSON.
//
// Compiled into the default grn binary (the `-tags agentbase` gate was dropped at GA).
package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Manifest is the top-level deploy spec. Name is the shared join key across the
// three services (there is no cross-service FK). Memory is optional — omit the
// whole block for a stateless agent. Identity is always created.
type Manifest struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Identity    IdentitySpec `json:"identity"`
	Memory      *MemorySpec  `json:"memory,omitempty"`
	Runtime     RuntimeSpec  `json:"runtime"`
}

// IdentitySpec configures the agent's digital identity. AllowedReturnURLs is the
// only knob (the identity name is the manifest name).
type IdentitySpec struct {
	AllowedReturnURLs []string `json:"allowedReturnUrls,omitempty"`
}

// MemorySpec configures the agent-core-memory container. At least one strategy
// is required server-side when the block is present.
type MemorySpec struct {
	EventExpiryDuration      int                      `json:"eventExpiryDuration,omitempty"`
	LongTermMemoryStrategies []ManifestMemoryStrategy `json:"strategies"`
}

// ManifestMemoryStrategy is one long-term extraction strategy. Type is a
// built-in key (USER_PREFERENCE, SEMANTIC, CUSTOM, …); NamespaceTemplate
// resolves to the namespace records are stored under.
type ManifestMemoryStrategy struct {
	Name                                  string `json:"name"`
	Type                                  string `json:"type"`
	NamespaceTemplate                     string `json:"namespaceTemplate"`
	CustomFactExtractionPrompt            string `json:"customFactExtractionPrompt,omitempty"`
	EnableAutomaticMemoryRecordGeneration bool   `json:"enableAutomaticMemoryRecordGeneration,omitempty"`
}

// RuntimeSpec configures the agent-core-runtime container. Image + FlavorID are
// required. ImageAuth resolves the private-registry pull credentials.
type RuntimeSpec struct {
	Image       string            `json:"image"`
	ImageAuth   ImageAuthSpec     `json:"imageAuth,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	FlavorID    string            `json:"flavorId"`
	Autoscaling ManifestAutoscale `json:"autoscaling,omitempty"`
}

// ManifestAutoscale is the HPA config (bounds enforced server-side).
type ManifestAutoscale struct {
	MinReplicas       int `json:"minReplicas,omitempty"`
	MaxReplicas       int `json:"maxReplicas,omitempty"`
	CPUUtilization    int `json:"cpuUtilization,omitempty"`
	MemoryUtilization int `json:"memoryUtilization,omitempty"`
}

// ImageAuthSpec is the manifest's imageAuth union. It decodes from EITHER:
//
//	imageAuth: auto              # resolve from the cr robot account
//	imageAuth: {username: ..., password: ...}
//
// `auto` wins when set. A zero-value ImageAuthSpec (field absent) means no
// private-registry auth (public image).
type ImageAuthSpec struct {
	Auto     bool
	Username string
	Password string
}

// IsSet reports whether any imageAuth was specified in the manifest.
func (a ImageAuthSpec) IsSet() bool {
	return a.Auto || a.Username != "" || a.Password != ""
}

// UnmarshalJSON accepts the string "auto" or an explicit {username, password}
// object. Anything else is an error surfaced at manifest load time.
func (a *ImageAuthSpec) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return fmt.Errorf("imageAuth: %w", err)
		}
		if s == "auto" {
			a.Auto = true
			return nil
		}
		return fmt.Errorf("imageAuth string must be %q (got %q); or use {username, password}", "auto", s)
	}
	var explicit struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(trimmed, &explicit); err != nil {
		return fmt.Errorf("imageAuth: %w", err)
	}
	a.Username = explicit.Username
	a.Password = explicit.Password
	return nil
}
