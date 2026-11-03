package configuration

import (
	"fmt"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List config groups",
	Long: "List the config groups in your project.\n\n" +
		"Results are paginated; --page is 1-based and the items come back under " +
		"'content'. The listing includes PostgreSQL Cluster groups (deployType " +
		"'cluster'); 'grn vdb relational catalog list-config-groups' is the filtered " +
		"view meant for attaching to a single instance.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a config group",
	Long: "Show one config group, including every parameter it sets.\n\n" +
		"The parameters live in the nested 'values' map and the attached instances in " +
		"'instances', so this prints a key/value view rather than a table.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var listParamsCmd = &cobra.Command{
	Use:   "list-params",
	Short: "List the parameters an engine accepts in a config group",
	Long: "List the database parameters that can be set for one engine and version, with " +
		"their type, allowed range or values, and whether changing one restarts the " +
		"database.\n\n" +
		"This is the reference for 'configuration update --set': a parameter not listed " +
		"here, or one with modifiable false, will be rejected.",
	Args: cobra.NoArgs,
	RunE: runListParams,
}

var paramColumns = []string{
	"name", "type", "min", "max", "modifiable", "restartRequired", "description",
}

func init() {
	l := listCmd.Flags()
	l.Int("page", 1, "Page number (1-based)")
	l.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")

	g := getCmd.Flags()
	g.String("config-id", "", "Config group ID (required)")
	getCmd.MarkFlagRequired("config-id")                                                        //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck

	p := listParamsCmd.Flags()
	p.String("datastore-type", "", "Engine: MySQL, MariaDB or PostgreSQL (required)")
	p.String("datastore-version", "", "Engine version, e.g. 8.0 (required)")
	p.String("deploy-type", "", "Deploy type: single_node or cluster (PostgreSQL only)")
	listParamsCmd.MarkFlagRequired("datastore-type")    //nolint:errcheck
	listParamsCmd.MarkFlagRequired("datastore-version") //nolint:errcheck

	listParamsCmd.RegisterFlagCompletionFunc("datastore-type", cli.ResourceCompletion(datastoreTypeResource))       //nolint:errcheck
	listParamsCmd.RegisterFlagCompletionFunc("datastore-version", cli.ResourceCompletion(datastoreVersionResource)) //nolint:errcheck
	listParamsCmd.RegisterFlagCompletionFunc("deploy-type", cli.FlagValues(deployTypes...))                         //nolint:errcheck
}

// Completion keys owned by the relational catalog package.
const (
	configGroupResource      = "vdb:relational-config-group"
	datastoreTypeResource    = "vdb:relational-datastore-type"
	datastoreVersionResource = "vdb:relational-datastore-version"
)

func runList(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: page, PageSize: pageSize})
	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return fmt.Errorf("failed to list config groups: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, listColumns)
}

func runGet(cmd *cobra.Command, args []string) error {
	configID, _ := cmd.Flags().GetString("config-id")
	if err := requireConfigID(configID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// The ID is a QUERY parameter here, not a path segment: /configurations/id?id=…
	query := url.Values{}
	query.Set("id", configID)

	result, err := vdbclient.Get(apiClient, basePath+"/id", query)
	if err != nil {
		return fmt.Errorf("failed to get config group %s: %w", configID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runListParams(cmd *cobra.Command, args []string) error {
	datastoreType, _ := cmd.Flags().GetString("datastore-type")
	datastoreVersion, _ := cmd.Flags().GetString("datastore-version")
	deployType, _ := cmd.Flags().GetString("deploy-type")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("datastoreType", datastoreType)
	query.Set("datastoreVersion", datastoreVersion)
	if deployType != "" {
		query.Set("deployType", deployType)
	}

	result, err := vdbclient.Get(apiClient, basePath+"/params", query)
	if err != nil {
		return fmt.Errorf("failed to list config parameters for %s %s: %w", datastoreType, datastoreVersion, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, paramColumns)
}
