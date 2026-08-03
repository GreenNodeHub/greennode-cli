package vserver

import (
	"github.com/greennodehub/greennode-cli/cmd/vserver/flavor"
	"github.com/greennodehub/greennode-cli/cmd/vserver/image"
	"github.com/greennodehub/greennode-cli/cmd/vserver/secgroup"
	"github.com/greennodehub/greennode-cli/cmd/vserver/server"
	"github.com/greennodehub/greennode-cli/cmd/vserver/subnet"
	"github.com/greennodehub/greennode-cli/cmd/vserver/volume"
	"github.com/greennodehub/greennode-cli/cmd/vserver/volumetype"
	"github.com/greennodehub/greennode-cli/cmd/vserver/vpc"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

// VServerCmd is the parent command for all vServer subcommands.
var VServerCmd = &cobra.Command{
	Use:   "vserver",
	Short: "VNG Virtual Server (vServer) commands",
	Long:  "Manage vServer instances and related resources.",
	// Reject unknown subcommands (nested groups don't error by default in cobra).
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	VServerCmd.AddCommand(server.ServerCmd)
	VServerCmd.AddCommand(volume.VolumeCmd)
	VServerCmd.AddCommand(vpc.VpcCmd)
	VServerCmd.AddCommand(subnet.SubnetCmd)
	VServerCmd.AddCommand(secgroup.SecgroupCmd)
	VServerCmd.AddCommand(flavor.FlavorCmd)
	VServerCmd.AddCommand(volumetype.VolumeTypeCmd)
	VServerCmd.AddCommand(image.ImageCmd)
	cli.RegisterService(VServerCmd)
}
