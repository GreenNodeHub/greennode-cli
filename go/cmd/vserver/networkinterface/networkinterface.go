package networkinterface

import (
	"github.com/spf13/cobra"
)

// NetworkInterfaceCmd is the parent command for all network interface subcommands.
var NetworkInterfaceCmd = &cobra.Command{
	Use:   "network-interface",
	Short: "Manage elastic network interfaces",
	Long:  "Create, list, rename, update tags on, and delete elastic network interfaces.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	NetworkInterfaceCmd.AddCommand(listCmd)
	NetworkInterfaceCmd.AddCommand(createCmd)
	NetworkInterfaceCmd.AddCommand(updateCmd)
	NetworkInterfaceCmd.AddCommand(updateTagsCmd)
	NetworkInterfaceCmd.AddCommand(deleteCmd)
}
