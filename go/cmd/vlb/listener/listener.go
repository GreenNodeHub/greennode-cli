package listener

import (
	"github.com/greennodehub/greennode-cli/cmd/vlb/listener/policy"
	"github.com/spf13/cobra"
)

// ListenerCmd is the parent command for load balancer listener subcommands.
var ListenerCmd = &cobra.Command{
	Use:   "listener",
	Short: "Manage load balancer listeners",
	Long:  "Create, inspect, update and delete listeners and their L7 policies.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	ListenerCmd.AddCommand(listCmd)
	ListenerCmd.AddCommand(getCmd)
	ListenerCmd.AddCommand(createCmd)
	ListenerCmd.AddCommand(updateCmd)
	ListenerCmd.AddCommand(deleteCmd)
	ListenerCmd.AddCommand(policy.PolicyCmd)
}
