package agentbase

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRequireSubcommand covers the group-command RunE: bare invocation prints
// help and exits 0 (discoverability), an unknown subcommand word is a hard
// error instead of silently printing help with a success exit code (34838).
func TestRequireSubcommand(t *testing.T) {
	cmd := newSubcommandTestGroup(t)

	t.Run("bare_invocation_prints_help", func(t *testing.T) {
		var out bytes.Buffer
		cmd.SetOut(&out)
		if err := requireSubcommand(cmd, nil); err != nil {
			t.Fatalf("bare invocation must not error: %v", err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("expected help output, got: %q", out.String())
		}
	})

	t.Run("unknown_word_errors", func(t *testing.T) {
		err := requireSubcommand(cmd, []string{"bogus"})
		if err == nil {
			t.Fatal("expected error for unknown subcommand")
		}
		if !strings.Contains(err.Error(), `unknown subcommand "bogus"`) {
			t.Errorf("unexpected error text: %v", err)
		}
		if !strings.Contains(err.Error(), "--help") {
			t.Errorf("error should point at --help: %v", err)
		}
	})
}

// TestGroupCommands_AreRunnable is the 34838 invariant as a tree walk: every
// command that owns subcommands must be runnable (Run/RunE set — wired to
// requireSubcommand), otherwise an unknown subcommand word silently prints
// help and exits 0. Walking the real tree also catches future groups that
// forget the wiring.
func TestGroupCommands_AreRunnable(t *testing.T) {
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		if c.HasSubCommands() {
			if !c.Runnable() {
				t.Errorf("%s: group command without Run/RunE (unknown subcommands would exit 0)", path)
			}
			for _, child := range c.Commands() {
				walk(child, path+" "+child.Name())
			}
		}
	}
	walk(AgentbaseCmd, "agentbase")
}

// newSubcommandTestGroup builds a throwaway group command with one child, so
// requireSubcommand's help/error paths are exercised on a real cobra command.
func newSubcommandTestGroup(t *testing.T) *cobra.Command {
	t.Helper()
	group := &cobra.Command{Use: "group", Short: "test group", RunE: requireSubcommand}
	group.AddCommand(&cobra.Command{Use: "child", Short: "test child", RunE: func(*cobra.Command, []string) error { return nil }})
	return group
}
