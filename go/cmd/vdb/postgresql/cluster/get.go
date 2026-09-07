package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a PostgreSQL Cluster",
	Long: "Show the full details of one vDB PostgreSQL Cluster.\n\n" +
		"Served by the Relational Database get-by-id endpoint, which accepts cluster " +
		"IDs; a cluster answers with deployType 'cluster', numberOfNodes and the " +
		"privateRwIp / publicRwIp / privateRoIp / publicRoIp fields.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	getCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	getCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
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

	result, err := apiClient.Get(detailPath(clusterID), nil)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL Cluster %s: %w", clusterID, err)
	}

	// Output, not OutputWithColumns: a detail payload has several nested arrays
	// and table extraction would pick one at random.
	return vdbclient.Output(cmd, result)
}
