package loadbalancer

import (
	"github.com/spf13/cobra"
)

// LoadBalancerCmd is the parent command for load balancer subcommands.
var LoadBalancerCmd = &cobra.Command{
	Use:     "loadbalancer",
	Aliases: []string{"lb"},
	Short:   "Manage load balancers",
	Long:    "Create, inspect, resize, clone and delete load balancers.",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	LoadBalancerCmd.AddCommand(listCmd)
	LoadBalancerCmd.AddCommand(getCmd)
	LoadBalancerCmd.AddCommand(createCmd)
	LoadBalancerCmd.AddCommand(deleteCmd)
	LoadBalancerCmd.AddCommand(resizeCmd)
	LoadBalancerCmd.AddCommand(cloneCmd)
	LoadBalancerCmd.AddCommand(configureScalingCmd)
	LoadBalancerCmd.AddCommand(updateBandwidthCmd)
	LoadBalancerCmd.AddCommand(getCloneMetadataCmd)
	LoadBalancerCmd.AddCommand(listScaleHistoryCmd)
	LoadBalancerCmd.AddCommand(listHeadersCmd)
	LoadBalancerCmd.AddCommand(listBySubnetCmd)
	LoadBalancerCmd.AddCommand(waitActiveCmd)
	LoadBalancerCmd.AddCommand(waitDeletedCmd)
}
