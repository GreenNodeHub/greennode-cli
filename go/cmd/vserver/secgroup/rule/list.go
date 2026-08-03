package rule

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules in a security group",
	RunE:  runList,
}

func init() {
	listCmd.Flags().String("secgroup-id", "", "Security group ID (required)")
	listCmd.Flags().Int("page", 1, "Page number (1-based)")
	listCmd.Flags().Int("page-size", 50, "Number of items per page")
	listCmd.MarkFlagRequired("secgroup-id")
}

func runList(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
		return err
	}

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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/secgroups/%s/secGroupRules", projectID, secgroupID), params)
	if err != nil {
		return fmt.Errorf("failed to list rules for security group %s: %w", secgroupID, err)
	}

	return outputResult(cmd, cfg, result)
}
