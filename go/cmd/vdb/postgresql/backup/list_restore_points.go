package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listRestorePointsCmd = &cobra.Command{
	Use:   "list-restore-points",
	Short: "List the restore points of a PostgreSQL Cluster",
	Long: "List the individual snapshots taken for a cluster, with their size and the " +
		"point in time they restore to.\n\n" +
		"A restore point ID ('bk-db-pt-...') is what --backup-point-id expects on " +
		"'cluster create' when creating a cluster from a backup.",
	Args: cobra.NoArgs,
	RunE: runListRestorePoints,
}

func init() {
	f := listRestorePointsCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	listRestorePointsCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in
	// cmd/vdb/relational/instance/completion.go.
	listRestorePointsCmd.RegisterFlagCompletionFunc("cluster-id", cli.ResourceCompletion(clusterResourceKey)) //nolint:errcheck
}

func runListRestorePoints(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(clusterPath(clusterID, "/restore-point"), nil)
	if err != nil {
		return fmt.Errorf("failed to list restore points of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, restorePointColumns)
}
