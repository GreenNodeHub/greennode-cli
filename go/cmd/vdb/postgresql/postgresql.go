// Package postgresql is the command root for vDB PostgreSQL Cluster.
//
// The product is unusual: a cluster is served by 21 API operations spread over
// TWO prefixes — 14 of its own under /vdb-postgresql/v1, and 7 borrowed from
// /vdb-relational/v1 (list, get-by-id, histories, secrules read+write, reboot and
// delete), which the PostgreSQL group simply does not have. The command tree hides
// that split: every command a user needs is here, and each one calls whichever
// prefix actually serves it. See go/cmd/vdb/CLAUDE.md for the full map.
package postgresql

import (
	"github.com/greennodehub/greennode-cli/cmd/vdb/postgresql/backup"
	"github.com/greennodehub/greennode-cli/cmd/vdb/postgresql/catalog"
	"github.com/greennodehub/greennode-cli/cmd/vdb/postgresql/cluster"
	"github.com/spf13/cobra"
)

// PostgresqlCmd groups the PostgreSQL Cluster commands.
var PostgresqlCmd = &cobra.Command{
	Use:   "postgresql",
	Short: "Manage PostgreSQL Cluster resources",
	Long: "Manage vDB PostgreSQL Clusters: cluster lifecycle, backups and the " +
		"catalog of flavors, volume types and backup policies they are built from.\n\n" +
		"PostgreSQL Cluster is a distinct product from the single-instance PostgreSQL " +
		"offered by 'grn vdb relational' — a cluster has 2-10 nodes, its own flavors " +
		"and its own backup service. Cluster IDs start with 'pg-'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	PostgresqlCmd.AddCommand(cluster.ClusterCmd)
	PostgresqlCmd.AddCommand(catalog.CatalogCmd)
	PostgresqlCmd.AddCommand(backup.BackupCmd)
}
