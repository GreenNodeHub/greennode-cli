package dhcp

import (
	"github.com/spf13/cobra"
)

// DhcpCmd is the parent command for all DHCP option subcommands.
var DhcpCmd = &cobra.Command{
	Use:   "dhcp",
	Short: "Manage DHCP options",
	Long:  "Create, list, inspect, and delete DHCP option sets, and manage their VPC associations.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	DhcpCmd.AddCommand(listCmd)
	DhcpCmd.AddCommand(getCmd)
	DhcpCmd.AddCommand(createCmd)
	DhcpCmd.AddCommand(listVpcsCmd)
	DhcpCmd.AddCommand(associateVpcCmd)
	DhcpCmd.AddCommand(deleteCmd)
}
