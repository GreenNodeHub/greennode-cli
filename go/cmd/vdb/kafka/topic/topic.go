// Package topic holds the Kafka topic commands.
//
// Topics have no equivalent in the other three vDB products: they hang off a
// cluster path (`/vdb-kafka/clusters/{clusterId}/topics`), so every command needs
// --cluster-id alongside the topic's own ID.
package topic

import (
	"github.com/spf13/cobra"
)

// TopicCmd is the parent command for Kafka topic commands.
var TopicCmd = &cobra.Command{
	Use:   "topic",
	Short: "Manage the topics of a Kafka cluster",
	Long: "List, create, update and delete the topics of a vDB Kafka cluster.\n\n" +
		"Every command takes --cluster-id: topics live under a cluster and their IDs are " +
		"only meaningful there.\n\n" +
		"Retention is set per topic, by time (--retention-seconds) and by size " +
		"(--retention-bytes); whichever limit is reached first removes the oldest " +
		"records.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	TopicCmd.AddCommand(listCmd)
	TopicCmd.AddCommand(getCmd)
	TopicCmd.AddCommand(createCmd)
	TopicCmd.AddCommand(updateCmd)
	TopicCmd.AddCommand(deleteCmd)
}
