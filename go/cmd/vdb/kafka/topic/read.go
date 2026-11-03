package topic

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// topicColumns matches TopicDto. Every field fits, so this is the whole object
// rather than a subset — Kafka's topics are the one narrow resource in vDB.
var topicColumns = []string{
	"id", "name", "partitions", "replicas", "retentionSeconds", "retentionBytes",
	"status", "createdAt",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the topics of a Kafka cluster",
	Long: "List every topic on a cluster with its partition count, replication factor " +
		"and retention limits.\n\n" +
		"The endpoint takes no pagination and no filters, so this is the complete list.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a Kafka topic",
	Long:  "Show one topic of a cluster.",
	Args:  cobra.NoArgs,
	RunE:  runGet,
}

func init() {
	listCmd.Flags().String("cluster-id", "", "Kafka cluster ID (required)")
	listCmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
	listCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck

	g := getCmd.Flags()
	g.String("cluster-id", "", "Kafka cluster ID (required)")
	g.String("topic-id", "", "Topic ID (required)")
	getCmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
	getCmd.MarkFlagRequired("topic-id")                                    //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("topic-id", topicIDCompletion())     //nolint:errcheck
}

func runList(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validateClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(topicsPath(clusterID), nil)
	if err != nil {
		return fmt.Errorf("failed to list topics of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, topicColumns)
}

func runGet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	topicID, _ := cmd.Flags().GetString("topic-id")
	if err := validateIDs(clusterID, topicID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(topicPath(clusterID, topicID), nil)
	if err != nil {
		return fmt.Errorf("failed to get topic %s of Kafka cluster %s: %w", topicID, clusterID, err)
	}

	// A topic carries no nested arrays, so a column set would be unambiguous here —
	// but a single resource still reads better as a key/value table.
	return vdbclient.Output(cmd, result)
}
