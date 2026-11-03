package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Relational Database instances",
	Long: "List vDB Relational Database instances.\n\n" +
		"Results are paginated; --page is 1-based. Use --name and --status to filter " +
		"server-side.",
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	f.String("name", "", "Filter by instance name")
	f.String("status", "", "Filter by status (comma-separated for multiple values)")

	// Bound here, next to the flag: see the init-order note in completion.go.
	listCmd.RegisterFlagCompletionFunc("status", statusCompletion()) //nolint:errcheck
}

// listOptions maps the command's flags onto the shared list query. Split out
// from runList so the flag wiring is testable without a configured client.
func listOptions(cmd *cobra.Command) vdbclient.ListOptions {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	name, _ := cmd.Flags().GetString("name")
	status, _ := cmd.Flags().GetString("status")

	return vdbclient.ListOptions{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		Statuses: cli.ParseCommaSeparated(status),
	}
}

func runList(cmd *cobra.Command, args []string) error {
	query := vdbclient.BuildListQuery(listOptions(cmd))

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return fmt.Errorf("failed to list database instances: %w", err)
	}

	return outputList(cmd, result)
}
