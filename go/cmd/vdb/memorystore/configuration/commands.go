package configuration

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List config groups",
	Long: "List the MemoryStore config groups in your project.\n\n" +
		"Results are paginated; --page is 1-based and the items come back under 'content'.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a config group",
	Long: "Show one config group, including every parameter it sets.\n\n" +
		"The parameters live in the nested 'values' map, so this prints a key/value view " +
		"rather than a table. Note the path: /configurations/{id}/detail, where the " +
		"relational API takes the ID as a query parameter instead.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a config group",
	Long: "Create an empty config group for one Redis version.\n\n" +
		"Creating a group costs nothing and changes no instance: it holds parameters until " +
		"you attach it with 'instance update-config-group'. Set the parameters afterwards " +
		"with 'configuration update'.\n\n" +
		"Unlike the relational equivalent there is no --deploy-type: MemoryStore has one " +
		"deployment shape.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Replace the parameters of a config group",
	Long: "Set Redis parameters in a config group.\n\n" +
		"THE API REPLACES THE WHOLE PARAMETER SET — verified on the relational endpoint, " +
		"which shares this request. Whatever the group held and you do not repeat is " +
		"dropped. Pass --merge to keep the existing parameters and only add or override " +
		"the ones you name; either way the command prints the parameters before and " +
		"after.\n\n" +
		"Values are sent typed: 1 as a number, true as a boolean, anything else as a " +
		"string. Run 'configuration list-params' for the accepted names and ranges.\n\n" +
		"The change is asynchronous, so a read straight afterwards may still show the old " +
		"parameters.",
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a config group",
	Long: "Delete a config group.\n\n" +
		"A group still attached to instances cannot be deleted — detach it first with " +
		"'instance update-config-group --detach'. 'configuration get' lists what is " +
		"attached under `instances`; the instanceCount the API returns is always 0 and " +
		"cannot be used for this.\n\n" +
		"The request is a POST with a JSON ARRAY body and no ID in the path.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

var listParamsCmd = &cobra.Command{
	Use:   "list-params",
	Short: "List the parameters Redis accepts in a config group",
	Long: "List the Redis parameters that can be set for one version, with their type, " +
		"allowed range or values, and whether changing one restarts the instance.\n\n" +
		"This is the reference for 'configuration update --set': a parameter not listed " +
		"here, or one with modifiable false, will be rejected.",
	Args: cobra.NoArgs,
	RunE: runListParams,
}

func init() {
	l := listCmd.Flags()
	l.Int("page", 1, "Page number (1-based)")
	l.Int("page-size", vdbclient.DefaultPageSize, "Number of items per page")

	g := getCmd.Flags()
	g.String("config-id", "", "Config group ID (required)")
	getCmd.MarkFlagRequired("config-id")                                                        //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck

	c := createCmd.Flags()
	c.String("name", "", "Config group name (required)")
	c.String("datastore-type", "Redis", "Engine (required)")
	c.String("datastore-version", "", "Engine version, e.g. 7.2 (required)")
	c.String("description", "", "Free-text description")
	c.Bool("dry-run", false, "Print the request that would be sent without creating anything")
	createCmd.MarkFlagRequired("name")                                                                          //nolint:errcheck
	createCmd.MarkFlagRequired("datastore-version")                                                             //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("datastore-type", cli.ResourceCompletion(datastoreTypeResource))       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("datastore-version", cli.ResourceCompletion(datastoreVersionResource)) //nolint:errcheck

	u := updateCmd.Flags()
	u.String("config-id", "", "Config group ID (required)")
	u.StringArray("set", nil, "Parameter as name=value, repeatable (required)")
	u.Bool("merge", false, "Keep the group's existing parameters and only add or override the ones given")
	u.Bool("dry-run", false, "Print the request that would be sent without applying it")
	u.Bool("force", false, "Skip the confirmation prompt")
	updateCmd.MarkFlagRequired("config-id")                                                        //nolint:errcheck
	updateCmd.MarkFlagRequired("set")                                                              //nolint:errcheck
	updateCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck

	d := deleteCmd.Flags()
	d.String("config-id", "", "Config group ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteCmd.MarkFlagRequired("config-id")                                                        //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck

	p := listParamsCmd.Flags()
	p.String("datastore-type", "Redis", "Engine (required)")
	p.String("datastore-version", "", "Engine version, e.g. 7.2 (required)")
	listParamsCmd.MarkFlagRequired("datastore-version")                                                             //nolint:errcheck
	listParamsCmd.RegisterFlagCompletionFunc("datastore-type", cli.ResourceCompletion(datastoreTypeResource))       //nolint:errcheck
	listParamsCmd.RegisterFlagCompletionFunc("datastore-version", cli.ResourceCompletion(datastoreVersionResource)) //nolint:errcheck
}

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

	group, err := fetchGroup(apiClient, configID)
	if err != nil {
		return err
	}

	return vdbclient.Output(cmd, group)
}

