package flavorzone

import "github.com/spf13/cobra"

var FlavorZoneCmd = &cobra.Command{
	Use:   "flavor-zone",
	Short: "Discover flavor zones, families, and platform codes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	FlavorZoneCmd.AddCommand(listCodesCmd)
	FlavorZoneCmd.AddCommand(listCustomsCmd)
	FlavorZoneCmd.AddCommand(listCustomClustersCmd)
	FlavorZoneCmd.AddCommand(listFamiliesCmd)
	FlavorZoneCmd.AddCommand(listFamilyClustersCmd)
	FlavorZoneCmd.AddCommand(listProductsCmd)
	FlavorZoneCmd.AddCommand(listProductCmd)
	FlavorZoneCmd.AddCommand(getCmd)
}
