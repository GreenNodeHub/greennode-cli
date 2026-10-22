package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var unregisterFleetCmd = &cobra.Command{
	Use:   "unregister-fleet",
	Short: "Unregister a VKS cluster from fleet management",
	RunE:  runUnregisterFleet,
}

func init() {
	f := unregisterFleetCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Bool("dry-run", false, "Preview fleet unregistration without executing")
	f.Bool("force", false, "Skip confirmation prompt")

	unregisterFleetCmd.MarkFlagRequired("cluster-id")
}

func runUnregisterFleet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if dryRun {
		cli.PrintDryRun("unregister", fmt.Sprintf("fleet for cluster %s", clusterID), map[string]any{})
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Unregister cluster %s from fleet management?", clusterID)) {
		fmt.Println("Aborted.")
		return cli.ConfirmationError()
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Put(fmt.Sprintf("/v1/clusters/%s/unregister-fleet", clusterID), nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
