package userimage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user image",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("user-image-id", "", "User image ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the user image deletion without executing")
	deleteCmd.MarkFlagRequired("user-image-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	imageID, _ := cmd.Flags().GetString("user-image-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(imageID, "user-image-id"); err != nil {
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

	fmt.Println("The following user image will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", imageID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this user image?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/user-images/%s", projectID, imageID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete user image %s: %w", imageID, err)
	}

	return outputResult(cmd, cfg, result)
}
