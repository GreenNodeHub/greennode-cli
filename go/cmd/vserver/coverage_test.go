package vserver

import (
	"sort"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestEveryCoreVServerMutationDryRunIsOffline(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, name := range []string{"GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
		t.Setenv(name, "")
	}
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })

	ancillary := map[string]bool{}
	for _, op := range ancillaryOperations() {
		ancillary[strings.Join(append(op.Parents, op.Use), " ")] = true
	}
	commands := map[string]*cobra.Command{}
	collectVServerLeafCommands(VServerCmd, nil, commands)
	covered := 0
	for route, command := range commands {
		if ancillary[route] || command.Flags().Lookup("dry-run") == nil {
			continue
		}
		covered++
		t.Run(route, func(t *testing.T) {
			resetVServerCoverageFlags(t, command)
			t.Cleanup(func() { resetVServerCoverageFlags(t, command) })
			command.Flags().VisitAll(func(flag *pflag.Flag) {
				if len(flag.Annotations[cobra.BashCompOneRequiredFlag]) == 0 {
					return
				}
				if err := command.Flags().Set(flag.Name, vserverCoverageFlagValue(flag)); err != nil {
					t.Fatalf("set required --%s: %v", flag.Name, err)
				}
			})
			applyVServerCoverageOverrides(t, route, command)
			if err := command.Flags().Set("dry-run", "true"); err != nil {
				t.Fatal(err)
			}
			cli.SetNonInteractive(true)
			if err := command.RunE(command, nil); err != nil {
				t.Fatalf("offline dry-run failed: %v", err)
			}
			if err := cli.ConfirmationError(); err != nil {
				t.Fatalf("dry-run attempted confirmation: %v", err)
			}
		})
	}
	if covered != 74 {
		t.Fatalf("core vServer mutations exercised = %d, want 74", covered)
	}
}

func TestCoreVServerStateClassificationCounts(t *testing.T) {
	ancillary := map[string]bool{}
	for _, op := range ancillaryOperations() {
		ancillary[strings.Join(append(op.Parents, op.Use), " ")] = true
	}
	commands := map[string]*cobra.Command{}
	collectVServerLeafCommands(VServerCmd, nil, commands)
	reads := make([]string, 0)
	mutations := make([]string, 0)
	for route, command := range commands {
		if ancillary[route] {
			continue
		}
		if command.Flags().Lookup("dry-run") == nil {
			reads = append(reads, route)
		} else {
			mutations = append(mutations, route)
		}
	}
	sort.Strings(reads)
	sort.Strings(mutations)
	if len(reads) != 69 || len(mutations) != 74 {
		t.Fatalf("core vServer classification = %d reads and %d mutations, want 69 and 74; reads=%q; mutations=%q", len(reads), len(mutations), reads, mutations)
	}
}

func collectVServerLeafCommands(command *cobra.Command, parents []string, commands map[string]*cobra.Command) {
	path := append(parents, command.Name())
	children := command.Commands()
	if len(children) == 0 {
		if len(path) > 1 {
			commands[strings.Join(path[1:], " ")] = command
		}
		return
	}
	for _, child := range children {
		collectVServerLeafCommands(child, path, commands)
	}
}

func resetVServerCoverageFlags(t *testing.T, command *cobra.Command) {
	t.Helper()
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if slice, ok := flag.Value.(pflag.SliceValue); ok {
			if err := slice.Replace(nil); err != nil {
				t.Fatalf("reset --%s: %v", flag.Name, err)
			}
			flag.Changed = false
			return
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset --%s: %v", flag.Name, err)
		}
		flag.Changed = false
	})
}

func vserverCoverageFlagValue(flag *pflag.Flag) string {
	switch flag.Value.Type() {
	case "bool":
		return "true"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "1"
	}
	switch flag.Name {
	case "name", "new-name":
		return "resource-a"
	case "description":
		return "Valid description"
	case "cidr":
		return "10.0.0.0/24"
	case "ip", "private-ip":
		return "10.0.0.10"
	case "protocol":
		return "TCP"
	case "port", "port-min", "port-max":
		return "80"
	default:
		return "value-1"
	}
}

