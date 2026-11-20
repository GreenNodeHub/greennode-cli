package pool

import (
	"github.com/greennodehub/greennode-cli/cmd/vlb/pool/member"
	"github.com/spf13/cobra"
)

// PoolCmd is the parent command for load balancer pool subcommands.
var PoolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Manage load balancer pools",
	Long:  "Create, inspect, update and delete backend pools and their members.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	PoolCmd.AddCommand(listCmd)
	PoolCmd.AddCommand(getCmd)
	PoolCmd.AddCommand(createCmd)
	PoolCmd.AddCommand(updateCmd)
	PoolCmd.AddCommand(deleteCmd)
	PoolCmd.AddCommand(getHealthMonitorCmd)
	PoolCmd.AddCommand(member.MemberCmd)
}
