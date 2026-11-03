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

// deployTypes are the values the API documents. Only PostgreSQL offers "cluster".
var deployTypes = []string{"single_node", "cluster"}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a config group",
	Long: "Create an empty config group for one engine and version.\n\n" +
		"Creating a group costs nothing and changes no instance: it holds parameters " +
		"until you attach it with 'instance update-config-group'. Set the parameters " +
		"afterwards with 'configuration update'.\n\n" +
		"--deploy-type cluster produces a group usable by PostgreSQL Cluster instead of " +
		"single instances (PostgreSQL only).",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Replace the parameters of a config group",
	Long: "Set database parameters in a config group.\n\n" +
		"THE API REPLACES THE WHOLE PARAMETER SET. Whatever the group held and you do " +
		"not repeat is dropped — verified against the live API, and not what the spec " +
		"suggests. Pass --merge to keep the existing parameters and only add or override " +
		"the ones you name. Either way the command prints the parameters before and " +
		"after, and --dry-run shows the exact request.\n\n" +
		"Repeat --set for each parameter: --set autocommit=1 --set long_query_time=5. " +
		"Values are sent typed — 1 as a number, true as a boolean, anything else as a " +
		"string — because the API's own parameter definitions are typed. Run " +
		"'configuration list-params' for the accepted names, ranges and whether a " +
		"parameter needs a restart.\n\n" +
		"Attached instances may need a reboot for a change to take effect.",
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a config group",
	Long: "Delete a config group.\n\n" +
		"A group that is still attached to instances cannot be deleted — detach it first " +
		"with 'instance update-config-group --detach'. 'configuration get' lists what is " +
		"attached under `instances`; the instanceCount the API returns is always 0 and " +
		"cannot be used for this.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	c := createCmd.Flags()
	c.String("name", "", "Config group name (required)")
	c.String("datastore-type", "", "Engine: MySQL, MariaDB or PostgreSQL (required)")
	c.String("datastore-version", "", "Engine version, e.g. 8.0 (required)")
	c.String("deploy-type", "", fmt.Sprintf("Deploy type: %s (default: single_node)", strings.Join(deployTypes, " or ")))
	c.String("description", "", "Free-text description")
	c.Bool("dry-run", false, "Print the request that would be sent without creating anything")
	createCmd.MarkFlagRequired("name")              //nolint:errcheck
	createCmd.MarkFlagRequired("datastore-type")    //nolint:errcheck
	createCmd.MarkFlagRequired("datastore-version") //nolint:errcheck

	createCmd.RegisterFlagCompletionFunc("datastore-type", cli.ResourceCompletion(datastoreTypeResource))       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("datastore-version", cli.ResourceCompletion(datastoreVersionResource)) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("deploy-type", cli.FlagValues(deployTypes...))                         //nolint:errcheck

	u := updateCmd.Flags()
	u.String("config-id", "", "Config group ID (required)")
	u.StringArray("set", nil, "Parameter as name=value, repeatable (required)")
	u.Bool("merge", false, "Keep the group's existing parameters and only add or override the ones given")
	u.Bool("dry-run", false, "Print the request that would be sent without applying it")
	u.Bool("force", false, "Skip the confirmation prompt")
	updateCmd.MarkFlagRequired("config-id") //nolint:errcheck
	updateCmd.MarkFlagRequired("set")       //nolint:errcheck

	updateCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck

	d := deleteCmd.Flags()
	d.String("config-id", "", "Config group ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteCmd.MarkFlagRequired("config-id") //nolint:errcheck

	deleteCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck
}

func runCreate(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	datastoreType, _ := flags.GetString("datastore-type")
	datastoreVersion, _ := flags.GetString("datastore-version")
	deployType, _ := flags.GetString("deploy-type")
	description, _ := flags.GetString("description")

	if deployType != "" && deployType != "single_node" && deployType != "cluster" {
		return fmt.Errorf("invalid --deploy-type %q: must be %s", deployType, strings.Join(deployTypes, " or "))
	}

	body := map[string]interface{}{
		"name":             name,
		"datastoreType":    datastoreType,
		"datastoreVersion": datastoreVersion,
	}
	if deployType != "" {
		body["deployType"] = deployType
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

	// Read the current parameters: needed for --merge, and shown either way so the
	// replace semantics are visible before anything is sent.
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

	// Two reasons this prints a pointer instead of the result: the update response is
	// a stub (id null, values empty), and the change is applied ASYNCHRONOUSLY — a
	// re-read straight afterwards still returns the old parameters (observed live),
	// so showing it would be worse than showing nothing.
	fmt.Printf("\nAccepted: %d parameter(s) on config group %s.\n", len(values), configID)
	fmt.Println("The API applies this asynchronously — run the following to confirm:")
	fmt.Printf("\n  grn vdb relational configuration get --config-id %s --query values\n", configID)
	return nil
}

// currentValues reads a config group's parameter map.
func currentValues(apiClient *vdbclient.Client, configID string) (map[string]interface{}, error) {
	group, err := fetchGroup(apiClient, configID)
	if err != nil {
		return nil, err
	}
	values, _ := group["values"].(map[string]interface{})
	return values, nil
}

func fetchGroup(apiClient *vdbclient.Client, configID string) (map[string]interface{}, error) {
	query := url.Values{}
	query.Set("id", configID)

	result, err := vdbclient.Get(apiClient, basePath+"/id", query)
	if err != nil {
		return nil, fmt.Errorf("failed to read config group %s: %w", configID, err)
	}
	group, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("config group %s not found (the API returned an empty payload)", configID)
	}
	return group, nil
}

// describeValues renders a parameter map as "autocommit=1, long_query_time=5",
// sorted so two runs can be compared by eye.
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

// plainValue keeps numbers readable. Every JSON number decodes to float64, and %v
// switches to exponent form once that is shorter — a buffer size of 8388608 would
// print as 8.388608e+06.
func plainValue(value interface{}) string {
	if number, ok := value.(float64); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", value)
}

func runDelete(cmd *cobra.Command, args []string) error {
	configID, _ := cmd.Flags().GetString("config-id")
	if err := requireConfigID(configID); err != nil {
		return err
	}

	// A JSON ARRAY body, and no ID in the path — the whole request is the array.
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

	result, err := apiClient.DeleteWithBody(basePath+"/delete", body)
	if err != nil {
		return fmt.Errorf("failed to delete config group %s: %w", configID, err)
	}

	return vdbclient.Output(cmd, result)
}

// parseValues turns each name=value into an entry of the request's `values` map,
// inferring the JSON type.
//
// Typing matters: the API's parameter definitions are typed (integer, boolean,
// string), and its example sends `{"autocommit": 1}` — a number, not "1". Sending
// every value as a string would leave the caller at the mercy of whatever coercion
// the backend happens to do.
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

// typedValue converts a flag value to the JSON type it looks like: an integer or
// float stays a number, true/false become booleans, everything else is a string.
//
// Limitation worth knowing: there is no way to force a numeric-looking value to be
// sent as a string — shell quoting does not survive into the flag value. No
// documented parameter needs that today; if one appears, this needs an explicit
// syntax rather than a guess.
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
