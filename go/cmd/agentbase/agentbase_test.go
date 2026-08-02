package agentbase

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
)

// subCmdExists reports whether cmd has a direct child named name. Used for
// subtree assertions instead of cmd.Find([name]): cobra's Find treats unknown
// trailing args as positionals and returns no error, so Find does not reliably
// signal that a subcommand is absent (and an existence check via Find's error
// is therefore vacuous). Iterating Commands() is unambiguous.
func subCmdExists(c *cobra.Command, name string) bool {
	for _, sub := range c.Commands() {
		if sub.Name() == name {
			return true
		}
	}
	return false
}

func assertSubCommands(t *testing.T, c *cobra.Command, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !subCmdExists(c, want) {
			t.Errorf("%s missing subcommand %q", c.Name(), want)
		}
	}
}

func assertNoSubCommands(t *testing.T, c *cobra.Command, goners ...string) {
	t.Helper()
	for _, gone := range goners {
		if subCmdExists(c, gone) {
			t.Errorf("%s should not have subcommand %q", c.Name(), gone)
		}
	}
}

// TestAgentbaseCmd_HasContextSubtree verifies the scaffold mounted the `context`
// group under `grn agentbase` with its expected children. No network, no creds.
// `context switch` was dropped — env is set via 'grn configure set iam_env' or
// 'grn login --iam-env'.
func TestAgentbaseCmd_HasContextSubtree(t *testing.T) {
	contextCmd, _, err := AgentbaseCmd.Find([]string{"context"})
	if err != nil {
		t.Fatalf("agentbase has no 'context' subcommand: %v", err)
	}
	for _, want := range []string{"current", "headers", "decorators"} {
		if _, _, err := contextCmd.Find([]string{want}); err != nil {
			t.Errorf("context missing subcommand %q: %v", want, err)
		}
	}
}

// TestAgentbaseCmd_PersistentFlags verifies the agentbase-specific persistent
// flags (and that -i/-o shorthands exist; --output shadows grn's root flag;
// --profile is inherited from root, not re-registered here). The --env flag was
// dropped when agentbase unified onto the shared profile's iam_env.
func TestAgentbaseCmd_PersistentFlags(t *testing.T) {
	for _, flag := range []string{"interactive", "output"} {
		if AgentbaseCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("agentbase missing persistent flag %q", flag)
		}
	}
	if AgentbaseCmd.PersistentFlags().Lookup("env") != nil {
		t.Error("agentbase should no longer register --env (env is iam_env in the shared profile)")
	}
	if AgentbaseCmd.PersistentFlags().ShorthandLookup("i") == nil {
		t.Error("agentbase missing -i shorthand for --interactive")
	}
	if AgentbaseCmd.PersistentFlags().ShorthandLookup("o") == nil {
		t.Error("agentbase missing -o shorthand for --output")
	}
}

// TestAgentbaseCmd_HasIdentitySubtree verifies the identity group and its
// workload CRUD subtree mounted under `grn agentbase`. identity login/logout
// were removed when agentbase unified onto `grn configure`/`grn login`/`grn logout`;
// identity whoami/config were removed to defer config display to 'grn configure'.
func TestAgentbaseCmd_HasIdentitySubtree(t *testing.T) {
	identityCmd, _, err := AgentbaseCmd.Find([]string{"identity"})
	if err != nil {
		t.Fatalf("agentbase has no 'identity' subcommand: %v", err)
	}
	for _, want := range []string{"workload", "outbound-auth"} {
		if _, _, err := identityCmd.Find([]string{want}); err != nil {
			t.Errorf("identity missing subcommand %q: %v", want, err)
		}
	}
	for _, gone := range []string{"login", "logout", "whoami", "config"} {
		found := false
		for _, c := range identityCmd.Commands() {
			if c.Name() == gone {
				found = true
				break
			}
		}
		if found {
			t.Errorf("identity should no longer have %q (defer to grn configure/grn login/grn logout)", gone)
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

// TestAgentbaseCmd_HasGatewaySubtree verifies the gateway lifecycle group and
// its CRUD + wait + generate leaves are mounted under `grn agentbase`.
func TestAgentbaseCmd_HasGatewaySubtree(t *testing.T) {
	gwCmd, _, err := AgentbaseCmd.Find([]string{"gateway"})
	if err != nil {
		t.Fatalf("agentbase has no 'gateway' subcommand: %v", err)
	}
	for _, want := range []string{"create", "generate", "list", "get", "update", "delete", "wait"} {
		if _, _, err := gwCmd.Find([]string{want}); err != nil {
			t.Errorf("gateway missing subcommand %q: %v", want, err)
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
