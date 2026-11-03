package relational

import (
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational/backup"
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational/backupstorage"
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational/catalog"
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational/configuration"
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational/instance"
	"github.com/spf13/cobra"
)

// RelationalCmd groups the Relational Database commands, which sit under the
// /vdb-relational/v1 API prefix (MySQL / PostgreSQL single instances).
var RelationalCmd = &cobra.Command{
	Use:   "relational",
	Short: "Manage Relational Database (MySQL, PostgreSQL) resources",
	Long:  "Manage vDB Relational Database instances, backups and configurations.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	RelationalCmd.AddCommand(instance.InstanceCmd)
	RelationalCmd.AddCommand(catalog.CatalogCmd)
	RelationalCmd.AddCommand(backup.BackupCmd)
	RelationalCmd.AddCommand(configuration.ConfigurationCmd)
	RelationalCmd.AddCommand(backupstorage.BackupStorageCmd)
}
