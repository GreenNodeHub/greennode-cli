package vserver

import (
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func previewCommand(t *testing.T, route string, flags map[string]string) *cobra.Command {
	t.Helper()
	cmd, rest, err := VServerCmd.Find(strings.Fields(route))
	if err != nil || len(rest) != 0 {
		t.Fatalf("command %s: %v", route, err)
	}
	resetVServerCoverageFlags(t, cmd)
	t.Cleanup(func() { resetVServerCoverageFlags(t, cmd) })
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestLegacyPreviewRunsBusinessValidation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct {
		route string
		flags map[string]string
		want  string
	}{
		{"subnet create", map[string]string{"vpc-id": "fixture-vpc", "cidr": "invalid", "zone-id": "fixture-zone"}, "CIDR"},
		{"dhcp create", map[string]string{"name": "fixture-name", "dns-server": "invalid"}, "DNS server IP"},
		{"dhcp associate-vpc", map[string]string{"vpc-id": "fixture-vpc", "dhcp-option-id": "fixture-dhcp", "detach": "true"}, "cannot be combined"},
		{"network-interface create", map[string]string{"name": "fixture-name", "zone-id": "fixture-zone", "tag": "invalid"}, "key=value"},
		{"server attach-internal-interface", map[string]string{"server-id": "fixture-server", "subnet-id": "fixture-subnet", "ip": "invalid"}, "valid IP"},
		{"secgroup rule create", map[string]string{"secgroup-id": "fixture-group", "protocol": "tcp", "direction": "invalid"}, "direction"},
		{"secgroup rule create", map[string]string{"secgroup-id": "fixture-group", "protocol": "tcp", "direction": "ingress", "port-range-min": "90", "port-range-max": "80"}, "port-range-min"},
		{"secgroup rule create", map[string]string{"secgroup-id": "fixture-group", "protocol": "tcp", "direction": "ingress", "remote-ip-prefix": "invalid"}, "remote-ip-prefix"},
		{"placement-group update", map[string]string{"placement-group-id": "fixture-group"}, "nothing to update"},
	} {
		t.Run(tc.route+"/"+tc.want, func(t *testing.T) {
			cmd := previewCommand(t, tc.route, tc.flags)
			if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v; want %s", err, tc.want)
			}
		})
	}
}

func TestLegacyPreviewIncludesHTTPBodyAndDefaults(t *testing.T) {
	for _, tc := range []struct {
		route string
		flags map[string]string
		want  []string
	}{
		{"server delete", map[string]string{"server-id": "fixture-server"}, []string{"DELETE /v2/<project-id>/servers/fixture-server", "deleteAllVolumes: false"}},
		{"server attach-internal-interface", map[string]string{"server-id": "fixture-server", "subnet-id": "fixture-subnet"}, []string{"POST /v2/<project-id>/servers/fixture-server/internal-network-interfaces", "subnetId:fixture-subnet", "ip:<nil>"}},
		{"dhcp create", map[string]string{"name": "fixture-name"}, []string{"POST /v2/<project-id>/dhcp_option", "dnsServers:", "name: fixture-name"}},
	} {
		t.Run(tc.route, func(t *testing.T) {
			cmd := previewCommand(t, tc.route, tc.flags)
			out := testutil.CaptureStdout(t, func() {
				if err := cmd.RunE(cmd, nil); err != nil {
					t.Fatal(err)
				}
			})
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("missing %q: %s", want, out)
				}
			}
		})
	}
}
