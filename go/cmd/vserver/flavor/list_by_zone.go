package flavor

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listByZoneCmd = &cobra.Command{Use: "list-by-zone", Short: "List flavors in a flavor zone", RunE: runListByZone}

func init() {
	listByZoneCmd.Flags().String("flavor-zone-id", "", "Flavor zone ID (required)")
	if err := listByZoneCmd.MarkFlagRequired("flavor-zone-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "flavor-zone-id", err))
	}
}

func runListByZone(cmd *cobra.Command, args []string) error {
	zoneID, _ := cmd.Flags().GetString("flavor-zone-id")
	if err := validator.ValidateID(zoneID, "flavor-zone-id"); err != nil {
		return err
	}
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v1/%s/%s/flavors", projectID, zoneID), nil)
	if err != nil {
		return fmt.Errorf("failed to list flavors for zone %s: %w", zoneID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
