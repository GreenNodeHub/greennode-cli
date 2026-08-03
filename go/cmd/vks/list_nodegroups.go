package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listNodegroupsCmd = &cobra.Command{
	Use:   "list-nodegroups",
	Short: "List node groups in a VKS cluster",
	RunE:  runListNodegroups,
}

func init() {
	f := listNodegroupsCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Int("page", -1, "Page number (0-based)")
	f.Int("page-size", 50, "Number of items per page")
	f.Bool("no-paginate", false, "Disable auto-pagination")

	listNodegroupsCmd.MarkFlagRequired("cluster-id")
}

func runListNodegroups(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	noPaginate, _ := cmd.Flags().GetBool("no-paginate")

	path := fmt.Sprintf("/v1/clusters/%s/node-groups", clusterID)
	var result interface{}

	if page >= 0 || noPaginate {
		if page < 0 {
			page = 0
		}
		params := map[string]string{
			"page":     fmt.Sprintf("%d", page),
			"pageSize": fmt.Sprintf("%d", pageSize),
		}
		result, err = apiClient.Get(path, params)
	} else {
		result, err = apiClient.GetAllPages(path, pageSize)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
