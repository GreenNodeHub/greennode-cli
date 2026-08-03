package dhcp

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a DHCP option",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("dhcp-option-id", "", "DHCP option ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the DHCP option deletion without executing")
	deleteCmd.MarkFlagRequired("dhcp-option-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	dhcpID, _ := cmd.Flags().GetString("dhcp-option-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(dhcpID, "dhcp-option-id"); err != nil {
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

	fmt.Println("The following DHCP option will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", dhcpID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this DHCP option?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/dhcp_option/%s", projectID, dhcpID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete DHCP option %s: %w", dhcpID, err)
	}

	return outputResult(cmd, cfg, result)
}
