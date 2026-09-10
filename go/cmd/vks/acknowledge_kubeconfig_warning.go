package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var acknowledgeKubeconfigWarningCmd = &cobra.Command{
	Use:   "acknowledge-kubeconfig-warning",
	Short: "Acknowledge a kubeconfig renewal warning for a VKS cluster",
	RunE:  runAcknowledgeKubeconfigWarning,
}

func init() {
	f := acknowledgeKubeconfigWarningCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Bool("dry-run", false, "Preview the acknowledgement without executing")

	acknowledgeKubeconfigWarningCmd.MarkFlagRequired("cluster-id")
}

func runAcknowledgeKubeconfigWarning(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if dryRun {
		cli.PrintDryRun("acknowledge", fmt.Sprintf("kubeconfig renewal warning for cluster %s", clusterID), map[string]any{})
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Put(fmt.Sprintf("/v1/clusters/%s/kubeconfig/acknowledge-warning", clusterID), nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
