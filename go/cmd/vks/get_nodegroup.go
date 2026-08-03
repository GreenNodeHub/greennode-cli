package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getNodegroupCmd = &cobra.Command{
	Use:   "get-nodegroup",
	Short: "Get details of a node group",
	RunE:  runGetNodegroup,
}

func init() {
	f := getNodegroupCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Node group ID (required)")

	getNodegroupCmd.MarkFlagRequired("cluster-id")
	getNodegroupCmd.MarkFlagRequired("nodegroup-id")
}

func runGetNodegroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(
		fmt.Sprintf("/v1/clusters/%s/node-groups/%s", clusterID, nodegroupID), nil,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
