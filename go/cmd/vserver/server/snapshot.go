package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage server snapshot points",
	Args:  cobra.NoArgs,
}

var snapshotPolicyCmd = &cobra.Command{
	Use:   "snapshot-policy",
	Short: "Manage the server snapshot policy",
	Args:  cobra.NoArgs,
}

func init() {
	list := &cobra.Command{Use: "list", Short: "List snapshot points", RunE: runSnapshotList}
	list.Flags().String("server-id", "", "Server ID (required)")
	list.Flags().Int("page", 1, "Page number (1-based)")
	list.Flags().Int("page-size", 10, "Number of items per page")
	markRequired(list, "server-id")

	create := &cobra.Command{Use: "create", Short: "Create a snapshot point now", RunE: runSnapshotCreate}
	create.Flags().String("server-id", "", "Server ID (required)")
	create.Flags().String("name", "", "Snapshot name (required)")
	create.Flags().String("description", "", "Snapshot description (required)")
	create.Flags().Bool("permanent", false, "Retain the snapshot permanently")
	create.Flags().Int("retained-days", 0, "Number of days to retain a non-permanent snapshot")
	addSnapshotMetadataFlags(create)
	markRequired(create, "server-id", "name", "description")

	detail := &cobra.Command{Use: "detail", Short: "Get server snapshot details", RunE: runSnapshotDetail}
	detail.Flags().String("server-id", "", "Server ID (required)")
	markRequired(detail, "server-id")

	rollback := &cobra.Command{Use: "rollback", Short: "Roll back a server to a snapshot point", RunE: runSnapshotRollback}
	rollback.Flags().String("server-id", "", "Server ID (required)")
	rollback.Flags().String("snapshot-point-id", "", "Snapshot server point ID (required)")
	rollback.Flags().Bool("restart", false, "Restart the server after rollback completes")
	addSnapshotMetadataFlags(rollback)
	rollback.Flags().Bool("force", false, "Skip confirmation prompt")
	markRequired(rollback, "server-id", "snapshot-point-id")

	deletePoint := &cobra.Command{Use: "delete", Short: "Delete a server snapshot point", RunE: runSnapshotDelete}
	deletePoint.Flags().String("server-id", "", "Server ID (required)")
	deletePoint.Flags().String("snapshot-point-id", "", "Snapshot server point ID (required)")
	deletePoint.Flags().Bool("dry-run", false, "Preview deletion without executing")
	deletePoint.Flags().Bool("force", false, "Skip confirmation prompt")
	markRequired(deletePoint, "server-id", "snapshot-point-id")

	snapshotCmd.AddCommand(list, create, detail, rollback, deletePoint)

	createPolicy := &cobra.Command{Use: "create", Short: "Create or assign a server snapshot policy", RunE: runSnapshotPolicyCreate}
	createPolicy.Flags().String("server-id", "", "Server ID (required)")
	createPolicy.Flags().String("description", "", "Snapshot policy description (required)")
	createPolicy.Flags().String("name", "", "Snapshot policy name")
	createPolicy.Flags().Bool("enable", false, "Enable automatic snapshots")
	createPolicy.Flags().String("snapshot-policy-id", "", "Snapshot policy ID")
	createPolicy.Flags().String("volume-ids", "", "Volume IDs included in the snapshot, comma-separated")
	addSnapshotMetadataFlags(createPolicy)
	markRequired(createPolicy, "server-id", "description")

	deletePolicy := &cobra.Command{Use: "delete", Short: "Delete the server snapshot policy", RunE: runSnapshotPolicyDelete}
	deletePolicy.Flags().String("server-id", "", "Server ID (required)")
	deletePolicy.Flags().Bool("dry-run", false, "Preview deletion without executing")
	deletePolicy.Flags().Bool("force", false, "Skip confirmation prompt")
	markRequired(deletePolicy, "server-id")

	enablePolicy := newSnapshotPolicyToggleCommand("enable", "enable-auto", "Enable automatic server snapshots")
	disablePolicy := newSnapshotPolicyToggleCommand("disable", "disable-auto", "Disable automatic server snapshots")

	updatePolicy := &cobra.Command{Use: "update", Short: "Update the server snapshot policy", RunE: runSnapshotPolicyUpdate}
	updatePolicy.Flags().String("server-id", "", "Server ID (required)")
	updatePolicy.Flags().String("snapshot-policy-id", "", "Snapshot policy ID (required)")
	addSnapshotMetadataFlags(updatePolicy)
	markRequired(updatePolicy, "server-id", "snapshot-policy-id")

	listShared := &cobra.Command{Use: "list-shared", Short: "List shared server snapshots", RunE: runSnapshotPolicyListShared}
	listShared.Flags().String("server-id", "", "Server ID (required)")
	markRequired(listShared, "server-id")

	revokeShared := &cobra.Command{Use: "revoke-shared", Short: "Revoke a shared server snapshot", RunE: runSnapshotPolicyRevokeShared}
	revokeShared.Flags().String("server-id", "", "Server ID (required)")
	revokeShared.Flags().String("shared-snapshot-id", "", "Shared snapshot ID (required)")
	revokeShared.Flags().Bool("dry-run", false, "Preview revocation without executing")
	revokeShared.Flags().Bool("force", false, "Skip confirmation prompt")
	markRequired(revokeShared, "server-id", "shared-snapshot-id")

	snapshotPolicyCmd.AddCommand(createPolicy, deletePolicy, enablePolicy, disablePolicy, updatePolicy, listShared, revokeShared)
}

