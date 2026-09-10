package server

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{Use: "migrate", Short: "Run a server migration action", RunE: runMigrate}
var startMigrationCmd = &cobra.Command{Use: "start-migration", Short: "Start migrating a server", RunE: runStartMigration}
var completeMigrationCmd = &cobra.Command{Use: "complete-migration", Short: "Complete or fail a server migration", RunE: runCompleteMigration}

func init() {
	mf := migrateCmd.Flags()
	mf.String("server-id", "", "Server ID (required)")
	mf.String("action", "", "Migration action: SNAPSHOT, MIGRATE, or CONFIRM-MIGRATE (required)")
	mf.Bool("confirm-migrate", false, "Confirm migration rather than rollback")
	mf.String("zone-id", "", "Availability zone ID")
	mf.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	mf.Bool("dry-run", false, "Preview the migration action without executing")
	mf.Bool("force", false, "Skip confirmation prompt")
	for _, name := range []string{"server-id", "action"} {
		if err := migrateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}

	for _, cmd := range []*cobra.Command{startMigrationCmd, completeMigrationCmd} {
		cmd.Flags().String("server-id", "", "Server ID (required)")
		cmd.Flags().Bool("dry-run", false, "Preview the migration action without executing")
		cmd.Flags().Bool("force", false, "Skip confirmation prompt")
		if err := cmd.MarkFlagRequired("server-id"); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "server-id", err))
		}
	}
	completeMigrationCmd.Flags().Bool("success", false, "Mark the migration successful; false marks it failed")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	action, _ := cmd.Flags().GetString("action")
	confirmMigrate, _ := cmd.Flags().GetBool("confirm-migrate")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	action = strings.ToUpper(action)
	if action != "SNAPSHOT" && action != "MIGRATE" && action != "CONFIRM-MIGRATE" {
		return fmt.Errorf("--action must be SNAPSHOT, MIGRATE, or CONFIRM-MIGRATE")
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("migrate")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Run migration action %s on server %s?", action, serverID)) {
		return cli.ConfirmationError()
	}
	body := map[string]any{"action": action, "confirmMigrate": confirmMigrate, "zoneId": nilIfEmpty(zoneID), "tags": tags}
	return runMigrationPut(cmd, serverID, "migrate", body)
}

func runStartMigration(cmd *cobra.Command, args []string) error {
	return runSimpleMigration(cmd, "start-migrating", "start migration")
}

func runCompleteMigration(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	success, _ := cmd.Flags().GetBool("success")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("complete migration")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Complete migration for server %s with success=%t?", serverID, success)) {
		return cli.ConfirmationError()
	}
	return runMigrationPut(cmd, serverID, "complete-migrating", map[string]any{"success": success})
}

func runSimpleMigration(cmd *cobra.Command, suffix, action string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice(action)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Run %s for server %s?", action, serverID)) {
		return cli.ConfirmationError()
	}
	return runMigrationPut(cmd, serverID, suffix, nil)
}

func runMigrationPut(cmd *cobra.Command, serverID, suffix string, body any) error {
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
		return fmt.Errorf("failed migration operation for server %s: %w", serverID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
