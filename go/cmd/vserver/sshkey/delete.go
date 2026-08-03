package sshkey

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a SSH key",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("sshkey-id", "", "SSH key ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the SSH key deletion without executing")
	deleteCmd.MarkFlagRequired("sshkey-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	sshKeyID, _ := cmd.Flags().GetString("sshkey-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(sshKeyID, "sshkey-id"); err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	fmt.Println("The following SSH key will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", sshKeyID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this SSH key?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/sshKeys/%s", projectID, sshKeyID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete SSH key %s: %w", sshKeyID, err)
	}

	return outputResult(cmd, cfg, result)
}
