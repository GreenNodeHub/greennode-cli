package configuration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// groupColumns is the table view of a ConfigGroupDto. `versions` is left out: it is a
// nested array, so it cannot be a table cell, and including it would make
// OutputWithColumns treat the versions as the rows. Use 'configuration get' for them.
var groupColumns = []string{"id", "name", "description", "status", "createdAt"}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kafka config groups",
	Long: "List the Kafka config groups in the project.\n\n" +
		"Each group's versions come back nested in its row, so use --output json to see " +
		"them — the version IDs are what a cluster attaches to.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a Kafka config group",
	Long: "Show one config group with all of its versions.\n\n" +
		"This is where the version IDs for " +
		"'grn vdb kafka cluster update-config-group --config-group-version-id' come from.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var getVersionCmd = &cobra.Command{
	Use:   "get-version",
	Short: "Get one version of a Kafka config group",
	Long: "Show a single version of a config group: its properties and the clusters " +
		"currently using it.\n\n" +
		"The associatedClusterIds/associatedClusterNames fields answer 'what breaks if I " +
		"change this?' — check them before applying a different version to a cluster.",
	Args: cobra.NoArgs,
	RunE: runGetVersion,
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Kafka config group",
	Long: "Create a config group and its first version.\n\n" +
		"Properties are given as key=value pairs, repeat --property for each one:\n" +
		"  --property num.partitions=3 --property log.retention.hours=168\n\n" +
		"Kafka has no endpoint listing the valid property names, so they are not " +
		"validated or completed here — a wrong key is rejected by the API.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var createVersionCmd = &cobra.Command{
	Use:   "create-version",
	Short: "Add a version to a Kafka config group",
	Long: "Create a new version of an existing config group.\n\n" +
		"This is how a config group is 'edited': versions are immutable, so a change " +
		"means a new version, which then has to be applied with " +
		"'grn vdb kafka cluster update-config-group'. Creating one changes nothing on " +
		"any cluster by itself.\n\n" +
		"The properties given here are the WHOLE version — they do not merge with the " +
		"previous one. Use --from-version to start from an existing version's " +
		"properties and override only what you pass.",
	Args: cobra.NoArgs,
	RunE: runCreateVersion,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Kafka config group",
	Long: "Delete a config group and every version of it.\n\n" +
		"Check 'configuration get' first: a version in use by a cluster is listed in its " +
		"associatedClusterIds, and deleting the group it belongs to is not something the " +
		"CLI can undo.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	for _, cmd := range []*cobra.Command{getCmd, createVersionCmd, deleteCmd, getVersionCmd} {
		cmd.Flags().String("config-group-id", "", "Kafka config group ID (required)")
		cmd.MarkFlagRequired("config-group-id")                                      //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("config-group-id", configGroupIDCompletion()) //nolint:errcheck
	}

	getVersionCmd.Flags().String("config-group-version-id", "", "Config group version ID (required)")
	getVersionCmd.MarkFlagRequired("config-group-version-id")                                             //nolint:errcheck
	getVersionCmd.RegisterFlagCompletionFunc("config-group-version-id", configGroupVersionIDCompletion()) //nolint:errcheck

	c := createCmd.Flags()
	c.String("name", "", "Name of the new config group (required)")
	c.String("description", "", "Description of the config group")
	c.StringArray("property", nil, "Broker property as key=value; repeat for each one (required)")
	createCmd.MarkFlagRequired("name")     //nolint:errcheck
	createCmd.MarkFlagRequired("property") //nolint:errcheck

	v := createVersionCmd.Flags()
	v.StringArray("property", nil, "Broker property as key=value; repeat for each one")
	v.String("from-version", "", "Start from this version's properties and override only what --property sets")
	v.Bool("dry-run", false, "Print the request that would be sent without creating a version")
	v.Bool("force", false, "Skip the confirmation prompt")
	createVersionCmd.RegisterFlagCompletionFunc("from-version", configGroupVersionIDCompletion()) //nolint:errcheck

	d := deleteCmd.Flags()
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(basePath, nil)
	if err != nil {
		return fmt.Errorf("failed to list Kafka config groups: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, groupColumns)
}

func runGet(cmd *cobra.Command, args []string) error {
	configGroupID, _ := cmd.Flags().GetString("config-group-id")
	if err := validateConfigGroupID(configGroupID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(groupPath(configGroupID), nil)
	if err != nil {
		return fmt.Errorf("failed to get Kafka config group %s: %w", configGroupID, err)
	}

	// A group nests its versions, so print key/value rather than letting table
	// extraction turn the versions into the rows.
	return vdbclient.Output(cmd, result)
}

func runGetVersion(cmd *cobra.Command, args []string) error {
	configGroupID, _ := cmd.Flags().GetString("config-group-id")
	versionID, _ := cmd.Flags().GetString("config-group-version-id")
	if err := validateConfigGroupID(configGroupID); err != nil {
		return err
	}
	if err := validateVersionID(versionID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(versionPath(configGroupID, versionID), nil)
	if err != nil {
		return fmt.Errorf("failed to get version %s of Kafka config group %s: %w",
			versionID, configGroupID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	pairs, _ := cmd.Flags().GetStringArray("property")

	properties, err := parseProperties(pairs)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"properties":  properties,
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(basePath, body)
	if err != nil {
		return fmt.Errorf("failed to create Kafka config group %q: %w", name, err)
	}

	return vdbclient.Output(cmd, result)
}

func runCreateVersion(cmd *cobra.Command, args []string) error {
	configGroupID, _ := cmd.Flags().GetString("config-group-id")
	if err := validateConfigGroupID(configGroupID); err != nil {
		return err
	}

	pairs, _ := cmd.Flags().GetStringArray("property")
	overrides, err := parsePropertyMap(pairs)
	if err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	// --from-version exists because the new version REPLACES rather than extends: the
	// request carries the complete property set, so without a starting point every
	// property has to be retyped to change one. Same trap as relational's
	// `configuration update`, which drops every parameter it is not given.
	base := map[string]string{}
	if fromVersion, _ := cmd.Flags().GetString("from-version"); fromVersion != "" {
		if err := validateVersionID(fromVersion); err != nil {
			return err
		}
		base, err = propertiesOfVersion(apiClient, configGroupID, fromVersion)
		if err != nil {
			return err
		}
	}
	for key, value := range overrides {
		base[key] = value
	}
	if len(base) == 0 {
		return fmt.Errorf("nothing to create: pass --property, or --from-version to copy an existing version")
	}

	body := map[string]interface{}{"properties": propertyList(base)}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create",
			fmt.Sprintf("a version of Kafka config group %s", configGroupID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Create a new version of Kafka config group %s with %d properties? No cluster changes until you apply it.",
		configGroupID, len(base))) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(versionsPath(configGroupID), body)
	if err != nil {
		return fmt.Errorf("failed to create a version of Kafka config group %s: %w", configGroupID, err)
	}

	fmt.Println("Apply it with 'grn vdb kafka cluster update-config-group --config-group-version-id <the id below>'.")
	return vdbclient.Output(cmd, result)
}

func runDelete(cmd *cobra.Command, args []string) error {
	configGroupID, _ := cmd.Flags().GetString("config-group-id")
	if err := validateConfigGroupID(configGroupID); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send DELETE %s\n", groupPath(configGroupID))
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Delete Kafka config group %s and every version of it? Check 'configuration get' for clusters still using one.",
		configGroupID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	// The response is an unspecified bare string; the HTTP status is the result.
	if err := apiClient.NoContent("DELETE", groupPath(configGroupID), nil, nil); err != nil {
		return fmt.Errorf("failed to delete Kafka config group %s: %w", configGroupID, err)
	}

	fmt.Printf("Kafka config group %s deleted.\n", configGroupID)
	return nil
}

// parseProperties turns key=value flags into the API's list of {key, value} objects.
// Note the shape: a LIST of pairs, not the map that relational's config groups use.
func parseProperties(pairs []string) ([]interface{}, error) {
	values, err := parsePropertyMap(pairs)
	if err != nil {
		return nil, err
	}
	return propertyList(values), nil
}

func parsePropertyMap(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --property %q: expected key=value", pair)
		}
		out[key] = value
	}
	return out, nil
}

// propertyList renders the map back as the API's array, sorted so a --dry-run of the
// same input always prints the same thing.
func propertyList(values map[string]string) []interface{} {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]interface{}{"key": key, "value": values[key]})
	}
	return out
}

// propertiesOfVersion reads an existing version's properties for --from-version.
func propertiesOfVersion(apiClient *vdbclient.Client, configGroupID, versionID string) (map[string]string, error) {
	result, err := apiClient.Get(versionPath(configGroupID, versionID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read version %s of Kafka config group %s: %w",
			versionID, configGroupID, err)
	}
	version, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("version %s of Kafka config group %s not found (the API returned an empty payload)",
			versionID, configGroupID)
	}

	out := map[string]string{}
	items, _ := version["properties"].([]interface{})
	for _, item := range items {
		property, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := property["key"].(string)
		value, _ := property["value"].(string)
		if key != "" {
			out[key] = value
		}
	}
	return out, nil
}
