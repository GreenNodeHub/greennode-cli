package cmd

import (
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var contributionCounts = map[string]int{
	"iam":              103,
	"vbackup":          29,
	"vcr":              23,
	"vdb":              139,
	"vlb":              37,
	"vmonitor":         101,
	"vmonitor-log":     63,
	"vstorage":         97,
	"vstorage-gateway": 3,
	"saas-ai":          2,
}

func contributedLeaves(t *testing.T) []*cobra.Command {
	t.Helper()
	var leaves []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if len(c.Commands()) == 0 && c.Runnable() {
			leaves = append(leaves, c)
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	seen := map[string]bool{}
	for _, service := range cli.Services() {
		want, ok := contributionCounts[service.Name()]
		if !ok {
			continue
		}
		if seen[service.Name()] || service.Parent() != rootCmd {
			t.Fatalf("service registration invalid: %s", service.Name())
		}
		seen[service.Name()] = true
		before := len(leaves)
		walk(service)
		if got := len(leaves) - before; got != want {
			t.Errorf("%s has %d commands, want %d", service.Name(), got, want)
		}
	}
	for name := range contributionCounts {
		if !seen[name] {
			t.Errorf("service not registered: %s", name)
		}
	}
	return leaves
}

func TestContributionSafetyFlags(t *testing.T) {
	reads := words("debug get list mapping search test usage validate whoami")
	writes := words("acknowledge activate associate attach authorize backup change clone complete compose config copy create deactivate delete detach disable download enable favorite generate import insert install migrate move ping reboot regenerate register refresh reorder rename reset resize restore revoke rollback send start stop synthesize transcribe uninstall unregister update upgrade verify")
	destructive := words("delete download revoke rollback stop reboot reset refresh regenerate restore uninstall unregister")
	for _, c := range contributedLeaves(t) {
		t.Run(c.CommandPath(), func(t *testing.T) {
			verb, _, _ := strings.Cut(c.Name(), "-")
			if reads[verb] == writes[verb] {
				t.Fatalf("unclassified or ambiguous verb: %s", verb)
			}
			if (c.Flags().Lookup("dry-run") != nil) != writes[verb] {
				t.Error("dry-run flag disagrees with state classification")
			}
			if destructive[verb] && c.Flags().Lookup("force") == nil {
				t.Error("destructive command lacks force flag")
			}
			if strings.HasPrefix(c.CommandPath(), "grn iam ") && writes[verb] && c.Flags().Lookup("force") == nil {
				t.Error("IAM mutation lacks confirmation opt-in")
			}
			if c.RunE == nil || strings.TrimSpace(c.Short) == "" {
				t.Error("command requires RunE and short help")
			}
		})
	}
}

func TestContributionSecretDisclosure(t *testing.T) {
	want := words(
		"iam/s3-key/create iam/swift-user/create iam/service-account/reset-secret iam/iam-user/update-google iam/iam-user/current/update-google " +
			"vcr/user/create vcr/user/refresh-secret " +
			"vdb/kafka/user/create-user vdb/kafka/user/get-user-authen-credential vdb/kafka/user/regenerate-user-authen-credential " +
			"vmonitor/api-key/create-metric vmonitor/api-key/list-metric " +
			"vmonitor-log/archive/create vmonitor-log/archive/get vmonitor-log/archive/list vmonitor-log/archive/update " +
			"vmonitor-log/refill/create vmonitor-log/refill/create-from-archive vmonitor-log/refill/get vmonitor-log/refill/list")
	seen := map[string]bool{}
	for _, c := range contributedLeaves(t) {
		path := strings.ReplaceAll(strings.TrimPrefix(c.CommandPath(), "grn "), " ", "/")
		if strings.HasPrefix(path, "iam/") && (c.Name() == "get" || c.Name() == "list" || c.Name() == "whoami") {
			want[path] = true
		}
		if c.Flags().Lookup("show-secret") != nil {
			seen[path] = true
			if !want[path] {
				t.Errorf("unreviewed secret disclosure: %s", path)
			}
		}
	}
	for path := range want {
		if !seen[path] {
			t.Errorf("missing secret-disclosure opt-in: %s", path)
		}
	}
}

func words(s string) map[string]bool {
	result := map[string]bool{}
	for _, word := range strings.Fields(s) {
		result[word] = true
	}
	return result
}
