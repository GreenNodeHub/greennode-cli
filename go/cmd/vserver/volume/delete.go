package volume

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a volume",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("volume-id", "", "Volume ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the volume deletion without executing")
	deleteCmd.MarkFlagRequired("volume-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
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

	response, err := apiClient.Get(fmt.Sprintf("/v2/%s/volumes/%s", projectID, volumeID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch volume %s: %w", volumeID, err)
	}

	volumeData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type from API: %T", response)
	}

	if err := printVolumeDeletePreview(volumeData["data"]); err != nil {
		return err
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this volume?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/volumes/%s", projectID, volumeID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete volume %s: %w", volumeID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printVolumeDeletePreview(volume interface{}) error {
	v, ok := volume.(map[string]interface{})
	if !ok || v == nil {
		return fmt.Errorf("could not parse volume details from API response (type: %T)", volume)
	}

	fmt.Println("The following volume will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:     %v\n", v["id"])
	fmt.Printf("  Name:   %v\n", v["name"])
	fmt.Printf("  Size:   %v GiB\n", v["size"])
	fmt.Printf("  Status: %v\n", v["status"])
	fmt.Println()
	fmt.Println("This action is irreversible.")
	return nil
}
