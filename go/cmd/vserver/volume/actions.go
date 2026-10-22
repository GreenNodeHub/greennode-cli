package volume

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var changeDeviceTypeCmd = &cobra.Command{Use: "change-device-type", Short: "Run a volume type migration action", RunE: runChangeDeviceType}
var renameCmd = &cobra.Command{Use: "rename", Short: "Rename a volume", RunE: runRename}
var attachCmd = &cobra.Command{Use: "attach", Short: "Attach a volume to a server", RunE: func(cmd *cobra.Command, args []string) error { return runAttachment(cmd, "attach") }}
var detachCmd = &cobra.Command{Use: "detach", Short: "Detach a volume from a server", RunE: func(cmd *cobra.Command, args []string) error { return runAttachment(cmd, "detach") }}

func init() {
	changeDeviceTypeCmd.Flags().String("volume-id", "", "Volume ID (required)")
	changeDeviceTypeCmd.Flags().String("volume-type-id", "", "Target volume type ID")
	changeDeviceTypeCmd.Flags().String("action", "", "Migration action: SNAPSHOT, MIGRATE, or CONFIRM-MIGRATE (required)")
	changeDeviceTypeCmd.Flags().Bool("confirm-migrate", false, "Confirm migration rather than rollback")
	addVolumeMetadataFlags(changeDeviceTypeCmd)
	changeDeviceTypeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	markVolumeRequired(changeDeviceTypeCmd, "volume-id", "action")

	renameCmd.Flags().String("volume-id", "", "Volume ID (required)")
	renameCmd.Flags().String("name", "", "New volume name (required)")
	addVolumeMetadataFlags(renameCmd)
	markVolumeRequired(renameCmd, "volume-id", "name")

	for _, cmd := range []*cobra.Command{attachCmd, detachCmd} {
		cmd.Flags().String("volume-id", "", "Volume ID (required)")
		cmd.Flags().String("server-id", "", "Server ID (required)")
		cmd.Flags().Bool("persistent-volume", false, "Keep the volume when its server is deleted")
		addVolumeMetadataFlags(cmd)
		markVolumeRequired(cmd, "volume-id", "server-id")
	}
}

func markVolumeRequired(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func addVolumeMetadataFlags(cmd *cobra.Command) {
	cmd.Flags().String("zone-id", "", "Availability zone ID")
	cmd.Flags().StringArray("tag", nil, "Tag in key=value form (repeatable)")
	cmd.Flags().Bool("dry-run", false, "Preview the operation without executing")
}

func volumeMutationBody(cmd *cobra.Command, fields map[string]any) (map[string]any, bool, error) {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	tags, err := parseTags(rawTags)
	if err != nil {
		return nil, false, err
	}
	fields["tags"] = tags
	fields["zoneId"] = nilIfEmpty(zoneID)
	if dryRun {
		cli.PrintDryRun("update", "volume", fields)
		return fields, true, nil
	}
	return fields, false, nil
}

func runChangeDeviceType(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	volumeTypeID, _ := cmd.Flags().GetString("volume-type-id")
	action, _ := cmd.Flags().GetString("action")
	confirmMigrate, _ := cmd.Flags().GetBool("confirm-migrate")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	action = strings.ToUpper(action)
	if action != "SNAPSHOT" && action != "MIGRATE" && action != "CONFIRM-MIGRATE" {
		return fmt.Errorf("--action must be SNAPSHOT, MIGRATE, or CONFIRM-MIGRATE")
	}
	if volumeTypeID != "" {
		if err := validator.ValidateID(volumeTypeID, "volume-type-id"); err != nil {
			return err
		}
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{
		"action":         action,
		"confirmMigrate": confirmMigrate,
		"volumeTypeId":   nilIfEmpty(volumeTypeID),
	})
	if err != nil || dryRun {
		return err
	}
	if !cli.Confirm(force, fmt.Sprintf("Run migration action %s on volume %s?", action, volumeID)) {
		return cli.ConfirmationError()
	}
	return runVolumePut(cmd, volumeID, "change-device-type", body)
}

func runRename(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	name, _ := cmd.Flags().GetString("name")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{"newName": name})
	if err != nil || dryRun {
		return err
	}
	return runVolumePut(cmd, volumeID, "rename", body)
}

func runAttachment(cmd *cobra.Command, action string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	serverID, _ := cmd.Flags().GetString("server-id")
	persistent, _ := cmd.Flags().GetBool("persistent-volume")
	for _, check := range []struct{ value, name string }{{volumeID, "volume-id"}, {serverID, "server-id"}} {
		if err := validator.ValidateID(check.value, check.name); err != nil {
			return err
		}
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{"persistentVolume": persistent})
	if err != nil || dryRun {
		return err
	}
	return runVolumePut(cmd, volumeID, "servers/"+serverID+"/"+action, body)
}

func runVolumePut(cmd *cobra.Command, volumeID, suffix string, body any) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/volumes/%s/%s", projectID, volumeID, suffix), body)
	if err != nil {
		return fmt.Errorf("failed volume operation for %s: %w", volumeID, err)
	}
	return outputResult(cmd, cfg, result)
}
