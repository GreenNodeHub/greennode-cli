package secgroup

import (
	"github.com/greennodehub/greennode-cli/cmd/vserver/secgroup/rule"
	"github.com/spf13/cobra"
)

// SecgroupCmd is the parent command for all security group subcommands.
var SecgroupCmd = &cobra.Command{
	Use:   "secgroup",
	Short: "Manage security groups",
	Long:  "Create, list, and delete security groups and their rules.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	SecgroupCmd.AddCommand(listCmd)
	SecgroupCmd.AddCommand(getCmd)
	SecgroupCmd.AddCommand(createCmd)
	SecgroupCmd.AddCommand(deleteCmd)
	SecgroupCmd.AddCommand(rule.RuleCmd)
}
