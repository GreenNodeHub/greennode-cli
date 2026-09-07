package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var getVolumeUsedCmd = &cobra.Command{
	Use:   "get-volume-used",
	Short: "Show storage used by a PostgreSQL Cluster",
	Long: "Show how much of a cluster's provisioned storage is in use.\n\n" +
		"One of the cluster's own endpoints (/vdb-postgresql/v1). The API answers with " +
		"an array — one entry per node — so this prints it as-is rather than as a table.",
	Args: cobra.NoArgs,
	RunE: runGetVolumeUsed,
}

func init() {
	f := getVolumeUsedCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	getVolumeUsedCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	getVolumeUsedCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runGetVolumeUsed(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(pgPath(clusterID, "/volume-used"), nil)
	if err != nil {
		return fmt.Errorf("failed to get storage usage of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}