func addSnapshotMetadataFlags(cmd *cobra.Command) {
	cmd.Flags().String("zone-id", "", "Availability zone ID")
	cmd.Flags().StringArray("tag", nil, "Tag in key=value form (repeatable)")
	cmd.Flags().Bool("dry-run", false, "Preview the operation without executing")
}

func newSnapshotPolicyToggleCommand(use, suffix, short string) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: short}
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return runSnapshotPolicyToggle(cmd, suffix) }
	cmd.Flags().String("server-id", "", "Server ID (required)")
	cmd.Flags().Bool("dry-run", false, "Preview the operation without executing")
	markRequired(cmd, "server-id")
	return cmd
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	params, err := vserverclient.ValidatedPaginationParams(cmd, "page", "size")
	if err != nil {
		return err
	}
	return runSnapshotGet(cmd, serverID, "snapshots", params)
}

func runSnapshotDetail(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	return runSnapshotGet(cmd, serverID, "snapshots/detail", nil)
}

func runSnapshotCreate(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	permanent, _ := cmd.Flags().GetBool("permanent")
	retainedDays, _ := cmd.Flags().GetInt("retained-days")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	if name == "" || description == "" {
		return fmt.Errorf("--name and --description are required")
	}
	if retainedDays < 0 {
		return fmt.Errorf("--retained-days cannot be negative")
	}
	body, dryRun, err := snapshotBody(cmd, map[string]any{
		"name":          name,
		"description":   description,
		"isPermanently": permanent,
		"retainedDays":  retainedDays,
	})
	if err != nil || dryRun {
		return err
	}
	return runSnapshotPost(cmd, serverID, "snapshots", body)
}

func runSnapshotRollback(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	pointID, _ := cmd.Flags().GetString("snapshot-point-id")
	restart, _ := cmd.Flags().GetBool("restart")
	force, _ := cmd.Flags().GetBool("force")
	if err := validateIDs(serverID, pointID, "snapshot-point-id"); err != nil {
		return err
	}
	body, dryRun, err := snapshotBody(cmd, map[string]any{
		"serverId":                         serverID,
		"snapshotServerPointId":            pointID,
		"restartServerWhenRevertCompleted": restart,
	})
	if err != nil || dryRun {
		return err
	}
	if !cli.Confirm(force, fmt.Sprintf("Roll back server %s to snapshot point %s?", serverID, pointID)) {
		return cli.ConfirmationError()
	}
	return runSnapshotPost(cmd, serverID, "snapshots/rollback/"+pointID, body)
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	pointID, _ := cmd.Flags().GetString("snapshot-point-id")
	if err := validateIDs(serverID, pointID, "snapshot-point-id"); err != nil {
		return err
	}
	return runSnapshotDeleteRequest(cmd, serverID, "snapshots/"+pointID, "snapshot point "+pointID)
}

