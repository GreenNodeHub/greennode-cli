// Package configuration holds the Kafka config-group commands.
//
// Kafka config groups differ from the ones in Relational Database and MemoryStore in
// the way that matters most: they are VERSIONED and IMMUTABLE. A group is a named
// container; the properties live in a version of it, and a cluster is attached to a
// specific version. There is therefore no update endpoint — changing a property
// means creating a new version and applying that to the cluster.
//
// Two other differences: the properties are a list of {key, value} pairs rather than
// a map, and there is no parameter-catalogue endpoint (relational's
// `configuration list-params`), so the CLI cannot tell you which keys are valid.
package configuration

import (
	"github.com/spf13/cobra"
)

const (
	// basePath is the config-group collection. Note it hangs off /vdb-kafka
	// directly, NOT off a cluster: a group is project-wide and can be applied to
	// several clusters.
	basePath = "/vdb-kafka/config-groups"
)

func groupPath(configGroupID string) string {
	return basePath + "/" + configGroupID
}

func versionsPath(configGroupID string) string {
	return groupPath(configGroupID) + "/versions"
}

func versionPath(configGroupID, versionID string) string {
	return versionsPath(configGroupID) + "/" + versionID
}

// ConfigurationCmd is the parent command for Kafka config groups.
var ConfigurationCmd = &cobra.Command{
	Use:   "configuration",
	Short: "Manage Kafka config groups",
	Long: "Manage the config groups that carry Kafka broker properties.\n\n" +
		"Kafka config groups are VERSIONED: a group holds versions, a version holds the " +
		"properties, and a cluster is attached to one version. There is no update " +
		"command because a version cannot be edited — use 'create-version' and then " +
		"'grn vdb kafka cluster update-config-group' to apply it.\n\n" +
		"Unlike the other vDB products, Kafka has no endpoint listing the valid " +
		"property names, so there is no 'list-params' here.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	ConfigurationCmd.AddCommand(listCmd)
	ConfigurationCmd.AddCommand(getCmd)
	ConfigurationCmd.AddCommand(getVersionCmd)
	ConfigurationCmd.AddCommand(createCmd)
	ConfigurationCmd.AddCommand(createVersionCmd)
	ConfigurationCmd.AddCommand(deleteCmd)
}
