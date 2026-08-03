package vserver

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

func flagCompleters() map[string]cli.CompFunc {
	return map[string]cli.CompFunc{
		// Server
		"server-id": vserverclient.CompleteServerIDs,
		// Volume
		"volume-id":         vserverclient.CompleteVolumeIDs,
		"volume-type-id":    vserverclient.CompleteVolumeTypeIDs,
		"root-disk-type-id": vserverclient.CompleteVolumeTypeIDs,
		"data-disk-type-id": vserverclient.CompleteVolumeTypeIDs,
		// VPC / networking
		"vpc-id":     vserverclient.CompleteVPCIDs,
		"network-id": vserverclient.CompleteVPCIDs,
		"subnet-id":  vserverclient.CompleteSubnetIDs,
		// Security group
		"secgroup-id":    vserverclient.CompleteSecgroupIDs,
		"security-group": vserverclient.CompleteSecgroupIDs,
		// Image
		"image-id": vserverclient.CompleteImageIDs,
		// Zone
		"zone-id": vserverclient.CompleteZoneIDs,
		// SSH key
		"sshkey-id":  vserverclient.CompleteSSHKeyIDs,
		"ssh-key-id": vserverclient.CompleteSSHKeyIDs,
		// Floating IP
		"floating-ip-id": vserverclient.CompleteFloatingIPIDs,
		// Network interface
		"network-interface-id": vserverclient.CompleteNetworkInterfaceIDs,
		// Placement group
		"placement-group-id": vserverclient.CompletePlacementGroupIDs,
		"server-group-id":    vserverclient.CompletePlacementGroupIDs,
		// DHCP
		"dhcp-option-id": vserverclient.CompleteDhcpOptionIDs,
		// User image
		"user-image-id": vserverclient.CompleteUserImageIDs,
		// Tag
		"key": vserverclient.CompleteTagKeys,
		// Security group rule enums
		"direction":  cli.FlagValues("ingress", "egress"),
		"protocol":   cli.FlagValues("tcp", "udp", "icmp", "any"),
		"ether-type": cli.FlagValues("IPv4", "IPv6"),
		// SSH key file filter
		"public-key-file": func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"pub"}, cobra.ShellCompDirectiveFilterFileExt
		},
	}
}

// registerCompletions walks all vserver subcommands recursively and wires value
// completers for every flag whose name appears in flagCompleters. Commands that
// already registered a completion for a flag (inline in their own init) are left
// untouched — cobra silently ignores duplicate registrations via the ignored error.
func registerCompletions() {
	completers := flagCompleters()
	walkCommands(VServerCmd, func(c *cobra.Command) {
		for name, fn := range completers {
			if c.Flags().Lookup(name) != nil {
				_ = c.RegisterFlagCompletionFunc(name, fn)
			}
		}
	})
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	for _, c := range cmd.Commands() {
		fn(c)
		walkCommands(c, fn)
	}
}