func runSnapshotPolicyCreate(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	description, _ := cmd.Flags().GetString("description")
	name, _ := cmd.Flags().GetString("name")
	enable, _ := cmd.Flags().GetBool("enable")
	policyID, _ := cmd.Flags().GetString("snapshot-policy-id")
	volumeIDs, _ := cmd.Flags().GetString("volume-ids")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	if description == "" {
		return fmt.Errorf("--description is required")
	}
	volumes := parseCommaSeparated(volumeIDs)
	for _, id := range volumes {
		if err := validator.ValidateID(id, "volume-id"); err != nil {
			return err
		}
	}
	body, dryRun, err := snapshotBody(cmd, map[string]any{
		"description":      description,
		"name":             nilIfEmpty(name),
		"enableSnapshot":   enable,
		"snapshotPolicyId": nilIfEmpty(policyID),
		"volumeIds":        volumes,
	})
	if err != nil || dryRun {
		return err
	}
	return runSnapshotPost(cmd, serverID, "server-snapshots", body)
}

func runSnapshotPolicyDelete(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	return runSnapshotDeleteRequest(cmd, serverID, "server-snapshots", "snapshot policy")
}

func runSnapshotPolicyToggle(cmd *cobra.Command, suffix string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice(suffix)
		return nil
	}
	return runSnapshotPut(cmd, serverID, "server-snapshots/"+suffix, nil)
}

func runSnapshotPolicyUpdate(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	policyID, _ := cmd.Flags().GetString("snapshot-policy-id")
	if err := validateIDs(serverID, policyID, "snapshot-policy-id"); err != nil {
		return err
	}
	body, dryRun, err := snapshotBody(cmd, map[string]any{"snapshotPolicyId": policyID})
	if err != nil || dryRun {
		return err
	}
	return runSnapshotPut(cmd, serverID, "server-snapshots/policy", body)
}

func runSnapshotPolicyListShared(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validateServerID(serverID); err != nil {
		return err
	}
	return runSnapshotGet(cmd, serverID, "server-snapshots/shared", nil)
}

func runSnapshotPolicyRevokeShared(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	sharedID, _ := cmd.Flags().GetString("shared-snapshot-id")
	if err := validateIDs(serverID, sharedID, "shared-snapshot-id"); err != nil {
		return err
	}
	return runSnapshotDeleteRequest(cmd, serverID, "server-snapshots/shared/"+sharedID, "shared snapshot "+sharedID)
}

func snapshotBody(cmd *cobra.Command, fields map[string]any) (map[string]any, bool, error) {
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
		cli.PrintDryRun("snapshot operation", "server", fields)
		return fields, true, nil
	}
	return fields, false, nil
}

func validateServerID(serverID string) error {
	return validator.ValidateID(serverID, "server-id")
}

func validateIDs(serverID, otherID, otherName string) error {
	if err := validateServerID(serverID); err != nil {
		return err
	}
	return validator.ValidateID(otherID, otherName)
}

func runSnapshotGet(cmd *cobra.Command, serverID, suffix string, params map[string]string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/servers/%s/%s", projectID, serverID, suffix), params)
	if err != nil {
		return fmt.Errorf("failed to get server snapshots: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func runSnapshotPost(cmd *cobra.Command, serverID, suffix string, body map[string]any) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/servers/%s/%s", projectID, serverID, suffix), body)
	if err != nil {
		return fmt.Errorf("failed server snapshot operation: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func runSnapshotPut(cmd *cobra.Command, serverID, suffix string, body any) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/servers/%s/%s", projectID, serverID, suffix), body)
	if err != nil {
		return fmt.Errorf("failed server snapshot policy operation: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func runSnapshotDeleteRequest(cmd *cobra.Command, serverID, suffix, target string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if dryRun {
		cli.DryRunNotice("delete " + target)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Delete %s from server %s?", target, serverID)) {
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
	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/servers/%s/%s", projectID, serverID, suffix), nil)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", target, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
