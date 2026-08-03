package networkinterface

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a network interface",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("network-interface-id", "", "Network interface ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the network interface deletion without executing")
	deleteCmd.MarkFlagRequired("network-interface-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
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

	fmt.Println("The following network interface will be deleted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", interfaceID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this network interface?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/network-interfaces-elastic/%s", projectID, interfaceID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete network interface %s: %w", interfaceID, err)
	}

	return outputResult(cmd, cfg, result)
}
