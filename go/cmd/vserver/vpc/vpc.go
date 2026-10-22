package vpc

import (
	"github.com/spf13/cobra"
)

// VpcCmd is the parent command for all VPC subcommands.
var VpcCmd = &cobra.Command{
	Use:   "vpc",
	Short: "Manage VPCs (virtual private clouds)",
	Long:  "Create, list, get, and delete VPC networks.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	VpcCmd.AddCommand(listCmd)
	VpcCmd.AddCommand(getCmd)
	VpcCmd.AddCommand(createCmd)
	VpcCmd.AddCommand(deleteCmd)
}
