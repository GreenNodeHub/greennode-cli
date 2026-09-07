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
	Short: "List Kafka clusters",
	Long: "List the vDB Kafka clusters in the project.\n\n" +
		"The endpoint takes no pagination and no filters — it returns every cluster in " +
		"one response — so there is no --page here, and --name and --status are applied " +
		"by the CLI after the fact rather than by the server.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a Kafka cluster",
	Long: "Show the full details of one Kafka cluster, including its broker addresses, " +
		"authentication switches and security rules.\n\n" +
		"This is the only place the security rules and the attached config group version " +
		"are visible: Kafka has no separate endpoint for either.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var listHistoriesCmd = &cobra.Command{
	Use:   "list-histories",
	Short: "List the action history of a Kafka cluster",
	Long: "List the recorded actions of a cluster with their status and, for failures, " +
		"the error message.\n\n" +
		"Kafka records history in its own shape: startedAt/finishedAt rather than the " +
		"createdTime/updatedTime the other vDB products use, and no pagination.",
	Args: cobra.NoArgs,
	RunE: runListHistories,
}

var listSecrulesCmd = &cobra.Command{
	Use:   "list-secrules",
	Short: "List the security rules of a Kafka cluster",
	Long: "List the remote IP / port pairs allowed to reach the cluster.\n\n" +
		"Kafka has no endpoint that lists rules: they are read out of the cluster " +
		"object, which is what this command does. Their IDs are what " +
		"'cluster delete-secrule --secrule-id' expects.",
	Args: cobra.NoArgs,
	RunE: runListSecrules,
}

func init() {
	l := listCmd.Flags()
	l.String("name", "", "Show only clusters whose name contains this text (applied client-side)")
	l.String("status", "", "Show only clusters in these statuses (comma-separated, applied client-side)")
	listCmd.RegisterFlagCompletionFunc("status", statusCompletion()) //nolint:errcheck

	for _, cmd := range []*cobra.Command{getCmd, listHistoriesCmd, listSecrulesCmd} {
		cmd.Flags().String("cluster-id", "", "Kafka cluster ID (required)")
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(BasePath, nil)
	if err != nil {
		return fmt.Errorf("failed to list Kafka clusters: %w", err)
	}

	name, _ := cmd.Flags().GetString("name")
	status, _ := cmd.Flags().GetString("status")
	filtered := filterClusters(vdbclient.Unwrap(result), name, cli.ParseCommaSeparated(status))

	return vdbclient.OutputWithColumns(cmd, filtered, listColumns)
}

// filterClusters applies --name and --status in the CLI. The endpoint accepts no
// filter parameters at all, so doing it here is the only option; it also means the
// result is the complete match set rather than one page of it. Matching is
// case-insensitive because vDB status values are mixed-case (ACTIVE alongside
// lowercase "deleting").
func filterClusters(payload interface{}, name string, statuses []string) interface{} {
	items, ok := payload.([]interface{})
	if !ok || (name == "" && len(statuses) == 0) {
		return payload
	}

	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name != "" && !strings.Contains(
			strings.ToLower(stringField(row, "name")), strings.ToLower(name)) {
			continue
		}
		if len(statuses) > 0 && !matchesAnyStatus(stringField(row, "status"), statuses) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func matchesAnyStatus(value string, statuses []string) bool {
	for _, want := range statuses {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

func runGet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(BasePath+"/"+clusterID, nil)
	if err != nil {
		return fmt.Errorf("failed to get Kafka cluster %s: %w", clusterID, err)
	}

	// A cluster nests several arrays (fixedIps, floatingIps, securityGroupRules,
	// kafkaStorageUsage), so print key/value rather than letting table extraction
	// pick one of them at random.
	return vdbclient.Output(cmd, result)
}

func runListHistories(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(ClusterPath(clusterID, "/history"), nil)
	if err != nil {
		return fmt.Errorf("failed to list history of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, historyColumns)
}

func runListSecrules(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	clusterObj, err := fetchCluster(apiClient, clusterID)
	if err != nil {
		return err
	}

	rules, _ := clusterObj["securityGroupRules"].([]interface{})
	if rules == nil {
		// An empty slice prints as an empty list; a nil would print as "null".
		rules = []interface{}{}
	}
	return vdbclient.OutputWithColumns(cmd, rules, secruleColumns)
}
