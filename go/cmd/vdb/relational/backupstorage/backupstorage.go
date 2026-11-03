// Package backupstorage holds the Relational Database backup-storage commands.
//
// Backup storage is a purchased quota, not a per-instance setting: you buy a package,
// resize it, or give it up. Backups beyond the free allowance that comes with your
// instances' flavors consume it — see `grn vdb relational backup get-free-storage`.
//
// Its action requests use the resource type "dbaas-backup-storage", not "dbaas", and
// the resource IDs look like "db-bk-storage-…".
package backupstorage

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupStorageCmd is the parent command for backup-storage commands.
var BackupStorageCmd = &cobra.Command{
	Use:   "backup-storage",
	Short: "Manage Relational Database backup storage",
	Long: "Inspect, buy, resize and release the backup storage quota used by vDB " +
		"Relational Database backups.\n\n" +
		"Backups first consume the free allowance that comes with an instance's flavor; " +
		"beyond that they need purchased storage. 'backup-storage list' shows what you " +
		"have, 'list-packages' what you can buy.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath    = "/vdb-relational/v1/backup-storages"
	paymentPath = "/vdb-relational/v1/payment/backup-storages"

	// storageIDPrefix is what the API's own examples show for a backup storage.
	storageIDPrefix = "db-bk-storage-"
)

func init() {
	BackupStorageCmd.AddCommand(listCmd)
	BackupStorageCmd.AddCommand(listPackagesCmd)
	BackupStorageCmd.AddCommand(createCmd)
	BackupStorageCmd.AddCommand(resizeCmd)
	BackupStorageCmd.AddCommand(deleteCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

func requireStorageID(storageID string) error {
	return vdbclient.RequireIDWithPrefix(storageID, "storage-id", storageIDPrefix,
		"Backup storage IDs start with 'db-bk-storage-'; run 'backup-storage list' to find one")
}

// storageColumns is the table view of a BackupStorageDetail: what you own and how
// much of it is in use.
var storageColumns = []string{
	"id", "name", "usage", "quota", "status", "backupPackageName", "engineGroup",
}
