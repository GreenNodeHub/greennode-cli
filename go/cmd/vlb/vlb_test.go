package vlb

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// wantTree is the complete VLB command surface. It is spelled out so that
// adding or renaming a command is a deliberate, reviewable change.
var wantTree = map[string][]string{
	"vlb": {"certificate", "listener", "loadbalancer", "package", "pool"},
	"vlb loadbalancer": {
		"clone", "configure-scaling", "create", "delete", "get", "get-clone-metadata",
		"list", "list-by-subnet", "list-headers", "list-scale-history", "resize", "update-bandwidth",
		"wait-active", "wait-deleted",
	},
	"vlb listener":        {"create", "delete", "get", "list", "policy", "update"},
	"vlb listener policy": {"create", "delete", "get", "list", "update", "update-order"},
	"vlb pool":            {"create", "delete", "get", "get-health-monitor", "list", "member", "update"},
	"vlb pool member":     {"list", "update"},
	"vlb certificate":     {"create", "delete", "get", "list"},
	"vlb package":         {"list"},
}

func find(t *testing.T, path string) *cobra.Command {
	t.Helper()
	cmd := VlbCmd
	for _, name := range strings.Fields(path)[1:] {
		var next *cobra.Command
		for _, c := range cmd.Commands() {
			if c.Name() == name {
				next = c
				break
			}
		}
		if next == nil {
			t.Fatalf("command %q not found under %q", name, cmd.CommandPath())
		}
		cmd = next
	}
	return cmd
}

func TestCommandTree(t *testing.T) {
	for path, want := range wantTree {
		t.Run(path, func(t *testing.T) {
			var got []string
			for _, c := range find(t, path).Commands() {
				if c.Name() == "help" || c.Name() == "completion" {
					continue
				}
				got = append(got, c.Name())
			}
			sort.Strings(got)
			sort.Strings(want)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("subcommands of %q =\n  %v\nwant\n  %v", path, got, want)
			}
		})
	}
}

func TestGroupsAreNotRunnableAndRejectStrayArgs(t *testing.T) {
	for path := range wantTree {
		group := find(t, path)
		if group.Args == nil {
			t.Errorf("%q should set Args (cobra.NoArgs) so a typo'd subcommand errors", path)
		}
	}
}

func TestAliases(t *testing.T) {
	for path, alias := range map[string]string{
		"vlb loadbalancer": "lb",
		"vlb certificate":  "ca",
	} {
		cmd := find(t, path)
		found := false
		for _, a := range cmd.Aliases {
			if a == alias {
				found = true
			}
		}
		if !found {
			t.Errorf("%q should have the alias %q, got %v", path, alias, cmd.Aliases)
		}
	}
}

// TestProjectScopedCommandsRequireTheirParentIDs guards the nested resources:
// every VLB path embeds the parent IDs, so those flags must be required.
func TestRequiredParentIDFlags(t *testing.T) {
	cases := map[string][]string{
		"vlb loadbalancer get":             {"load-balancer-id"},
		"vlb loadbalancer delete":          {"load-balancer-id"},
		"vlb listener list":                {"load-balancer-id"},
		"vlb listener get":                 {"load-balancer-id", "listener-id"},
		"vlb listener policy list":         {"load-balancer-id", "listener-id"},
		"vlb listener policy get":          {"load-balancer-id", "listener-id", "policy-id"},
		"vlb listener policy update-order": {"load-balancer-id", "listener-id"},
		"vlb pool list":                    {"load-balancer-id"},
		"vlb pool get":                     {"load-balancer-id", "pool-id"},
		"vlb pool member list":             {"load-balancer-id", "pool-id"},
		"vlb pool member update":           {"load-balancer-id", "pool-id"},
		"vlb certificate get":              {"certificate-id"},
	}
	for path, required := range cases {
		cmd := find(t, path)
		for _, name := range required {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("%q has no --%s flag", path, name)
				continue
			}
			if flag.Annotations[cobra.BashCompOneRequiredFlag] == nil {
				t.Errorf("%q must mark --%s as required", path, name)
			}
		}
	}
}

// TestMutatingCommandsHaveDryRun complements the repo-wide conventions test,
// which only requires --dry-run on delete/stop/reboot.
func TestMutatingCommandsHaveDryRun(t *testing.T) {
	mutating := []string{
		"vlb loadbalancer create", "vlb loadbalancer resize", "vlb loadbalancer clone",
		"vlb loadbalancer configure-scaling", "vlb loadbalancer update-bandwidth",
		"vlb listener create", "vlb listener update",
		"vlb listener policy create", "vlb listener policy update", "vlb listener policy update-order",
		"vlb pool create", "vlb pool update", "vlb pool member update",
		"vlb certificate create",
	}
	for _, path := range mutating {
		if find(t, path).Flags().Lookup("dry-run") == nil {
			t.Errorf("%q must offer --dry-run", path)
		}
	}
}

// TestDestructiveCommandsConfirm: pool member update replaces the whole list,
// so it carries --force like the deletes even though its verb is "update".
// TestWaitersAreNotMutating: a waiter only polls, so it must not carry the
// --dry-run/--force apparatus that would imply it changes something.
func TestWaitersAreNotMutating(t *testing.T) {
	for _, path := range []string{"vlb loadbalancer wait-active", "vlb loadbalancer wait-deleted"} {
		cmd := find(t, path)
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) != nil {
				t.Errorf("%q declares --%s, but a waiter changes nothing", path, flag)
			}
		}
		for _, flag := range []string{"delay", "max-attempts"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%q should offer --%s", path, flag)
			}
		}
	}
}

func TestMemberUpdateHasForce(t *testing.T) {
	if find(t, "vlb pool member update").Flags().Lookup("force") == nil {
		t.Error("`vlb pool member update` removes members not listed; it must offer --force")
	}
}
