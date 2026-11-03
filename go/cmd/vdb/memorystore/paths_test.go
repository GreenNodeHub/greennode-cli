package memorystore

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestGroupIsComplete: the five nouns, and every command under them, so a dropped
// registration shows up as a test failure rather than a missing command.
func TestGroupIsComplete(t *testing.T) {
	want := map[string][]string{
		"instance": {
			"list", "get", "list-histories", "list-replicas", "list-secrules",
			"create", "resize-instance", "delete", "start", "stop", "reboot",
			"create-replica", "detach-replica",
			"update-settings", "update-config-group", "update-secrule",
		},
		"catalog": {
			"list-engines", "list-datastores", "list-families", "list-flavor-codes",
			"list-flavors", "list-networks", "list-subnets", "list-volume-types",
			"list-config-groups",
		},
		"backup":         {"list", "get", "create", "delete", "restore", "get-free-storage"},
		"configuration":  {"list", "get", "create", "update", "delete", "list-params"},
		"backup-storage": {"list", "list-packages", "create", "resize", "delete"},
	}

	for noun, commands := range want {
		group, _, err := MemorystoreCmd.Find([]string{noun})
		if err != nil || group.Name() != noun {
			t.Errorf("noun %q is not registered", noun)
			continue
		}
		present := map[string]bool{}
		for _, sub := range group.Commands() {
			present[sub.Name()] = true
		}
		for _, name := range commands {
			if !present[name] {
				t.Errorf("memorystore %s %s is missing", noun, name)
			}
		}
	}

	// The status listing deliberately has no command: the product owner reported the
	// endpoint outdated (2026-08-13).
	catalog, _, _ := MemorystoreCmd.Find([]string{"catalog"})
	for _, sub := range catalog.Commands() {
		if strings.Contains(sub.Name(), "status") {
			t.Errorf("catalog %s exists, but /database/status is outdated and must not be used", sub.Name())
		}
	}
}

// TestEveryLeafHasHelp mirrors the repo-wide convention test, scoped to this group so
// a new command cannot land without a Short.
func TestEveryLeafHasHelp(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Runnable() && len(sub.Commands()) == 0 && strings.TrimSpace(sub.Short) == "" {
				t.Errorf("%s has no Short description", sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(MemorystoreCmd)
}
