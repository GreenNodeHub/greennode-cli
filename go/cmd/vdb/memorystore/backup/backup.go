// Package backup holds the MemoryStore backup commands.
//
// Every path differs from the relational backup API, and two of them differ in a way
// that changes the request, not just the URL:
//
//	operation            relational                        memorystore
//	get one              GET  /backups/detail/{id}         GET  /backups/{id}/detail
//	delete               DELETE /backups/{id}/delete       POST /backups/delete  (no ID in the path)
//	list by instance     GET  /backups/insId/{id}          GET  /database-instances/{id}/backups
//
// Both deletes take a JSON ARRAY body, so the memorystore one carries the ID only
// there — which also means it could delete several at once, though this command sends
// one to keep the confirmation honest.
package backup

import (
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupCmd is the parent command for MemoryStore backup commands.
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage MemoryStore backups",
	Long: "List, inspect, create, delete and restore backups of vDB MemoryStore " +
		"instances.\n\n" +
		"Backups are billed against your backup storage quota beyond the free allowance — " +
		"see 'backup get-free-storage' and 'grn vdb memorystore backup-storage'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath     = "/vdb-memory/v1/backups"
	instanceBase = "/vdb-memory/v1/database-instances"

	backupIDPrefix   = "bk-"
	instanceIDPrefix = "db-"

	// Backup retention bounds, as in every other vdb group.
	minBackupDuration = 2
	maxBackupDuration = 14
)

// Completion keys. The instance one is MemoryStore's own: relational IDs share the
// "db-" prefix but belong to a different listing.
const (
	instanceResourceKey = "vdb:memorystore-instance"
	zoneResourceKey     = "vdb:relational-zone"
	subnetResourceKey   = "vdb:relational-subnet"
	configGroupResource = "vdb:memorystore-config-group"
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

func requireBackupID(backupID string) error {
	return vdbclient.RequireIDWithPrefix(backupID, "backup-id", backupIDPrefix,
		"Backup IDs start with 'bk-'; run 'backup list' to find one")
}

func requireInstanceID(instanceID string) error {
	return vdbclient.RequireIDWithPrefix(instanceID, "instance-id", instanceIDPrefix,
		"MemoryStore instance IDs start with 'db-'; run 'memorystore instance list' to find one")
}

func validateInstanceID(instanceID string) error {
	return validator.ValidateID(instanceID, "instance-id")
}

var backupColumns = []string{
	"id", "name", "dbInstanceId", "instanceName", "backupType",
	"status", "size", "datastoreType", "datastoreVersion", "created",
}

// instanceBackupColumns drops the instance identity: the per-instance endpoint
// leaves those fields empty, as its relational counterpart does.
var instanceBackupColumns = []string{
	"id", "name", "backupType", "status", "size", "parentName",
	"datastoreType", "datastoreVersion", "created",
}
