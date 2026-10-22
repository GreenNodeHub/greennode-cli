package volume

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{Use: "snapshot", Short: "Manage volume snapshot points", Args: cobra.NoArgs}
var snapshotPolicyCmd = &cobra.Command{Use: "snapshot-policy", Short: "Manage the volume snapshot policy", Args: cobra.NoArgs}

func init() {
	list := &cobra.Command{Use: "list", Short: "List volume snapshot points", RunE: runSnapshotList}
	list.Flags().String("volume-id", "", "Volume ID (required)")
	list.Flags().Int("page", 1, "Page number (1-based)")
	list.Flags().Int("page-size", 10, "Number of items per page")
	markVolumeRequired(list, "volume-id")

	create := &cobra.Command{Use: "create", Short: "Create a volume snapshot point now", RunE: runSnapshotCreate}
	create.Flags().String("volume-id", "", "Volume ID (required)")
	create.Flags().String("name", "", "Snapshot name (required)")
	create.Flags().String("description", "", "Snapshot description (required)")
	create.Flags().Bool("permanent", false, "Retain the snapshot permanently")
	create.Flags().Int("retained-days", 0, "Number of days to retain a non-permanent snapshot")
	addVolumeMetadataFlags(create)
	markVolumeRequired(create, "volume-id", "name", "description")

	rollback := &cobra.Command{Use: "rollback", Short: "Roll back a volume to a snapshot point", RunE: runSnapshotRollback}
	rollback.Flags().String("volume-id", "", "Volume ID (required)")
	rollback.Flags().String("snapshot-point-id", "", "Snapshot volume point ID (required)")
	rollback.Flags().Bool("restart", false, "Restart the attached server after rollback completes")
	addVolumeMetadataFlags(rollback)
	rollback.Flags().Bool("force", false, "Skip confirmation prompt")
	markVolumeRequired(rollback, "volume-id", "snapshot-point-id")

	deletePoint := &cobra.Command{Use: "delete", Short: "Delete a volume snapshot point", RunE: runSnapshotDelete}
	deletePoint.Flags().String("volume-id", "", "Volume ID (required)")
	deletePoint.Flags().String("snapshot-point-id", "", "Snapshot volume point ID (required)")
	deletePoint.Flags().Bool("dry-run", false, "Preview deletion without executing")
	deletePoint.Flags().Bool("force", false, "Skip confirmation prompt")
	markVolumeRequired(deletePoint, "volume-id", "snapshot-point-id")

	snapshotCmd.AddCommand(list, create, rollback, deletePoint)

	deletePolicy := &cobra.Command{Use: "delete", Short: "Delete the volume snapshot policy", RunE: runSnapshotPolicyDelete}
	deletePolicy.Flags().String("volume-id", "", "Volume ID (required)")
	deletePolicy.Flags().Bool("dry-run", false, "Preview deletion without executing")
	deletePolicy.Flags().Bool("force", false, "Skip confirmation prompt")
	markVolumeRequired(deletePolicy, "volume-id")

	detail := &cobra.Command{Use: "detail", Short: "Get volume snapshot policy details", RunE: runSnapshotPolicyDetail}
	detail.Flags().String("volume-id", "", "Volume ID (required)")
	markVolumeRequired(detail, "volume-id")

	update := &cobra.Command{Use: "update", Short: "Update the volume snapshot policy", RunE: runSnapshotPolicyUpdate}
	update.Flags().String("volume-id", "", "Volume ID (required)")
	update.Flags().String("snapshot-policy-id", "", "Snapshot policy ID (required)")
	addVolumeMetadataFlags(update)
	markVolumeRequired(update, "volume-id", "snapshot-policy-id")

	enable := newVolumeSnapshotToggleCommand("enable", "enable-auto", "Enable automatic volume snapshots")
	disable := newVolumeSnapshotToggleCommand("disable", "disable-auto", "Disable automatic volume snapshots")
	snapshotPolicyCmd.AddCommand(deletePolicy, detail, update, enable, disable)
}

func newVolumeSnapshotToggleCommand(use, suffix, short string) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: short}
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return runSnapshotPolicyToggle(cmd, suffix) }
	cmd.Flags().String("volume-id", "", "Volume ID (required)")
	cmd.Flags().String("server-id", "", "Server ID (required)")
	cmd.Flags().Bool("dry-run", false, "Preview the operation without executing")
	markVolumeRequired(cmd, "volume-id", "server-id")
	return cmd
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	params, err := vserverclient.ValidatedPaginationParams(cmd, "page", "size")
	if err != nil {
		return err
	}
	return runVolumeSnapshotGet(cmd, volumeID, "snapshots", params)
}

