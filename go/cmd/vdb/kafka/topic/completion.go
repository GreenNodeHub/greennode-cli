package topic

import (
	"context"
	"fmt"

	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka/cluster"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers live here; flags are bound in the init() of the file that DEFINES
// them — see the init-order note in cmd/vdb/CLAUDE.md.

// TopicResourceKey lists the topic IDs of the cluster named on the command line. It
// is CONTEXT-DEPENDENT: topics are scoped to a cluster, so it yields nothing until
// --cluster-id is set.
const TopicResourceKey = "vdb:kafka-topic"

// TopicNameResourceKey lists topic NAMES rather than IDs. The user permission flags
// take names, not IDs — the one place in the Kafka API where a topic is referenced
// by name — so a completer that offered IDs there would suggest values the API
// silently ignores.
const TopicNameResourceKey = "vdb:kafka-topic-name"

func init() {
	cli.RegisterResourceCompleter(TopicResourceKey, cli.FlagFromAPI(topicField("id")))
	cli.RegisterResourceCompleter(TopicNameResourceKey, cli.FlagFromAPI(topicField("name")))
}

func clusterIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(cluster.ClusterResourceKey)
}

func topicIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(TopicResourceKey)
}

// topicField reads one field from every topic of the cluster on the command line.
func topicField(field string) func(context.Context, *cobra.Command) ([]string, error) {
	return func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		clusterID, _ := cmd.Flags().GetString("cluster-id")
		if clusterID == "" {
			return nil, nil
		}
		apiClient, err := vdbclient.BuildClient(cmd)
		if err != nil {
			return nil, err
		}
		result, err := apiClient.Get(topicsPath(clusterID), nil)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(vdbclient.Unwrap(result), field), nil
	}
}

// topicsPath is the topic collection of a cluster. The path is built from the
// cluster package's helper rather than by string-joining "/vdb-kafka/clusters" here:
// one owner per path prefix, per the rule in cmd/vdb/CLAUDE.md.
func topicsPath(clusterID string) string {
	return cluster.ClusterPath(clusterID, "/topics")
}

func topicPath(clusterID, topicID string) string {
	return topicsPath(clusterID) + "/" + topicID
}

func validateClusterID(clusterID string) error {
	return cluster.ValidateClusterID(clusterID)
}

func validateIDs(clusterID, topicID string) error {
	if err := cluster.ValidateClusterID(clusterID); err != nil {
		return err
	}
	return validator.ValidateID(topicID, "topic-id")
}

// fetchTopic reads one topic, reporting a null payload as a miss rather than
// handing back zeroed fields.
func fetchTopic(apiClient *vdbclient.Client, clusterID, topicID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(topicPath(clusterID, topicID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read topic %s of Kafka cluster %s: %w", topicID, clusterID, err)
	}
	topicObj, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("topic %s of Kafka cluster %s not found (the API returned an empty payload)",
			topicID, clusterID)
	}
	return topicObj, nil
}
