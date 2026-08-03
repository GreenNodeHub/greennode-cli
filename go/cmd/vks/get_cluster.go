package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getClusterCmd = &cobra.Command{
	Use:   "get-cluster",
	Short: "Get details of a VKS cluster",
	RunE:  runGetCluster,
}

func init() {
	getClusterCmd.Flags().String("cluster-id", "", "Cluster ID (required)")
	getClusterCmd.MarkFlagRequired("cluster-id")
}

func runGetCluster(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
