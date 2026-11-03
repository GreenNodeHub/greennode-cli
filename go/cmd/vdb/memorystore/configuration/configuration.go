// Package configuration holds the MemoryStore config-group commands.
//
// Differences from the relational config-group API:
//
//	operation     relational                      memorystore
//	get one       GET  /configurations/id?id=…    GET  /configurations/{configId}/detail
//	delete        DELETE /configurations/delete   POST /configurations/delete
//	create body   has deployType                  has no deployType (Redis has one shape)
//
// The rest — the paginated listing keyed on "content", the params lookup, and the
// update that REPLACES the whole values map — behaves as it does in relational.
package configuration

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// ConfigurationCmd is the parent command for MemoryStore config groups.
var ConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Manage MemoryStore config groups",
	Long: "Create, inspect, change and delete the config groups that hold Redis " +
		"parameters for vDB MemoryStore instances.\n\n" +
		"A group is tied to one engine version and can only be attached to matching " +
		"instances, with 'grn vdb memorystore instance update-config-group'. Use " +
		"'configuration list-params' to see which parameters Redis accepts before setting " +
		"them.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath       = "/vdb-memory/v1/configurations"
	configIDPrefix = "cfg-"
)

// Completion keys owned by the memorystore catalog package.
const (
	datastoreTypeResource    = "vdb:memorystore-datastore-type"
	datastoreVersionResource = "vdb:memorystore-datastore-version"
	configGroupResource      = "vdb:memorystore-config-group"
)

func init() {
	ConfigurationCmd.AddCommand(listCmd)
	ConfigurationCmd.AddCommand(getCmd)
	ConfigurationCmd.AddCommand(createCmd)
	ConfigurationCmd.AddCommand(updateCmd)
	ConfigurationCmd.AddCommand(deleteCmd)
	ConfigurationCmd.AddCommand(listParamsCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

func requireConfigID(configID string) error {
	return vdbclient.RequireIDWithPrefix(configID, "config-id", configIDPrefix,
		"Config group IDs start with 'cfg-'; run 'configuration list' to find one")
}

// listColumns omits deployType, which MemoryStore groups do not carry, and
// instanceCount, which the listing endpoint always answers 0 — verified live on
// 2026-08-13 with a group that had an instance attached. Showing it would tell the user
// a group is unused when it is not; 'configuration get' reports the truth in `instances`.
var listColumns = []string{
	"id", "name", "datastoreName", "datastoreVersionName",
	"description", "created",
}

var paramColumns = []string{
	"name", "type", "min", "max", "modifiable", "restartRequired", "description",
}
