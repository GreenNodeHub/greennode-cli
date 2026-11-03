package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the backup record of a PostgreSQL Cluster",
	Long: "Show one cluster's backup record: whether backup is enabled, which policy " +
		"and location it uses, the latest record and the total size.\n\n" +
		"The flag is --cluster-id, not a backup ID: this API keys backups by the cluster " +
		"they belong to.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	getCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in
	// cmd/vdb/relational/instance/completion.go.
	getCmd.RegisterFlagCompletionFunc("cluster-id", cli.ResourceCompletion(clusterResourceKey)) //nolint:errcheck
}

func runGet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(clusterPath(clusterID, "/detail"), nil)
	if err != nil {
		return fmt.Errorf("failed to get the backup record of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	// A backup record nests backupDestination and policy objects, so print it as
	// key/value rather than picking a nested array at random.
	return vdbclient.Output(cmd, result)
}
