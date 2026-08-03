package volume

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a volume's size or change its volume type",
	Long: `Resize a volume by changing its size in GiB, its volume type, or both.

At least one of --size or --volume-type-id must be provided.

Examples:
  # Expand volume to 100 GiB
  grn vserver volume resize --volume-id vol-abc123 --size 100

  # Change volume type
  grn vserver volume resize --volume-id vol-abc123 --volume-type-id vtype-xyz789

  # Resize and change type at the same time
  grn vserver volume resize --volume-id vol-abc123 --size 200 --volume-type-id vtype-xyz789

  # Dry-run validation only
  grn vserver volume resize --volume-id vol-abc123 --size 100 --dry-run`,
	RunE: runResize,
}

func init() {
	f := resizeCmd.Flags()
	f.String("volume-id", "", "Volume ID (required)")
	f.Int("size", 0, "New volume size in GiB (must be equal to or greater than current size)")
	f.String("volume-type-id", "", "New volume type ID — run 'grn vserver volume-type list' to see options")
	f.Bool("dry-run", false, "Validate parameters without sending the resize request")

	if err := resizeCmd.MarkFlagRequired("volume-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "volume-id", err))
	}
}

func runResize(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	size, _ := cmd.Flags().GetInt("size")
	sizeSet := cmd.Flags().Changed("size")
	volumeTypeID, _ := cmd.Flags().GetString("volume-type-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}

	if size == 0 && volumeTypeID == "" {
		return fmt.Errorf("at least one of --size or --volume-type-id must be provided")
	}

	if dryRun {
		return validateResize(volumeID, size, volumeTypeID)
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(fmt.Sprintf("/v2/%s/volumes/%s", projectID, volumeID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch current volume %s: %w", volumeID, err)
	}

	currentData, err := extractVolumeData(current)
	if err != nil {
		return err
	}

	printVolumeResizePreview(currentData, size, volumeTypeID)

	if volumeTypeID == "" {
		id, ok := currentData["volumeTypeId"].(string)
		if !ok || id == "" {
			return fmt.Errorf("could not extract volumeTypeId from volume response")
		}
		volumeTypeID = id
	}

	body := map[string]interface{}{
		"newVolumeTypeId": volumeTypeID,
	}
	if sizeSet && size > 0 {
		body["newSize"] = size
	}

	result, err := apiClient.Put(
		fmt.Sprintf("/v2/%s/volumes/%s/resize", projectID, volumeID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to resize volume %s: %w", volumeID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printVolumeResizePreview(v map[string]interface{}, newSize int, newVolumeTypeID string) {
	fmt.Println("Current volume info:")
	fmt.Println()
	fmt.Printf("  ID:           %v\n", v["id"])
	fmt.Printf("  Name:         %v\n", v["name"])
	fmt.Printf("  Size:         %v GiB\n", v["size"])
	fmt.Printf("  Volume Type:  %v\n", v["volumeTypeId"])
	fmt.Printf("  Status:       %v\n", v["status"])
	fmt.Println()
	fmt.Println("Resize plan:")
	fmt.Println()
	if newSize > 0 {
		fmt.Printf("  Size:         %v GiB  →  %d GiB\n", v["size"], newSize)
	} else {
		fmt.Printf("  Size:         %v GiB  (unchanged)\n", v["size"])
	}
	if newVolumeTypeID != "" && newVolumeTypeID != fmt.Sprintf("%v", v["volumeTypeId"]) {
		fmt.Printf("  Volume Type:  %v  →  %s\n", v["volumeTypeId"], newVolumeTypeID)
	} else {
		fmt.Printf("  Volume Type:  %v  (unchanged)\n", v["volumeTypeId"])
	}
	fmt.Println()
}

func validateResize(volumeID string, size int, volumeTypeID string) error {
	var errs []string

	if size < 0 {
		errs = append(errs, fmt.Sprintf("size %d GiB is invalid (must be a positive integer)", size))
	}

	fmt.Println("=== DRY RUN: Validation results ===")
	fmt.Println()
	fmt.Printf("  Volume ID:      %s\n", volumeID)
	if size > 0 {
		fmt.Printf("  New size:       %d GiB\n", size)
	}
	if volumeTypeID != "" {
		fmt.Printf("  New type:       %s\n", volumeTypeID)
	} else {
		fmt.Printf("  New type:       (reuse current volume type)\n")
	}
	fmt.Println()

	if len(errs) > 0 {
		fmt.Printf("Found %d error(s):\n", len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("dry-run validation failed with %d error(s)", len(errs))
	}

	fmt.Println("All parameters are valid. Run without --dry-run to resize the volume.")
	return nil
}

func extractVolumeData(response interface{}) (map[string]interface{}, error) {
	m, ok := response.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format from API")
	}
	data, ok := m["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not parse volume data from API response")
	}
	return data, nil
}