func runCreate(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	datastoreType, _ := flags.GetString("datastore-type")
	datastoreVersion, _ := flags.GetString("datastore-version")
	description, _ := flags.GetString("description")

	// CreateMemConfigGroupRequest has exactly these four fields — no deployType.
	body := map[string]interface{}{
		"name":             name,
		"datastoreType":    datastoreType,
		"datastoreVersion": datastoreVersion,
	}
	if description != "" {
		body["description"] = description
	}

	if dryRun, _ := flags.GetBool("dry-run"); dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("config group %q", name), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// No confirmation: creating a group is free and touches no instance.
	result, err := apiClient.Post(basePath+"/create", body)
	if err != nil {
		return fmt.Errorf("failed to create config group %q: %w", name, err)
	}

	return vdbclient.Output(cmd, result)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	configID, _ := cmd.Flags().GetString("config-id")
	if err := requireConfigID(configID); err != nil {
		return err
	}

	specs, _ := cmd.Flags().GetStringArray("set")
	values, err := parseValues(specs)
	if err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	existing, err := currentValues(apiClient, configID)
	if err != nil {
		return err
	}

	merge, _ := cmd.Flags().GetBool("merge")
	if merge {
		merged := make(map[string]interface{}, len(existing)+len(values))
		for name, value := range existing {
			merged[name] = value
		}
		for name, value := range values {
			merged[name] = value
		}
		values = merged
	}

	fmt.Printf("Current parameters in %s: %s\n", configID, describeValues(existing))
	fmt.Printf("Parameters after this change: %s\n", describeValues(values))
	if !merge && len(existing) > 0 {
		fmt.Println("\nThis REPLACES the parameter set; anything above that is not in the new set is dropped.")
	}

	body := map[string]interface{}{"id": configID, "values": values}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update", fmt.Sprintf("config group %s", configID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Apply this parameter set to config group %s? Attached instances may need a reboot to pick it up.",
		configID)) {
		fmt.Println("Aborted.")
		return nil
	}

	if _, err := apiClient.Put(basePath+"/update", body); err != nil {
		return fmt.Errorf("failed to update config group %s: %w", configID, err)
	}

	fmt.Printf("\nAccepted: %d parameter(s) on config group %s.\n", len(values), configID)
	fmt.Println("The API applies this asynchronously — run the following to confirm:")
	fmt.Printf("\n  grn vdb memorystore configuration get --config-id %s --query values\n", configID)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	configID, _ := cmd.Flags().GetString("config-id")
	if err := requireConfigID(configID); err != nil {
		return err
	}

	body := []interface{}{
		map[string]interface{}{"id": configID},
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("delete", fmt.Sprintf("config group %s", configID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Delete config group %s?", configID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// POST, not DELETE — the relational API uses DELETE for the same operation.
	result, err := apiClient.Post(basePath+"/delete", body)
	if err != nil {
		return fmt.Errorf("failed to delete config group %s: %w", configID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runListParams(cmd *cobra.Command, args []string) error {
	datastoreType, _ := cmd.Flags().GetString("datastore-type")
	datastoreVersion, _ := cmd.Flags().GetString("datastore-version")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("datastoreType", datastoreType)
	query.Set("datastoreVersion", datastoreVersion)

	result, err := vdbclient.Get(apiClient, basePath+"/params", query)
	if err != nil {
		return fmt.Errorf("failed to list config parameters for %s %s: %w", datastoreType, datastoreVersion, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, paramColumns)
}

// fetchGroup reads one config group. The ID is a path segment ending in /detail.
func fetchGroup(apiClient *vdbclient.Client, configID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(basePath+"/"+configID+"/detail", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read config group %s: %w", configID, err)
	}
	group, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("config group %s not found (the API returned an empty payload)", configID)
	}
	return group, nil
}

func currentValues(apiClient *vdbclient.Client, configID string) (map[string]interface{}, error) {
	group, err := fetchGroup(apiClient, configID)
	if err != nil {
		return nil, err
	}
	values, _ := group["values"].(map[string]interface{})
	return values, nil
}

// parseValues turns each name=value into an entry of the request's values map,
// inferring the JSON type — the API's parameter definitions are typed.
func parseValues(specs []string) (map[string]interface{}, error) {
	values := map[string]interface{}{}

	for _, spec := range specs {
		name, raw, found := strings.Cut(spec, "=")
		name = strings.TrimSpace(name)
		if !found || name == "" {
			return nil, fmt.Errorf("invalid --set %q: expected name=value", spec)
		}
		if _, duplicate := values[name]; duplicate {
			return nil, fmt.Errorf("invalid --set: %q given more than once", name)
		}
		values[name] = typedValue(strings.TrimSpace(raw))
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no parameters given: pass at least one --set name=value")
	}
	return values, nil
}

func typedValue(raw string) interface{} {
	if number, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return number
	}
	if number, err := strconv.ParseFloat(raw, 64); err == nil {
		return number
	}
	switch strings.ToLower(raw) {
	case "true":
		return true
	case "false":
		return false
	}
	return raw
}

// describeValues renders a parameter map sorted, with numbers in plain decimal —
// %v would print a float64 8388608 as 8.388608e+06.
func describeValues(values map[string]interface{}) string {
	if len(values) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%s", name, plainValue(values[name])))
	}
	return strings.Join(parts, ", ")
}

func plainValue(value interface{}) string {
	if number, ok := value.(float64); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", value)
}
