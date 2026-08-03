package placementgroup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a placement group",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("placement-group-id", "", "Placement group ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the placement group deletion without executing")
	deleteCmd.MarkFlagRequired("placement-group-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	groupID, _ := cmd.Flags().GetString("placement-group-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(groupID, "placement-group-id"); err != nil {
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

	fmt.Println("The following placement group will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", groupID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this placement group?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/serverGroups/%s", projectID, groupID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete placement group %s: %w", groupID, err)
	}

	return outputResult(cmd, cfg, result)
}
