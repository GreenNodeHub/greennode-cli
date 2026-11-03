package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listHistoriesCmd = &cobra.Command{
	Use:   "list-histories",
	Short: "List the action history of a PostgreSQL Cluster",
	Long: "List the recorded actions of a cluster — create, resize, restart, backup — " +
		"with their status and, for failures, the error message.\n\n" +
		"Served by the Relational Database histories endpoint. Results are paginated; " +
		"--page is 1-based and no filters are accepted. Table output omits the " +
		"description field, which embeds the whole order cart; use --output json for it.",
	Args: cobra.NoArgs,
	RunE: runListHistories,
}

func init() {
	f := listHistoriesCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	listHistoriesCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	listHistoriesCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

// historiesOptions maps the paging flags onto the shared list query. This
// endpoint accepts pageNumber/pageSize only.
func historiesOptions(cmd *cobra.Command) vdbclient.ListOptions {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	return vdbclient.ListOptions{Page: page, PageSize: pageSize}
}

func runListHistories(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, relPath(clusterID, "/histories"),
		vdbclient.BuildListQuery(historiesOptions(cmd)))
	if err != nil {
		return fmt.Errorf("failed to list history of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, historyColumns)
}
