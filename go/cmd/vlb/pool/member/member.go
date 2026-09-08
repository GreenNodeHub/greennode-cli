package member

import (
	"github.com/spf13/cobra"
)

// MemberCmd is the parent command for pool member subcommands.
var MemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage the members of a pool",
	Long:  "List and replace the backend members of a load balancer pool.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	MemberCmd.AddCommand(listCmd)
	MemberCmd.AddCommand(updateCmd)
}
