package deploy

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestImageAuthSpec_Auto verifies the string form decodes to Auto=true.
func TestImageAuthSpec_Auto(t *testing.T) {
	var a ImageAuthSpec
	if err := json.Unmarshal([]byte(`"auto"`), &a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Auto || a.Username != "" {
		t.Errorf("expected Auto=true, got %+v", a)
	}
	if !a.IsSet() {
		t.Error("Auto should count as set")
	}
}

// TestImageAuthSpec_Explicit verifies the object form decodes username/password.
func TestImageAuthSpec_Explicit(t *testing.T) {
	var a ImageAuthSpec
	if err := json.Unmarshal([]byte(`{"username":"u","password":"p"}`), &a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Auto || a.Username != "u" || a.Password != "p" {
		t.Errorf("unexpected: %+v", a)
	}
}

func TestImageAuthSpec_NullIsUnset(t *testing.T) {
	var a ImageAuthSpec
	if err := json.Unmarshal([]byte(`null`), &a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.IsSet() {
		t.Error("null should be unset")
	}
}

func TestImageAuthSpec_InvalidStringRejected(t *testing.T) {
	var a ImageAuthSpec
	err := json.Unmarshal([]byte(`"manual"`), &a)
	if err == nil {
		t.Fatal("expected error for non-auto string")
	}
	if !strings.Contains(err.Error(), `"auto"`) {
		t.Errorf("error should hint at auto, got: %v", err)
	}
}

// TestManifest_Decode verifies the full manifest decodes through the map bridge,
// including imageAuth variants and the optional memory block.
func TestManifest_Decode(t *testing.T) {
	raw := []byte(`{
		"name": "my-agent",
		"description": "demo",
		"identity": {"allowedReturnUrls": ["https://app.example.com/cb"]},
		"memory": {
			"eventExpiryDuration": 3600,
			"strategies": [{"name": "prefs", "type": "USER_PREFERENCE", "namespaceTemplate": "/strategies/USER_PREFERENCE/actors/{actorId}"}]
		},
		"runtime": {
			"image": "registry.example.com/a:v1",
			"imageAuth": "auto",
			"command": ["./agent"],
			"env": {"LOG_LEVEL": "info"},
			"flavorId": "agent.small",
			"autoscaling": {"minReplicas": 1, "maxReplicas": 3}
		}
	}`)
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "my-agent" || m.Description != "demo" {
		t.Errorf("top-level mismatch: %+v", m)
	}
	if len(m.Identity.AllowedReturnURLs) != 1 {
		t.Errorf("identity urls mismatch: %+v", m.Identity)
	}
	if m.Memory == nil || len(m.Memory.LongTermMemoryStrategies) != 1 {
		t.Fatalf("memory missing: %+v", m.Memory)
	}
	if m.Memory.LongTermMemoryStrategies[0].Type != "USER_PREFERENCE" {
		t.Errorf("strategy mismatch: %+v", m.Memory.LongTermMemoryStrategies[0])
	}
	if m.Runtime.Image != "registry.example.com/a:v1" || !m.Runtime.ImageAuth.Auto {
		t.Errorf("runtime mismatch: %+v", m.Runtime)
	}
	if m.Runtime.Autoscaling.MaxReplicas != 3 {
		t.Errorf("autoscaling mismatch: %+v", m.Runtime.Autoscaling)
	}
}

// TestManifest_StatelessAgent verifies Memory may be absent.
func TestManifest_StatelessAgent(t *testing.T) {
	raw := []byte(`{"name":"a","runtime":{"image":"img","flavorId":"f","imageAuth":{"username":"u","password":"p"}}}`)
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Memory != nil {
		t.Errorf("memory should be absent for stateless agent: %+v", m.Memory)
	}
	if m.Runtime.ImageAuth.Auto || m.Runtime.ImageAuth.Username != "u" {
		t.Errorf("explicit imageAuth mismatch: %+v", m.Runtime.ImageAuth)
	}
}
