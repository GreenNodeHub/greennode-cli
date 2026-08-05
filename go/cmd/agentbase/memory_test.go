package agentbase

import (
	"strings"
	"testing"

	memorypkg "github.com/greennodehub/greennode-cli/internal/agentbase/memory"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
)

// TestAgentbaseCmd_HasMemorySubtree verifies the memory group and its leaves.
// Memory resources are synchronous (no async FSM), so there is no 'wait'.
func TestAgentbaseCmd_HasMemorySubtree(t *testing.T) {
	memCmd, _, err := AgentbaseCmd.Find([]string{"memory"})
	if err != nil {
		t.Fatalf("agentbase has no 'memory' subcommand: %v", err)
	}
	assertSubCommands(t, memCmd, "create", "generate", "list", "get", "delete", "search")
	// No 'wait' — memory resources are synchronous (no async FSM).
	assertNoSubCommands(t, memCmd, "wait")
}

// TestMemoryGenerate_RoundTripsThroughLoadMemorySpec proves the template that
// `generate` prints is itself a valid, loadable create spec.
func TestMemoryGenerate_RoundTripsThroughLoadMemorySpec(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := memoryGenerateCmd.RunE(memoryGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	req, err := loadMemorySpec([]byte(template))
	if err != nil {
		t.Fatalf("generated template is not a valid spec: %v", err)
	}
	if req.Name != "my-memory" || req.EventExpiryDuration != 30 {
		t.Errorf("round-trip decoded unexpected spec: %+v", req)
	}
	if len(req.LongTermMemoryStrategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(req.LongTermMemoryStrategies))
	}
	s := req.LongTermMemoryStrategies[0]
	if s.Name != "semantic" || s.Type != "SEMANTIC" || s.NamespaceTemplate != "/strategies/SEMANTIC/actors/{actorId}" {
		t.Errorf("strategy mismatch: %+v", s)
	}
}

func TestMemoryGenerate_PrintsJSONSkeleton(t *testing.T) {
	output.SetFormat(output.FormatJSON)
	defer output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := memoryGenerateCmd.RunE(memoryGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	if !strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Errorf("expected JSON skeleton; got:\n%s", got)
	}
}

// TestLoadMemorySpec_NestedCamelCaseDecodes exercises the YAML→map→JSON→struct
// bridge on the nested strategy list (camelCase keys: namespaceTemplate,
// customFactExtractionPrompt, enableAutomaticMemoryRecordGeneration).
func TestLoadMemorySpec_NestedCamelCaseDecodes(t *testing.T) {
	spec := []byte(`
name: nested-mem
description: a memory
eventExpiryDuration: 14
longTermMemoryStrategies:
  - name: semantic
    type: SEMANTIC
    namespaceTemplate: /strategies/SEMANTIC/actors/{actorId}
    enableAutomaticMemoryRecordGeneration: true
  - name: prefs
    type: USER_PREFERENCE
    namespaceTemplate: /strategies/USER_PREFERENCE/actors/{actorId}
    customFactExtractionPrompt: "Extract the user's stated preferences."
`)
	req, err := loadMemorySpec(spec)
	if err != nil {
		t.Fatalf("loadMemorySpec: %v", err)
	}
	if req.Name != "nested-mem" || req.EventExpiryDuration != 14 {
		t.Errorf("top-level mismatch: %+v", req)
	}
	if len(req.LongTermMemoryStrategies) != 2 {
		t.Fatalf("expected 2 strategies, got %d", len(req.LongTermMemoryStrategies))
	}
	s0 := req.LongTermMemoryStrategies[0]
	if s0.Name != "semantic" || s0.Type != "SEMANTIC" || !s0.EnableAutomaticMemoryRecordGeneration {
		t.Errorf("strategy[0] mismatch: %+v", s0)
	}
	if s0.NamespaceTemplate != "/strategies/SEMANTIC/actors/{actorId}" {
		t.Errorf("namespaceTemplate mismatch: %s", s0.NamespaceTemplate)
	}
	s1 := req.LongTermMemoryStrategies[1]
	if s1.CustomFactExtractionPrompt != "Extract the user's stated preferences." {
		t.Errorf("strategy[1] prompt mismatch: %s", s1.CustomFactExtractionPrompt)
	}

	// Ensure it round-trips into the request struct cleanly.
	_ = memorypkg.CreateMemoryRequest{
		Name: req.Name, EventExpiryDuration: req.EventExpiryDuration,
		LongTermMemoryStrategies: req.LongTermMemoryStrategies,
	}
}

func TestLoadMemorySpec_MissingRequiredFails(t *testing.T) {
	cases := map[string]string{
		"missing name":          `description: x`,
		"no strategies":         `name: m`,
		"strategy missing type": "name: m\nlongTermMemoryStrategies:\n  - name: s\n    namespaceTemplate: /ns",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadMemorySpec([]byte(spec)); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestScoreStr(t *testing.T) {
	f := 0.925
	if got := scoreStr(&f); got != "0.925" {
		t.Errorf("scoreStr(0.925)=%q", got)
	}
	if got := scoreStr(nil); got != "-" {
		t.Errorf("scoreStr(nil)=%q, want -", got)
	}
}
