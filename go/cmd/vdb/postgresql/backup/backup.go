// Package backup holds the PostgreSQL Cluster backup commands.
//
// These are the cluster product's own endpoints under /vdb-postgresql/v1/backup
// and they are NOT shaped like the relational ones. The most important
// difference: every path parameter here is a CLUSTER id, not a backup id —
// ".../backup-vdb/{clusterId}/detail" means "the backup record of this cluster",
// where relational's "/backups/detail/{backupId}" means "this backup". Restore
// points are the individual snapshots.
package backup

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupCmd is the parent command for PostgreSQL Cluster backup commands.
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage PostgreSQL Cluster backups",
	Long: "Inspect the backup record of a vDB PostgreSQL Cluster, list its restore " +
		"points and take an on-demand backup.\n\n" +
		"A cluster has one backup record (its policy, location and total size) holding " +
		"many restore points. Scheduled backups come from the policy chosen at creation; " +
		"'backup create' takes one immediately.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	backupBase = "/vdb-postgresql/v1/backup/backup-vdb"
)

// clusterPath builds a path whose parameter is a CLUSTER id.
func clusterPath(clusterID, suffix string) string {
	return backupBase + "/" + clusterID + suffix
}

func init() {
	BackupCmd.AddCommand(listCmd)
	BackupCmd.AddCommand(getCmd)
	BackupCmd.AddCommand(listRestorePointsCmd)
	BackupCmd.AddCommand(createCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// backupColumns is the table view of a BackupDatabase record. databaseId is the
// cluster the record belongs to.
//
// backupPolicyName and backupDestinationName are deliberately absent: the API
// leaves both null and puts the real values in the nested policy /
// backupDestination objects, which table output cannot reach into. Use
// --output json, or --query 'policy.name', for those.
var backupColumns = []string{
	"id", "name", "databaseId", "status", "backupEnabled",
	"totalBackupSize", "latestRecord", "createdAt",
}

var restorePointColumns = []string{
	"id", "backupName", "status", "time", "engineVersion",
	"compressedSize", "uncompressedSize", "createdAt",
}
