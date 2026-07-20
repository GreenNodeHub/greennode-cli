package agentbase

import "testing"

// TestAgentbaseCmd_HasContextSubtree verifies the scaffold mounted the `context`
// group under `grn agentbase` with its expected children. No network, no creds.
func TestAgentbaseCmd_HasContextSubtree(t *testing.T) {
	contextCmd, _, err := AgentbaseCmd.Find([]string{"context"})
	if err != nil {
		t.Fatalf("agentbase has no 'context' subcommand: %v", err)
	}
	for _, want := range []string{"switch", "current", "headers", "decorators"} {
		if _, _, err := contextCmd.Find([]string{want}); err != nil {
			t.Errorf("context missing subcommand %q: %v", want, err)
		}
	}
}

// TestAgentbaseCmd_PersistentFlags verifies the agentbase-specific persistent
// flags (and that -i/-o shorthands exist; --output shadows grn's root flag).
func TestAgentbaseCmd_PersistentFlags(t *testing.T) {
	for _, flag := range []string{"interactive", "env", "output"} {
		if AgentbaseCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("agentbase missing persistent flag %q", flag)
		}
	}
	if AgentbaseCmd.PersistentFlags().ShorthandLookup("i") == nil {
		t.Error("agentbase missing -i shorthand for --interactive")
	}
	if AgentbaseCmd.PersistentFlags().ShorthandLookup("o") == nil {
		t.Error("agentbase missing -o shorthand for --output")
	}
}
