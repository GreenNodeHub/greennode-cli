package server

import (
	"github.com/spf13/cobra"
)

// ServerCmd is the parent command for all server subcommands.
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage vServer instances",
	Long:  "Create, list, get, and manage vServer instances.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	ServerCmd.AddCommand(listCmd)
	ServerCmd.AddCommand(getCmd)
	ServerCmd.AddCommand(createCmd)
	ServerCmd.AddCommand(startCmd)
	ServerCmd.AddCommand(stopCmd)
	ServerCmd.AddCommand(rebootCmd)
	ServerCmd.AddCommand(resizeCmd)
	ServerCmd.AddCommand(updateSecgroupCmd)
	ServerCmd.AddCommand(tagKeyCmd)
	ServerCmd.AddCommand(tagValueCmd)
	ServerCmd.AddCommand(createImageCmd)
	ServerCmd.AddCommand(attachFloatingIPCmd)
	ServerCmd.AddCommand(detachFloatingIPCmd)
	ServerCmd.AddCommand(attachInternalInterfaceCmd)
	ServerCmd.AddCommand(detachInternalInterfaceCmd)
	ServerCmd.AddCommand(attachExternalInterfaceCmd)
	ServerCmd.AddCommand(detachExternalInterfaceCmd)
	ServerCmd.AddCommand(listInterfacesCmd)
	ServerCmd.AddCommand(deleteCmd)
}
