package catalog

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// clusterDeployType is the value a config group must carry to be usable by a
// cluster; the spec is explicit that --config-id "can only be compatible with a
// config group with a 'cluster' deploy type".
const clusterDeployType = "cluster"

var configGroupColumns = []string{
	"id", "name", "datastoreName", "datastoreVersionName", "deployType", "created",
}

var listConfigGroupsCmd = &cobra.Command{
	Use:   "list-config-groups",
	Short: "List config groups usable by a PostgreSQL Cluster",
	Long: "List the database config groups a cluster can be attached to.\n\n" +
		"The cluster product has no config-group endpoint of its own, so this reads the " +
		"Relational Database config groups and keeps only those with deployType " +
		"'cluster' — the only ones --config-id accepts. Pass --all to see every config " +
		"group, including the single-instance ones.",
	Args: cobra.NoArgs,
	RunE: runListConfigGroups,
}

func init() {
	f := listConfigGroupsCmd.Flags()
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	f.Bool("all", false, "Do not filter by deploy type; list single-instance config groups too")
}

func runListConfigGroups(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	all, _ := cmd.Flags().GetBool("all")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: page, PageSize: pageSize})
	result, err := vdbclient.Get(apiClient, configGroupsPath, query)
	if err != nil {
		return fmt.Errorf("failed to list config groups: %w", err)
	}

	payload := vdbclient.Unwrap(result)
	if !all {
		payload = keepClusterGroups(payload)
	}

	return vdbclient.OutputWithColumns(cmd, payload, configGroupColumns)
}

// keepClusterGroups filters the paginated payload's items in place of the server,
// which offers no deployType filter. This listing keys its items on "content",
// not "data" — 6 of vdb's 8 paginated endpoints do.
func keepClusterGroups(payload interface{}) interface{} {
	obj, ok := payload.(map[string]interface{})
	if !ok {
		return payload
	}
	items, ok := obj["content"].([]interface{})
	if !ok {
		return payload
	}

	kept := make([]interface{}, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if deployType, _ := row["deployType"].(string); deployType == clusterDeployType {
			kept = append(kept, row)
		}
	}

	out := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		out[k] = v
	}
	out["content"] = kept
	return out
}
