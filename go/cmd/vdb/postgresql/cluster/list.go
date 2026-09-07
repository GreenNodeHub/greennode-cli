package cluster

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List PostgreSQL Clusters",
	Long: "List vDB PostgreSQL Clusters.\n\n" +
		"This calls the Relational Database listing, the only endpoint that enumerates " +
		"clusters, and keeps just the 'pg-' rows. Because that endpoint's server-side " +
		"filters do not apply to cluster rows, --name and --status are applied here, on " +
		"the client: --page and --page-size still page the underlying mixed listing, so " +
		"a page can come back with fewer clusters than --page-size, and pageObject in " +
		"JSON output describes that mixed listing rather than the filtered result.",
	Args: cobra.NoArgs,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	f.String("name", "", "Filter by cluster name (substring match, applied client-side)")
	f.String("status", "", "Filter by status (comma-separated for multiple values, applied client-side)")

	// Bound here, next to the flag: see the init-order note in completion.go.
	listCmd.RegisterFlagCompletionFunc("status", statusCompletion()) //nolint:errcheck
}

func listOptions(cmd *cobra.Command) vdbclient.ListOptions {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	// Name and Statuses are deliberately NOT put on the query: the API ignores
	// them for pg- rows, so sending them would filter the db- rows we discard
	// anyway while silently doing nothing to the ones we keep.
	return vdbclient.ListOptions{Page: page, PageSize: pageSize}
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, relBase, vdbclient.BuildListQuery(listOptions(cmd)))
	if err != nil {
		return fmt.Errorf("failed to list PostgreSQL Clusters: %w", err)
	}

	name, _ := cmd.Flags().GetString("name")
	statusFlag, _ := cmd.Flags().GetString("status")
	statuses := cli.ParseCommaSeparated(statusFlag)

	return vdbclient.OutputWithColumns(cmd, keepClusters(vdbclient.Unwrap(result), name, statuses), listColumns)
}

// keepClusters replaces the listing's item array with only the cluster rows that
// match the client-side filters, leaving every other key (projectId, pageObject)
// untouched. It takes an already-unwrapped payload.
func keepClusters(payload interface{}, name string, statuses []string) interface{} {
	obj, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}
	items, ok := obj["data"].([]interface{})
	if !ok {
		return payload
	}

	kept := make([]interface{}, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); !strings.HasPrefix(id, clusterIDPrefix) {
			continue
		}
		if !matchesName(row, name) || !matchesStatus(row, statuses) {
			continue
		}
		kept = append(kept, row)
	}

	out := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		out[k] = v
	}
	out["data"] = kept
	return out
}

func matchesName(row map[string]interface{}, name string) bool {
	if name == "" {
		return true
	}
	rowName, _ := row["name"].(string)
	// Substring, case-insensitive — the server-side filter this replaces is a
	// substring match, and a cluster's generated name mixes case.
	return strings.Contains(strings.ToLower(rowName), strings.ToLower(name))
}

func matchesStatus(row map[string]interface{}, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	rowStatus, _ := row["status"].(string)
	for _, want := range statuses {
		if strings.EqualFold(rowStatus, want) {
			return true
		}
	}
	return false
}
