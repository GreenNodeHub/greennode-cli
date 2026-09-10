package placementgroup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a placement group",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("placement-group-id", "", "Placement group (server group) ID (required)")
	if err := getCmd.MarkFlagRequired("placement-group-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "placement-group-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	groupID, _ := cmd.Flags().GetString("placement-group-id")
	if err := validator.ValidateID(groupID, "placement-group-id"); err != nil {
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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/serverGroups/%s", projectID, groupID), nil)
	if err != nil {
		return fmt.Errorf("failed to get placement group %s: %w", groupID, err)
	}
	return outputGroupList(cmd, cfg, result)
}
