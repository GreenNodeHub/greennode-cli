// Package configuration holds the Relational Database config-group commands.
//
// Three shapes here are unlike the rest of vdb:
//
//   - `GET /configurations/id` takes the ID as a **query parameter**, not a path
//     segment. Building `/configurations/id/<id>` returns 404.
//   - `DELETE /configurations/delete` takes a JSON **ARRAY** body and no ID in the
//     path at all.
//   - `PUT /configurations/update` carries the parameters as a **map** under
//     `values`, whose entries are numbers, booleans or strings depending on the
//     parameter — see update.go.
package configuration

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// ConfigurationCmd is the parent command for config-group commands.
var ConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Manage Relational Database config groups",
	Long: "Create, inspect, change and delete the config groups that hold database " +
		"parameters for vDB Relational Database instances.\n\n" +
		"A config group is tied to one engine and version and can only be attached to " +
		"matching instances, with " +
		"'grn vdb relational instance update-config-group'. Use " +
		"'configuration list-params' to see which parameters an engine accepts before " +
		"setting them.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath        = "/vdb-relational/v1/configurations"
	configIDPrefix  = "cfg-"
	clusterIDPrefix = "pg-cfg-"
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

// requireConfigID accepts both single-instance ("cfg-") and cluster ("pg-cfg-")
// group IDs: this endpoint family serves both, and the deploy type in the payload is
// what tells them apart. It only guards against an ID from another resource
// entirely.
func requireConfigID(configID string) error {
	if err := vdbclient.RequireIDWithPrefix(configID, "config-id", configIDPrefix, ""); err == nil {
		return nil
	}
	return vdbclient.RequireIDWithPrefix(configID, "config-id", clusterIDPrefix,
		"Config group IDs start with 'cfg-' (or 'pg-cfg-' for PostgreSQL Cluster groups); run 'configuration list' to find one")
}

// listColumns is the table view of an ItemConfigInfo. The `values` map and the
// `instances` array are JSON-only: a group can hold dozens of parameters and be
// attached to many instances.
//
// instanceCount is left out although the listing carries it: the API always answers 0,
// verified live on 2026-08-13 against a group with an instance attached. A column
// claiming every group is unused is worse than no column; 'configuration get' reports
// the attachments in `instances`.
var listColumns = []string{
	"id", "name", "datastoreName", "datastoreVersionName", "deployType",
	"description", "created",
}
