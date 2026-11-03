// Package backupstorage holds the MemoryStore backup-storage commands.
//
// Two of the four paths mean the OPPOSITE of their relational namesakes, which is the
// trap here:
//
//	what you own          relational: /backup-storages/information   memorystore: /backup-storages
//	what you can buy      relational: /backup-storages               memorystore: /backup-storages/packages
//	release               relational: /actions/deletions (plural)    memorystore: /actions/delete (singular)
//
// So a `GET /backup-storages` written for relational lists purchasable packages and
// the same call here lists the quota you already have.
package backupstorage

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupStorageCmd is the parent command for MemoryStore backup storage.
var BackupStorageCmd = &cobra.Command{
	Use:   "backup-storage",
	Short: "Manage MemoryStore backup storage",
	Long: "Inspect, buy, resize and release the backup storage quota used by vDB " +
		"MemoryStore backups.\n\n" +
		"Backups first consume the free allowance that comes with an instance's flavor; " +
		"beyond that they need purchased storage. 'backup-storage list' shows what you " +
		"have, 'list-packages' what you can buy.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath    = "/vdb-memory/v1/backup-storages"
	paymentPath = "/vdb-memory/v1/payment/backup-storages"

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

var storageColumns = []string{
	"id", "name", "usage", "quota", "status", "backupPackageName", "engineGroup",
}

var packageColumns = []string{
	"engineGroup", "packageId", "packageName", "packageQuota", "price", "sku", "description",
}
