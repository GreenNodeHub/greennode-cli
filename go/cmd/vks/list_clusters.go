package vks

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listClustersCmd = &cobra.Command{
	Use:   "list-clusters",
	Short: "List all VKS clusters",
	RunE:  runListClusters,
}

func init() {
	listClustersCmd.Flags().Int("page", -1, "Page number (0-based)")
	listClustersCmd.Flags().Int("page-size", 50, "Number of items per page")
	listClustersCmd.Flags().Bool("no-paginate", false, "Disable auto-pagination")
}

func runListClusters(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	noPaginate, _ := cmd.Flags().GetBool("no-paginate")

	var result interface{}

	if page >= 0 || noPaginate {
		// Single page request
		if page < 0 {
			page = 0
		}
		params := map[string]string{
			"page":     fmt.Sprintf("%d", page),
			"pageSize": fmt.Sprintf("%d", pageSize),
		}
		result, err = apiClient.Get("/v1/clusters", params)
	} else {
		// Auto-pagination: fetch all pages
		result, err = apiClient.GetAllPages("/v1/clusters", pageSize)
	}

	if err != nil {
		return err
	}

	return outputResult(cmd, result)
}
