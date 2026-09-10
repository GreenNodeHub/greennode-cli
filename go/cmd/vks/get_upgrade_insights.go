package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getUpgradeInsightsCmd = &cobra.Command{
	Use:   "get-upgrade-insights",
	Short: "Get upgrade insights for a VKS cluster",
	RunE:  runGetUpgradeInsights,
}

func init() {
	f := getUpgradeInsightsCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Int("page", 0, "Page number (0-based)")
	f.Int("page-size", 10, "Page size")

	getUpgradeInsightsCmd.MarkFlagRequired("cluster-id")
}

func runGetUpgradeInsights(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validatePage(page, pageSize); err != nil {
		return err
	}

	params := map[string]string{}
	if cmd.Flags().Changed("page") {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if cmd.Flags().Changed("page-size") {
		params["pageSize"] = fmt.Sprintf("%d", pageSize)
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s/upgrade-insight", clusterID), params)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
