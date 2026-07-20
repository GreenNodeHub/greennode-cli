package agentbase

import (
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
)

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

// TestAgentbaseCmd_HasIdentitySubtree verifies the identity group and its
// workload CRUD subtree mounted under `grn agentbase`.
func TestAgentbaseCmd_HasIdentitySubtree(t *testing.T) {
	identityCmd, _, err := AgentbaseCmd.Find([]string{"identity"})
	if err != nil {
		t.Fatalf("agentbase has no 'identity' subcommand: %v", err)
	}
	for _, want := range []string{"login", "logout", "whoami", "workload", "outbound-auth"} {
		if _, _, err := identityCmd.Find([]string{want}); err != nil {
			t.Errorf("identity missing subcommand %q: %v", want, err)
		}
	}
	workloadCmd, _, err := identityCmd.Find([]string{"workload"})
	if err != nil {
		t.Fatalf("identity has no 'workload' subcommand: %v", err)
	}
	for _, want := range []string{"create", "list", "get", "update", "use", "delete"} {
		if _, _, err := workloadCmd.Find([]string{want}); err != nil {
			t.Errorf("workload missing subcommand %q: %v", want, err)
		}
	}
}

// TestJoinStrings_jsonsliceArray ports the agentbase helper test for joinStrings
// (defined in identity.go).
func TestJoinStrings_jsonsliceArray(t *testing.T) {
	if got := joinStrings(jsonslice.Array[string]{"a", "b"}, ", "); got != "a, b" {
		t.Errorf("got %q", got)
	}
	if got := joinStrings(jsonslice.Array[string]{}, "|"); got != "" {
		t.Errorf("empty: got %q", got)
	}
}
