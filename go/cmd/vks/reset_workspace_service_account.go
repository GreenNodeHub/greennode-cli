package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var resetWorkspaceServiceAccountCmd = &cobra.Command{
	Use:   "reset-workspace-service-account",
	Short: "Reset the service account for the current VKS workspace",
	RunE:  runResetWorkspaceServiceAccount,
}

func init() {
	f := resetWorkspaceServiceAccountCmd.Flags()
	f.Bool("dry-run", false, "Preview the service-account reset without executing")
	f.Bool("force", false, "Skip confirmation prompt")
}

func runResetWorkspaceServiceAccount(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if dryRun {
		cli.PrintDryRun("reset", "service account for the current VKS workspace", map[string]any{})
		return nil
	}
	if !cli.Confirm(force, "Reset the service account for the current VKS workspace?") {
		fmt.Println("Aborted.")
		return cli.ConfirmationError()
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Post("/v1/workspace/reset-service-account", nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
