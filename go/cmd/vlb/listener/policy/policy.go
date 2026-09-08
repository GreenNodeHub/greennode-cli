package policy

import (
	"github.com/spf13/cobra"
)

// PolicyCmd is the parent command for listener L7 policy subcommands.
var PolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage the L7 policies of a listener",
	Long:  "Create, inspect, update, reorder and delete the L7 routing policies of a listener.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	PolicyCmd.AddCommand(listCmd)
	PolicyCmd.AddCommand(getCmd)
	PolicyCmd.AddCommand(createCmd)
	PolicyCmd.AddCommand(updateCmd)
	PolicyCmd.AddCommand(deleteCmd)
	PolicyCmd.AddCommand(updateOrderCmd)
}
