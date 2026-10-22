package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var stopPOCCmd = &cobra.Command{
	Use:   "stop-poc",
	Short: "Stop proof-of-concept mode for a VKS cluster",
	RunE:  runStopPOC,
}

func init() {
	f := stopPOCCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Bool("dry-run", false, "Preview stopping proof-of-concept mode without executing")
	f.Bool("force", false, "Skip confirmation prompt")

	stopPOCCmd.MarkFlagRequired("cluster-id")
}

func runStopPOC(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if dryRun {
		cli.PrintDryRun("stop", fmt.Sprintf("proof-of-concept mode for cluster %s", clusterID), map[string]any{})
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Stop proof-of-concept mode for cluster %s?", clusterID)) {
		fmt.Println("Aborted.")
		return cli.ConfirmationError()
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Post(fmt.Sprintf("/v1/clusters/%s/stop-poc", clusterID), nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
