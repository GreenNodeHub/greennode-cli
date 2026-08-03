package floatingip

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a floating IP",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("floating-ip-id", "", "Floating IP ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the floating IP deletion without executing") //nolint:errcheck
	deleteCmd.MarkFlagRequired("floating-ip-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ipID, _ := cmd.Flags().GetString("floating-ip-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(ipID, "floating-ip-id"); err != nil {
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

	fmt.Println("The following floating IP will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", ipID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this floating IP?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/wanIps/%s", projectID, ipID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete floating IP %s: %w", ipID, err)
	}

	return outputResult(cmd, cfg, result)
}
