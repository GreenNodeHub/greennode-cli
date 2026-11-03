package catalog

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// configGroupsPath is the config groups an instance can be attached to. Note this
// is NOT /vdb-relational/v1/configurations, the paginated management listing — this
// one is the instance-facing lookup and returns a bare array.
const configGroupsPath = basePath + "/configuration"

// clusterDeployType marks a config group built for PostgreSQL Cluster. The endpoint
// returns those alongside the single-instance ones (verified live: IDs prefixed
// "pg-cfg-"), but a Relational Database instance cannot use them.
const clusterDeployType = "cluster"

var configGroupColumns = []string{
	"id", "name", "datastoreName", "datastoreVersionName", "deployType", "created",
}

var listConfigGroupsCmd = &cobra.Command{
	Use:   "list-config-groups",
	Short: "List config groups that can be attached to an instance",
	Long: "List the database config groups available to Relational Database instances.\n\n" +
		"A config group only fits an instance with the same engine and version — check the " +
		"datastoreName / datastoreVersionName columns. Its ID is what --config-id expects " +
		"on 'instance create' and 'instance update-config-group'.\n\n" +
		"The endpoint also returns PostgreSQL Cluster groups (deployType 'cluster', IDs " +
		"prefixed 'pg-cfg-'), which an instance cannot use, so they are filtered out here; " +
		"pass --all to see them, or use 'grn vdb postgresql catalog list-config-groups'.",
	Args: cobra.NoArgs,
	RunE: runListConfigGroups,
}

func init() {
	listConfigGroupsCmd.Flags().Bool("all", false,
		"Do not filter by deploy type; include the PostgreSQL Cluster config groups too")
}

func runListConfigGroups(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(configGroupsPath, nil)
	if err != nil {
		return fmt.Errorf("failed to list config groups: %w", err)
	}

	payload := vdbclient.Unwrap(result)
	if all, _ := cmd.Flags().GetBool("all"); !all {
		payload = dropClusterGroups(payload)
	}

	return vdbclient.OutputWithColumns(cmd, payload, configGroupColumns)
}

// dropClusterGroups removes the cluster config groups from a bare array payload.
// Unlike the paginated /configurations listing there is no wrapper object here, so
// the filter works on the slice directly.
func dropClusterGroups(payload interface{}) interface{} {
	items, ok := payload.([]interface{})
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
			continue
		}
		kept = append(kept, row)
	}
	return kept
}
