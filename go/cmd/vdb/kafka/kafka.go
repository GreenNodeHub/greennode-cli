// Package kafka is the command root for vDB Kafka.
//
// Kafka is the odd one out among the four vDB products, and almost nothing learned
// from relational, memorystore or postgresql transfers. The differences that shape
// every file under here:
//
//   - **No response envelope on most operations.** Only 9 of the 36 endpoints wrap
//     their payload in {code, message, data}; the rest return bare objects, bare
//     arrays or plain strings. vdbclient.Unwrap passes those through untouched, so
//     calling it stays correct — but nothing may ASSUME a wrapper.
//   - **No pagination.** The listings take no pageNumber/pageSize and no filters, so
//     the list commands have no --page or --name.
//   - **Mutating arguments travel in the query string.** Changing the broker count is
//     `PUT /clusters/{id}/kafka-broker-count?count=6&rebalance=true` with an empty
//     body — hence vdbclient.Client.Request, which the verb helpers cannot express.
//   - **Eleven mutating endpoints answer with an unspecified bare string.** Those
//     commands report the HTTP outcome and print nothing of the body; see
//     vdbclient.Client.NoContent.
//   - **Kafka owns nouns no other product has**: topics, users with per-topic
//     permissions, and config groups that are VERSIONED — a cluster is attached to a
//     config group *version*, not to the group.
//
// The path prefix is `/vdb-kafka`, with no `/v1` segment that the other three carry.
package kafka

import (
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/catalog"
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/cluster"
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/configuration"
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/topic"
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/user"
	"github.com/spf13/cobra"
)

// KafkaCmd groups the Kafka commands.
var KafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Manage Kafka clusters",
	Long: "Manage vDB Kafka clusters, their topics, users, config groups and " +
		"security rules.\n\n" +
		"A Kafka cluster is a set of 3-10 brokers. Unlike the other vDB products it has " +
		"no backups, no replicas and no config group attached directly: config groups " +
		"are versioned, and a cluster points at a specific VERSION of one.\n\n" +
		"Access is per-user and per-topic. Create a user with 'kafka user create', give " +
		"it produce/consume/admin permissions on the topics it needs, then read its " +
		"credentials with 'kafka user get-creds'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	KafkaCmd.AddCommand(cluster.ClusterCmd)
	KafkaCmd.AddCommand(topic.TopicCmd)
	KafkaCmd.AddCommand(user.UserCmd)
	KafkaCmd.AddCommand(configuration.ConfigurationCmd)
	KafkaCmd.AddCommand(catalog.CatalogCmd)
}
