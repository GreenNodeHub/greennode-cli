package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// These tests enforce cross-product CLI conventions at build time so new product
// commands stay consistent without relying on reviewers to remember the rules.
// They walk the assembled command tree from rootCmd.

// verbToken returns the action word of a command name: the segment before the
// first hyphen. For "create-nodegroup" -> "create", for "create" -> "create",
// for "cluster-active" (a wait condition) -> "cluster".
func verbToken(use string) string {
	name := strings.Fields(use)[0] // Use may be "name <args>"; take the name
	if i := strings.Index(name, "-"); i >= 0 {
		return name[:i]
	}
	return name
}

// leafCommands returns every runnable (non-group) command under root, excluding
// cobra's built-ins (help, completion).
func leafCommands() []*cobra.Command {
	var out []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			if sub.Runnable() && len(sub.Commands()) == 0 {
				out = append(out, sub)
			}
			walk(sub)
		}
	}
	walk(rootCmd)
	return out
}

// deniedVerbs are synonyms that must not creep in — keep the vocabulary uniform
// (use the canonical verb on the right instead).
var deniedVerbs = map[string]string{
	"remove":   "delete",
	"rm":       "delete",
	"del":      "delete",
	"destroy":  "delete",
	"ls":       "list",
	"fetch":    "get",
	"describe": "get",
	"show":     "get",
	"add":      "create",
	"new":      "create",
	"modify":   "update",
	"edit":     "update",
	"set":      "update", // note: `configure set` is exempt below (config file, not an API resource)
}

// destructiveVerbs identify commands that mutate/lose state and therefore must
// offer --dry-run and --force.
var destructiveVerbs = map[string]bool{
	"delete": true,
	"stop":   true,
	"reboot": true,
}

// configureExempt: the `grn configure` subcommands manage the local credential
// file (not API resources), so they use their own verbs (set/get/list).
func isConfigureSub(c *cobra.Command) bool {
	for p := c.Parent(); p != nil; p = p.Parent() {
		if p.Name() == "configure" {
			return true
		}
	}
	return false
}

// agentbaseExempt: the `grn agentbase` subtree is a gated, self-contained
// migrated subsystem (own v2 OAuth2 auth + .greennode.json config, compiled in
// only with -tags agentbase). It preserves its own command UX verbatim and is
// exempt from the cross-product verb/flag conventions for now; conformity will
// be revisited when agentbase flips to default-on.
func isAgentbaseSub(c *cobra.Command) bool {
	for p := c.Parent(); p != nil; p = p.Parent() {
		if p.Name() == "agentbase" {
			return true
		}
	}
	return false
}

func TestNoDeniedVerbs(t *testing.T) {
	for _, c := range leafCommands() {
		if isConfigureSub(c) || isAgentbaseSub(c) {
			continue
		}
		v := verbToken(c.Use)
		if canonical, denied := deniedVerbs[v]; denied {
			t.Errorf("command %q uses non-canonical verb %q; use %q instead",
				c.CommandPath(), v, canonical)
		}
	}
}

func TestDestructiveCommandsHaveDryRunAndForce(t *testing.T) {
	for _, c := range leafCommands() {
		if isAgentbaseSub(c) {
			continue
		}
		if !destructiveVerbs[verbToken(c.Use)] {
			continue
		}
		if c.Flags().Lookup("dry-run") == nil {
			t.Errorf("destructive command %q must define a --dry-run flag", c.CommandPath())
		}
		if c.Flags().Lookup("force") == nil {
			t.Errorf("destructive command %q must define a --force flag", c.CommandPath())
		}
	}
}

func TestEveryCommandHasShortHelp(t *testing.T) {
	for _, c := range leafCommands() {
		if strings.TrimSpace(c.Short) == "" {
			t.Errorf("command %q has no Short description", c.CommandPath())
		}
	}
}