func runSnapshotCreate(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	permanent, _ := cmd.Flags().GetBool("permanent")
	retainedDays, _ := cmd.Flags().GetInt("retained-days")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	if name == "" || description == "" {
		return fmt.Errorf("--name and --description are required")
	}
	if retainedDays < 0 {
		return fmt.Errorf("--retained-days cannot be negative")
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{
		"name": name, "description": description, "isPermanently": permanent, "retainedDays": retainedDays,
	})
	if err != nil || dryRun {
		return err
	}
	return runVolumeSnapshotPost(cmd, volumeID, "snapshots", body)
}

func runSnapshotRollback(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	pointID, _ := cmd.Flags().GetString("snapshot-point-id")
	restart, _ := cmd.Flags().GetBool("restart")
	force, _ := cmd.Flags().GetBool("force")
	if err := validateVolumeSnapshotIDs(volumeID, pointID, "snapshot-point-id"); err != nil {
		return err
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{
		"volumeId": volumeID, "snapshotVolumePointId": pointID, "restartServerWhenRevertCompleted": restart,
	})
	if err != nil || dryRun {
		return err
	}
	if !cli.Confirm(force, fmt.Sprintf("Roll back volume %s to snapshot point %s?", volumeID, pointID)) {
		return cli.ConfirmationError()
	}
	return runVolumeSnapshotPost(cmd, volumeID, "snapshots/rollback/"+pointID, body)
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	pointID, _ := cmd.Flags().GetString("snapshot-point-id")
	if err := validateVolumeSnapshotIDs(volumeID, pointID, "snapshot-point-id"); err != nil {
		return err
	}
	return runVolumeSnapshotDelete(cmd, volumeID, "snapshots/"+pointID, "snapshot point "+pointID)
}

func runSnapshotPolicyDelete(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	return runVolumeSnapshotDelete(cmd, volumeID, "volume-snapshots", "snapshot policy")
}

func runSnapshotPolicyDetail(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	return runVolumeSnapshotGet(cmd, volumeID, "volume-snapshots/detail", nil)
}

func runSnapshotPolicyUpdate(cmd *cobra.Command, args []string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	policyID, _ := cmd.Flags().GetString("snapshot-policy-id")
	if err := validateVolumeSnapshotIDs(volumeID, policyID, "snapshot-policy-id"); err != nil {
		return err
	}
	body, dryRun, err := volumeMutationBody(cmd, map[string]any{"snapshotPolicyId": policyID})
	if err != nil || dryRun {
		return err
	}
	return runVolumePut(cmd, volumeID, "volume-snapshots/policy", body)
}

func runSnapshotPolicyToggle(cmd *cobra.Command, suffix string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	serverID, _ := cmd.Flags().GetString("server-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validateVolumeSnapshotIDs(volumeID, serverID, "server-id"); err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice(suffix)
		return nil
	}
	return runVolumePut(cmd, volumeID, "volume-snapshots/servers/"+serverID+"/"+suffix, nil)
}

func validateVolumeSnapshotIDs(volumeID, otherID, otherName string) error {
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
		return err
	}
	return validator.ValidateID(otherID, otherName)
}

func runVolumeSnapshotGet(cmd *cobra.Command, volumeID, suffix string, params map[string]string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/volumes/%s/%s", projectID, volumeID, suffix), params)
	if err != nil {
		return fmt.Errorf("failed to get volume snapshots: %w", err)
	}
	return outputResult(cmd, cfg, result)
}

func runVolumeSnapshotPost(cmd *cobra.Command, volumeID, suffix string, body map[string]any) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/volumes/%s/%s", projectID, volumeID, suffix), body)
	if err != nil {
		return fmt.Errorf("failed volume snapshot operation: %w", err)
	}
	return outputResult(cmd, cfg, result)
}

func runVolumeSnapshotDelete(cmd *cobra.Command, volumeID, suffix, target string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if dryRun {
		cli.DryRunNotice("delete " + target)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Delete %s from volume %s?", target, volumeID)) {
		return cli.ConfirmationError()
	}
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/volumes/%s/%s", projectID, volumeID, suffix), nil)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", target, err)
	}
	return outputResult(cmd, cfg, result)
}
