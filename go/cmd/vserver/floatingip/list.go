package floatingip

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all floating IPs",
	RunE:  runList,
}

func init() {
	listCmd.Flags().Int("page", 1, "Page number (1-based)")
	listCmd.Flags().Int("page-size", 50, "Number of items per page")
	listCmd.Flags().String("name", "", "Filter by floating IP name (substring match)")
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	filterName, _ := cmd.Flags().GetString("name")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	params := map[string]string{
		"page": fmt.Sprintf("%d", page),
		"size": fmt.Sprintf("%d", pageSize),
	}
	if filterName != "" {
		params["name"] = filterName
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/wanIps", projectID), params)
	if err != nil {
		return fmt.Errorf("failed to list floating IPs: %w", err)
	}

	return outputIPList(cmd, cfg, result)
}
