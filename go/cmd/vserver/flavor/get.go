package flavor

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{Use: "get", Short: "Get a flavor", RunE: runGet}

func init() {
	getCmd.Flags().String("flavor-id", "", "Flavor ID (required)")
	if err := getCmd.MarkFlagRequired("flavor-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "flavor-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	flavorID, _ := cmd.Flags().GetString("flavor-id")
	if err := validator.ValidateID(flavorID, "flavor-id"); err != nil {
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
	result, err := apiClient.Get(fmt.Sprintf("/v1/%s/flavors/%s", projectID, flavorID), nil)
	if err != nil {
		return fmt.Errorf("failed to get flavor %s: %w", flavorID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
