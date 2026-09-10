package vks

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var createWorkspaceCmd = &cobra.Command{
	Use:   "create-workspace",
	Short: "Create a VKS workspace for the current user",
	RunE:  runCreateWorkspace,
}

func init() {
	createWorkspaceCmd.Flags().Bool("dry-run", false, "Preview the workspace creation without executing")
}

func runCreateWorkspace(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		cli.PrintDryRun("create", "VKS workspace", map[string]any{})
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Post("/v1/workspace", nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
