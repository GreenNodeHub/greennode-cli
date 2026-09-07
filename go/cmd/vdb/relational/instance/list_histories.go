package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listHistoriesCmd = &cobra.Command{
	Use:   "list-histories",
	Short: "List the action history of a Relational Database instance",
	Long: "List the recorded actions (create, resize, reboot, backup, ...) of a vDB " +
		"Relational Database instance.\n\n" +
		"Results are paginated; --page is 1-based. This endpoint takes no filters. " +
		"Table output leaves out the description field, which runs to several hundred " +
		"characters — use --output json to read it. PostgreSQL Cluster IDs ('pg-') work " +
		"here too.",
	RunE: runListHistories,
}

func init() {
	f := listHistoriesCmd.Flags()
	f.String("instance-id", "", "Database instance ID (required)")
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	listHistoriesCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	listHistoriesCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
}

// historiesOptions maps the paging flags onto the shared list query. This
// endpoint accepts pageNumber/pageSize only — no name/status filter — so the
// filter fields of ListOptions are deliberately left empty.
func historiesOptions(cmd *cobra.Command) vdbclient.ListOptions {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	return vdbclient.ListOptions{Page: page, PageSize: pageSize}
}

func runListHistories(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")

	if err := validator.ValidateID(instanceID, "instance-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := vdbclient.BuildListQuery(historiesOptions(cmd))

	result, err := vdbclient.Get(apiClient, historiesPath(instanceID), query)
	if err != nil {
		return fmt.Errorf("failed to list history of database instance %s: %w", instanceID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, historyColumns)
}
