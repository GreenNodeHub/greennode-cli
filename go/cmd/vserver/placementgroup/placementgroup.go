package placementgroup

import (
	"github.com/spf13/cobra"
)

// PlacementGroupCmd is the parent command for all placement group subcommands.
var PlacementGroupCmd = &cobra.Command{
	Use:   "placement-group",
	Short: "Manage placement groups (server groups)",
	Long:  "List, create, edit, and delete placement groups, and list available policies.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	PlacementGroupCmd.AddCommand(listCmd)
	PlacementGroupCmd.AddCommand(listPoliciesCmd)
	PlacementGroupCmd.AddCommand(createCmd)
	PlacementGroupCmd.AddCommand(updateCmd)
	PlacementGroupCmd.AddCommand(deleteCmd)
}
