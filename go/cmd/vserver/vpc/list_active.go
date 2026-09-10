package vpc

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listActiveCmd = &cobra.Command{
	Use:   "list-active",
	Short: "List active VPCs",
	RunE:  runListActive,
}

func init() {
	listActiveCmd.Flags().Int("page", 1, "Page number (1-based)")
	listActiveCmd.Flags().Int("page-size", 50, "Number of items per page")
}

func runListActive(cmd *cobra.Command, args []string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/active", projectID), vserverclient.PaginationParams(cmd, "page", "size", 50))
	if err != nil {
		return fmt.Errorf("failed to list active VPCs: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
