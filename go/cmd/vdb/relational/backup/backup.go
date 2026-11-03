// Package backup holds the Relational Database backup commands.
//
// Two things set this API apart from its MemoryStore and PostgreSQL Cluster
// counterparts, and both are load-bearing:
//
//   - Deleting a backup is `DELETE /backups/{id}/delete` with a JSON **ARRAY**
//     body — the ID travels in the path AND in the array. MemoryStore uses
//     `POST /backups/delete` instead.
//   - Restoring does not restore in place. It creates a NEW instance from the
//     backup, through the order/payment flow, with a request that describes the
//     whole target instance. See restore.go.
package backup

import (
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupCmd is the parent command for Relational Database backup commands.
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage Relational Database backups",
	Long: "List, inspect, create, delete and restore backups of vDB Relational " +
		"Database instances.\n\n" +
		"Backups are billed against your backup storage quota beyond the free " +
		"allowance — see 'backup get-free-storage' and " +
		"'grn vdb relational backup-storage'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath = "/vdb-relational/v1/backups"

	// instanceIDPrefix / backupIDPrefix: the listing endpoints are shared with
	// PostgreSQL Cluster, and a backup ID is not an instance ID, so both are
	// checked before they reach a path.
	instanceIDPrefix = "db-"
	backupIDPrefix   = "bk-"
)

func init() {
	BackupCmd.AddCommand(listCmd)
	BackupCmd.AddCommand(getCmd)
	BackupCmd.AddCommand(createCmd)
	BackupCmd.AddCommand(deleteCmd)
	BackupCmd.AddCommand(restoreCmd)
	BackupCmd.AddCommand(getFreeStorageCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// requireBackupID rejects anything that is not a backup ID. Passing an instance ID
// to `backup delete` would otherwise reach the API as a path segment.
func requireBackupID(backupID string) error {
	return vdbclient.RequireIDWithPrefix(backupID, "backup-id", backupIDPrefix,
		"Backup IDs start with 'bk-'; run 'backup list' to find one")
}

func validateInstanceID(instanceID string) error {
	return validator.ValidateID(instanceID, "instance-id")
}

// backupColumns is the table view of a BackupInfo. The record carries 35 fields,
// including the full instance spec the backup was taken from (ram, vcpu, packageId,
// netIds, username) — those are what `backup restore` reads, and JSON output keeps
// them.
var backupColumns = []string{
	"id", "name", "dbInstanceId", "instanceName", "backupType",
	"status", "size", "datastoreType", "datastoreVersion", "created",
}

// instanceBackupColumns is the same view minus the instance identity, for the
// per-instance listing: that endpoint returns dbInstanceId and instanceName empty
// on every row (verified live).
var instanceBackupColumns = []string{
	"id", "name", "backupType", "status", "size", "parentName",
	"datastoreType", "datastoreVersion", "created",
}
