package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a vServer instance",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("server-id", "", "Server ID (required)")
	f.Bool("delete-all-volumes", false, "Delete all volumes associated with the server")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the server deletion without executing")
	deleteCmd.MarkFlagRequired("server-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	deleteAllVolumes, _ := cmd.Flags().GetBool("delete-all-volumes")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

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

	response, err := apiClient.Get(fmt.Sprintf("/v2/%s/servers/%s", projectID, serverID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch server %s: %w", serverID, err)
	}

	serverData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type from API: %T", response)
	}

	if err := printDeletePreview(serverData["data"], deleteAllVolumes); err != nil {
		return err
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this server?") {
		fmt.Println("Aborted.")
		return nil
	}

	body := map[string]interface{}{
		"deleteAllVolumes": deleteAllVolumes,
	}

	result, err := apiClient.DeleteWithBody(fmt.Sprintf("/v2/%s/servers/%s", projectID, serverID), body)
	if err != nil {
		return fmt.Errorf("failed to delete server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printDeletePreview(server interface{}, deleteAllVolumes bool) error {
	s, ok := server.(map[string]interface{})
	if !ok || s == nil {
		return fmt.Errorf("could not parse server details from API response (type: %T)", server)
	}

	fmt.Println("The following server will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:     %v\n", s["uuid"])
	fmt.Printf("  Name:   %v\n", s["name"])
	fmt.Printf("  Status: %v\n", s["status"])
	if deleteAllVolumes {
		fmt.Printf("  Delete all volumes: true\n")
	}
	fmt.Println()
	fmt.Println("This action is irreversible.")
	return nil
}
