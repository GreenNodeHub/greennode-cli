package vks

import (
	"github.com/spf13/cobra"
)

var listClusterVersionsCmd = &cobra.Command{
	Use:   "list-cluster-versions",
	Short: "List available Kubernetes versions for VKS clusters",
	RunE:  runListClusterVersions,
}

func runListClusterVersions(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get("/v1/cluster-versions", nil)
	if err != nil {
		return err
	}

	return outputResult(cmd, result)
}