func applyVServerCoverageOverrides(t *testing.T, route string, command *cobra.Command) {
	t.Helper()
	overrides := map[string]map[string]string{
		"server create":                 {"name": "server-a", "flavor-id": "flavor-1", "image-id": "image-1", "network-id": "network-1", "subnet-id": "subnet-1", "root-disk-type-id": "type-1", "ssh-key-id": "key-1"},
		"server migrate":                {"action": "SNAPSHOT"},
		"server resize":                 {"flavor-id": "flavor-1"},
		"server update-secgroup":        {"security-group": "secgroup-1"},
		"volume create":                 {"name": "volume-a", "size": "20", "volume-type-id": "type-1"},
		"volume change-device-type":     {"action": "SNAPSHOT"},
		"volume resize":                 {"size": "30", "volume-type-id": "type-1"},
		"vpc update":                    {"name": "vpc-a"},
		"subnet update":                 {"name": "subnet-a"},
		"secgroup update":               {"name": "secgroup-a"},
		"secgroup rule create":          {"direction": "ingress", "ether-type": "IPv4", "remote-ip-prefix": "192.0.2.0/24"},
		"secgroup rule update":          {"description": "Valid description"},
		"network-interface edit":        {"description": "Valid description"},
		"network-interface update-tags": {"tag": "environment=test"},
		"user-image update-tags":        {"tag": "environment=test"},
		"server snapshot-policy create": {"name": "snapshot-a"},
		"server snapshot create":        {"name": "snapshot-a"},
		"volume snapshot create":        {"name": "snapshot-a"},
		"placement-group create":        {"policy-id": "policy-1"},
		"placement-group update":        {"name": "placement-a"},
		"sshkey import":                 {"public-key": "ssh-ed25519 fixture-public fixture-key"},
		"dhcp create":                   {"name": "dhcp-a"},
		"dhcp associate-vpc":            {"dhcp-option-id": "fixture-dhcp"},
		"subnet create":                 {"zone-id": "fixture-zone"},
	}
	for name, value := range overrides[route] {
		if command.Flags().Lookup(name) == nil {
			t.Fatalf("coverage override references absent --%s", name)
		}
		if err := command.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}

func TestDocumentedVServerOperationRoutesAreRegistered(t *testing.T) {
	expected := []string{
		"server list", "server create", "server get-external-interface", "server list-by-subnet", "server get", "server delete",
		"server list-actions", "server complete-migration", "server get-console-log", "server get-console-url",
		"server attach-external-interface", "server detach-external-interface", "server attach-internal-interface", "server detach-internal-interface",
		"server attach-internal-floating", "server detach-internal-floating", "server migrate", "server list-interfaces", "server reboot",
		"server rename", "server resize", "server list-secgroups", "server snapshot-policy create", "server snapshot-policy delete",
		"server snapshot-policy disable", "server snapshot-policy enable", "server snapshot-policy update", "server snapshot-policy list-shared",
		"server snapshot-policy revoke-shared", "server snapshot list", "server snapshot create", "server snapshot detail",
		"server snapshot rollback", "server snapshot delete", "server start", "server start-migration", "server stop",
		"server update-secgroup", "server attach-floating-ip", "server detach-floating-ip",

		"volume list", "volume create", "volume list-by-server", "volume get-boot", "volume get", "volume delete",
		"volume change-device-type", "volume history", "volume mapping", "volume rename", "volume resize", "volume attach", "volume detach",
		"volume snapshot list", "volume snapshot create", "volume snapshot rollback", "volume snapshot delete",
		"volume snapshot-policy delete", "volume snapshot-policy detail", "volume snapshot-policy update",
		"volume snapshot-policy disable", "volume snapshot-policy enable",

		"vpc list", "vpc create", "vpc get", "vpc delete", "vpc list-active", "vpc update", "vpc enable-dns", "dhcp associate-vpc",
		"subnet list", "subnet create", "subnet get", "subnet delete", "subnet update", "subnet create-secondary", "subnet delete-secondary",
		"secgroup list", "secgroup create", "secgroup get", "secgroup update", "secgroup delete", "secgroup list-servers",
		"secgroup rule list", "secgroup rule create", "secgroup rule list-samples", "secgroup rule get", "secgroup rule update", "secgroup rule delete",
		"flavor list-customs", "flavor list-custom-clusters", "flavor list", "flavor list-cluster", "flavor get", "flavor list-by-zone",
		"flavor-zone list-codes", "flavor-zone list-customs", "flavor-zone list-custom-clusters", "flavor-zone list-families",
		"flavor-zone list-family-clusters", "flavor-zone list-products", "flavor-zone list-product", "flavor-zone get",

		"image list", "volume-type list", "volume-type list-all", "volume-type list-zones", "volume-type get-default", "volume-type get", "volume-type get-zone",
	}

	routes := make(map[string]bool)
	collectLeafRoutes(VServerCmd, nil, routes)
	for _, route := range expected {
		if !routes[route] {
			t.Errorf("documented operation route %q is not registered", route)
		}
	}
}

func TestCoreVServerMutationsExposeDryRun(t *testing.T) {
	writeRoutes := []string{
		"server create", "server delete", "server complete-migration", "server attach-external-interface", "server detach-external-interface",
		"server attach-internal-interface", "server detach-internal-interface", "server attach-internal-floating", "server detach-internal-floating",
		"server migrate", "server reboot", "server rename", "server resize", "server snapshot-policy create", "server snapshot-policy delete",
		"server snapshot-policy disable", "server snapshot-policy enable", "server snapshot-policy update", "server snapshot-policy revoke-shared",
		"server snapshot create", "server snapshot rollback", "server snapshot delete", "server start", "server start-migration", "server stop",
		"server update-secgroup", "server attach-floating-ip", "server detach-floating-ip",
		"volume create", "volume delete", "volume change-device-type", "volume rename", "volume resize", "volume attach", "volume detach",
		"volume snapshot create", "volume snapshot rollback", "volume snapshot delete", "volume snapshot-policy delete", "volume snapshot-policy update",
		"volume snapshot-policy disable", "volume snapshot-policy enable",
		"vpc create", "vpc delete", "vpc update", "vpc enable-dns", "dhcp associate-vpc",
		"subnet create", "subnet delete", "subnet update", "subnet create-secondary", "subnet delete-secondary",
		"secgroup create", "secgroup update", "secgroup delete", "secgroup rule create", "secgroup rule update", "secgroup rule delete",
	}
	for _, route := range writeRoutes {
		cmd, _, err := VServerCmd.Find(strings.Fields(route))
		if err != nil || cmd == nil {
			t.Errorf("mutation route %q not found: %v", route, err)
			continue
		}
		if cmd.Flags().Lookup("dry-run") == nil {
			t.Errorf("mutation route %q does not expose --dry-run", route)
		}
	}
}

func TestCoreVServerDeletesExposeForce(t *testing.T) {
	deleteRoutes := []string{
		"server delete", "server snapshot-policy delete", "server snapshot-policy revoke-shared", "server snapshot delete",
		"volume delete", "volume snapshot delete", "volume snapshot-policy delete",
		"vpc delete", "subnet delete", "subnet delete-secondary", "secgroup delete", "secgroup rule delete",
	}
	for _, route := range deleteRoutes {
		cmd, _, err := VServerCmd.Find(strings.Fields(route))
		if err != nil || cmd == nil {
			t.Errorf("delete route %q not found: %v", route, err)
			continue
		}
		if cmd.Flags().Lookup("force") == nil {
			t.Errorf("delete route %q does not expose --force", route)
		}
	}
}

func collectLeafRoutes(cmd *cobra.Command, parent []string, routes map[string]bool) {
	path := append(parent, cmd.Name())
	children := cmd.Commands()
	if len(children) == 0 {
		if len(path) > 1 {
			routes[strings.Join(path[1:], " ")] = true
		}
		return
	}
	for _, child := range children {
		collectLeafRoutes(child, path, routes)
	}
}
