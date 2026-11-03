package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List MemoryStore instances",
	Long: "List vDB MemoryStore (Redis) instances.\n\n" +
		"Results are paginated; --page is 1-based. --name and --status filter " +
		"server-side. Unlike the Relational Database listing this one returns only " +
		"MemoryStore instances — no PostgreSQL Clusters mixed in.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a MemoryStore instance",
	Long: "Show the full details of one MemoryStore instance.\n\n" +
		"Note for anyone porting code: the path here is /database-instances/{id}, " +
		"without the /id/ segment the Relational Database API uses.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var listHistoriesCmd = &cobra.Command{
	Use:   "list-histories",
	Short: "List the action history of a MemoryStore instance",
	Long: "List the recorded actions of an instance with their status and, for failures, " +
		"the error message.\n\n" +
		"Results are paginated; --page is 1-based and no filters are accepted. Table " +
		"output omits description, which embeds the whole order cart — use --output json.",
	Args: cobra.NoArgs,
	RunE: runListHistories,
}

var listReplicasCmd = &cobra.Command{
	Use:   "list-replicas",
	Short: "List the read replicas of a MemoryStore instance",
	Long: "List the replicas created from an instance.\n\n" +
		"The API answers with a reduced view of each replica — no ip or port — so run " +
		"'instance get' on a replica ID for its full details.",
	Args: cobra.NoArgs,
	RunE: runListReplicas,
}

func init() {
	l := listCmd.Flags()
	l.Int("page", 1, "Page number (1-based)")
	l.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
	l.String("name", "", "Filter by instance name")
	l.String("status", "", "Filter by status (comma-separated for multiple values)")
	listCmd.RegisterFlagCompletionFunc("status", statusCompletion()) //nolint:errcheck

	for _, cmd := range []*cobra.Command{getCmd, listHistoriesCmd, listReplicasCmd} {
		cmd.Flags().String("instance-id", "", "MemoryStore instance ID (required)")
		cmd.MarkFlagRequired("instance-id")                                   //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
	}

	h := listHistoriesCmd.Flags()
	h.Int("page", 1, "Page number (1-based)")
	h.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")
}

// listOptions maps the list flags onto the shared query. MemoryStore honours the
// flattened name/status filters, same as relational.
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
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, basePath, vdbclient.BuildListQuery(listOptions(cmd)))
	if err != nil {
		return fmt.Errorf("failed to list MemoryStore instances: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, listColumns)
}

func runGet(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := validateInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(detailPath(instanceID), nil)
	if err != nil {
		return fmt.Errorf("failed to get MemoryStore instance %s: %w", instanceID, err)
	}

	// Detail payloads nest several arrays (ip, securityGroup, replicas), so print
	// key/value rather than letting table extraction pick one at random.
	return vdbclient.Output(cmd, result)
}

func runListHistories(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := validateInstanceID(instanceID); err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: page, PageSize: pageSize})
	result, err := vdbclient.Get(apiClient, instancePath(instanceID, "/histories"), query)
	if err != nil {
		return fmt.Errorf("failed to list history of MemoryStore instance %s: %w", instanceID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, historyColumns)
}

func runListReplicas(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := validateInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(instancePath(instanceID, "/replicas"), nil)
	if err != nil {
		return fmt.Errorf("failed to list replicas of MemoryStore instance %s: %w", instanceID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, replicaColumns)
}
