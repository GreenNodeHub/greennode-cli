// Package memorystore is the command root for vDB MemoryStore (Redis).
//
// MemoryStore has the same 5 nouns as Relational Database and almost none of the
// same paths. The differences are not cosmetic — see cmd/vdb/CLAUDE.md for the full
// table — but the ones that bite hardest:
//
//   - get instance is `/database-instances/{id}`, with no `/id/` segment.
//   - `update-setting` and `update-config-group` are hyphenated, where relational
//     nests them under `/update/`.
//   - the catalog lives under `/database/*`, not `/database-instances/*`.
//   - deleting a backup or a config group is a POST with no ID in the path.
//   - `GET /backup-storages` lists what you OWN here, while in relational the same
//     path lists what you can BUY.
//
// Redis-specific behaviour, which has no relational equivalent:
//
//   - There is no volume: an instance takes no volume type or size, and there is no
//     resize-storage command. Only the flavor is resized.
//   - Authentication is a master password on the instance (`redisPasswordEnabled` +
//     `redisPassword`), not a user/database pair. Public access requires it.
//   - Changing that password needs `editRedisPassword: true` alongside it.
package memorystore

import (
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore/backup"
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore/backupstorage"
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore/catalog"
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore/configuration"
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore/instance"
	"github.com/spf13/cobra"
)

// MemorystoreCmd groups the MemoryStore commands.
var MemorystoreCmd = &cobra.Command{
	Use:   "memorystore",
	Short: "Manage MemoryStore (Redis) resources",
	Long: "Manage vDB MemoryStore instances, their backups, config groups and backup " +
		"storage.\n\n" +
		"MemoryStore is Redis. An instance has no volume — its capacity comes from the " +
		"flavor — so there is no storage resize, and access is controlled by a master " +
		"password rather than a database user.\n\n" +
		"Instance IDs start with 'db-', the same prefix Relational Database uses: the two " +
		"products keep separate listings, so use 'memorystore instance list' to find IDs " +
		"for these commands.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	MemorystoreCmd.AddCommand(instance.InstanceCmd)
	MemorystoreCmd.AddCommand(catalog.CatalogCmd)
	MemorystoreCmd.AddCommand(backup.BackupCmd)
	MemorystoreCmd.AddCommand(configuration.ConfigurationCmd)
	MemorystoreCmd.AddCommand(backupstorage.BackupStorageCmd)
}
