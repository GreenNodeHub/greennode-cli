package rule

import (
	"github.com/spf13/cobra"
)

// RuleCmd is the parent command for all security group rule subcommands.
var RuleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Manage security group rules",
	Long:  "Create, list, and delete rules within a security group.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	RuleCmd.AddCommand(listCmd)
	RuleCmd.AddCommand(getCmd)
	RuleCmd.AddCommand(createCmd)
	RuleCmd.AddCommand(deleteCmd)
}
