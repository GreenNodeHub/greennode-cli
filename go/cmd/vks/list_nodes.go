package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listNodesCmd = &cobra.Command{
	Use:   "list-nodes",
	Short: "List nodes in a node group",
	RunE:  runListNodes,
}

func init() {
	f := listNodesCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Node group ID (required)")
	f.Int("page", 0, "Page number (0-based)")
	f.Int("page-size", 50, "Page size")

	listNodesCmd.MarkFlagRequired("cluster-id")
	listNodesCmd.MarkFlagRequired("nodegroup-id")
}

func runListNodes(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}

	// VKS pagination is 0-based; only send params the user explicitly set.
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

	result, err := apiClient.Get(
		fmt.Sprintf("/v1/clusters/%s/node-groups/%s/nodes", clusterID, nodegroupID), params,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
