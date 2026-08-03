package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Reboot a vServer instance",
	RunE:  runReboot,
}

func init() {
	f := rebootCmd.Flags()
	f.String("server-id", "", "Server ID (required)")
	f.Bool("dry-run", false, "Preview the reboot action without executing")
	f.Bool("force", false, "Skip confirmation prompt")
	rebootCmd.MarkFlagRequired("server-id")
}

func runReboot(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
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

	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	fmt.Println("The following server will be rebooted:")
	fmt.Println()
	fmt.Printf("  ID: %s\n", serverID)
	fmt.Println()
	fmt.Println("This will interrupt the server.")

	if dryRun {
		cli.DryRunNotice("reboot")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to reboot this server?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/servers/%s/reboot", projectID, serverID), nil)
	if err != nil {
		return fmt.Errorf("failed to reboot server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, transformServerResult(result))
}
