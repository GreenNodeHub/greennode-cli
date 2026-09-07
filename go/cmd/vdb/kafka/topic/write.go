package topic

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Bounds quoted from the API's own field descriptions, checked client-side so the
// error names the flag instead of arriving as a 400.
const (
	minPartitions = 1
	maxPartitions = 2048

	minRetentionSeconds = 3600
	maxRetentionSeconds = 7776000

	maxRetentionBytes = 1099511627776

	// unlimitedRetentionBytes is the documented sentinel for "no size limit". It is
	// the reason --retention-bytes is validated as "-1 or within range" rather than
	// "positive".
	unlimitedRetentionBytes = -1
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a topic on a Kafka cluster",
	Long: "Create a topic.\n\n" +
		"--partitions sets the parallelism (1-2048) and --replicas the replication " +
		"factor, which cannot exceed the cluster's broker count.\n\n" +
		"Retention limits are optional; Kafka applies its own defaults when they are " +
		"omitted. --retention-seconds is 3600-7776000 (1 hour to 90 days), and " +
		"--retention-bytes accepts -1 for unlimited.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Change the partitions, replicas or retention of a topic",
	Long: "Update a topic.\n\n" +
		"Flags left out keep their current values: the endpoint takes the full set on " +
		"every call, so the CLI reads the topic first and repeats what you did not " +
		"change. Without that, omitting a field would reset it.\n\n" +
		"Partition and replica counts normally only go UP in Kafka, and increasing " +
		"partitions changes which partition a key lands on — consumers that rely on " +
		"key ordering are affected.",
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a topic from a Kafka cluster",
	Long: "Delete a topic and every record in it.\n\n" +
		"This is irreversible and Kafka has no backup service, so nothing can bring the " +
		"records back.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	c := createCmd.Flags()
	c.String("cluster-id", "", "Kafka cluster ID (required)")
	c.String("name", "", "Topic name (required)")
	c.Int("partitions", 0, fmt.Sprintf("Partition count, %d-%d (required)", minPartitions, maxPartitions))
	c.Int("replicas", 0, "Replication factor, at least 1 and no more than the broker count (required)")
	c.Int("retention-seconds", 0,
		fmt.Sprintf("Retention time in seconds, %d-%d (default: Kafka's own)", minRetentionSeconds, maxRetentionSeconds))
	c.Int64("retention-bytes", 0,
		fmt.Sprintf("Retention size in bytes, up to %d, or -1 for unlimited (default: Kafka's own)", maxRetentionBytes))
	createCmd.MarkFlagRequired("name")       //nolint:errcheck
	createCmd.MarkFlagRequired("partitions") //nolint:errcheck
	createCmd.MarkFlagRequired("replicas")   //nolint:errcheck

	u := updateCmd.Flags()
	u.String("cluster-id", "", "Kafka cluster ID (required)")
	u.String("topic-id", "", "Topic ID (required)")
	u.Int("partitions", 0, fmt.Sprintf("New partition count, %d-%d (default: unchanged)", minPartitions, maxPartitions))
	u.Int("replicas", 0, "New replication factor (default: unchanged)")
	u.Int("retention-seconds", 0,
		fmt.Sprintf("New retention time in seconds, %d-%d (default: unchanged)", minRetentionSeconds, maxRetentionSeconds))
	u.Int64("retention-bytes", 0,
		fmt.Sprintf("New retention size in bytes, up to %d, or -1 for unlimited (default: unchanged)", maxRetentionBytes))
	u.Bool("dry-run", false, "Print the request that would be sent without changing anything")
	u.Bool("force", false, "Skip the confirmation prompt")

	d := deleteCmd.Flags()
	d.String("cluster-id", "", "Kafka cluster ID (required)")
	d.String("topic-id", "", "Topic ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")

	// Bound here, next to the flags — see the init-order note in completion.go.
	for _, cmd := range []*cobra.Command{createCmd, updateCmd, deleteCmd} {
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
	for _, cmd := range []*cobra.Command{updateCmd, deleteCmd} {
		cmd.MarkFlagRequired("topic-id")                                //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("topic-id", topicIDCompletion()) //nolint:errcheck
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := validateClusterID(clusterID); err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	partitions, _ := cmd.Flags().GetInt("partitions")
	replicas, _ := cmd.Flags().GetInt("replicas")

	if err := checkPartitions(int64(partitions)); err != nil {
		return err
	}
	if replicas < 1 {
		return fmt.Errorf("invalid --replicas %d: the replication factor is at least 1", replicas)
	}

	body := map[string]interface{}{
		"name":       name,
		"partitions": partitions,
		"replicas":   replicas,
	}
	// Retention is only sent when asked for: the API applies Kafka's own defaults for
	// an absent field, and a zero would be a value, not an absence.
	if cmd.Flags().Changed("retention-seconds") {
		seconds, _ := cmd.Flags().GetInt("retention-seconds")
		if err := checkRetentionSeconds(int64(seconds)); err != nil {
			return err
		}
		body["retentionSeconds"] = seconds
	}
	if cmd.Flags().Changed("retention-bytes") {
		bytes, _ := cmd.Flags().GetInt64("retention-bytes")
		if err := checkRetentionBytes(bytes); err != nil {
			return err
		}
		body["retentionBytes"] = bytes
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(topicsPath(clusterID), body)
	if err != nil {
		return fmt.Errorf("failed to create topic %q on Kafka cluster %s: %w", name, clusterID, err)
	}

	// This POST returns the created topic (TopicDto), unwrapped — unlike the PUT and
	// DELETE below, whose responses are bare strings.
	return vdbclient.Output(cmd, result)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	topicID, _ := cmd.Flags().GetString("topic-id")
	if err := validateIDs(clusterID, topicID); err != nil {
		return err
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	// The request carries the whole set, so unspecified flags are filled from the
	// topic as it stands. Sending only what changed would reset the rest — the same
	// trap that made relational's update/setting need the current backup schedule.
	current, err := fetchTopic(apiClient, clusterID, topicID)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"partitions": intFlagOr(cmd, "partitions", current["partitions"]),
		"replicas":   intFlagOr(cmd, "replicas", current["replicas"]),
	}
	// Retention is left OUT when the topic has none and the user asked for none:
	// a topic created without limits reads back retentionSeconds/retentionBytes as
	// null (verified live), and sending 0 would be a value — an invalid one, since
	// the minimum is 3600 seconds / 1 byte — rather than "unchanged".
	if value, ok := optionalInt(cmd, "retention-seconds", current["retentionSeconds"]); ok {
		if err := checkRetentionSeconds(value); err != nil {
			return err
		}
		body["retentionSeconds"] = value
	}
	if value, ok := optionalInt64(cmd, "retention-bytes", current["retentionBytes"]); ok {
		if err := checkRetentionBytes(value); err != nil {
			return err
		}
		body["retentionBytes"] = value
	}

	if err := checkPartitions(body["partitions"].(int64)); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update",
			fmt.Sprintf("topic %s of Kafka cluster %s", topicID, clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Update topic %s of Kafka cluster %s to %v partitions / %v replicas? Increasing partitions changes key-to-partition mapping.",
		topicID, clusterID, body["partitions"], body["replicas"])) {
		fmt.Println("Aborted.")
		return nil
	}

	// The response is an unspecified bare string; the HTTP status is the result.
	if err := apiClient.NoContent("PUT", topicPath(clusterID, topicID), nil, body); err != nil {
		return fmt.Errorf("failed to update topic %s of Kafka cluster %s: %w", topicID, clusterID, err)
	}

	fmt.Printf("Topic %s of Kafka cluster %s updated. Run 'grn vdb kafka topic get --cluster-id %s --topic-id %s' to confirm.\n",
		topicID, clusterID, clusterID, topicID)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	topicID, _ := cmd.Flags().GetString("topic-id")
	if err := validateIDs(clusterID, topicID); err != nil {
		return err
	}

	path := topicPath(clusterID, topicID)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send DELETE %s\n", path)
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Delete topic %s from Kafka cluster %s? Every record in it is lost and Kafka has no backups.",
		topicID, clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return err
	}

	if err := apiClient.NoContent("DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete topic %s from Kafka cluster %s: %w", topicID, clusterID, err)
	}

	fmt.Printf("Topic %s deleted from Kafka cluster %s.\n", topicID, clusterID)
	return nil
}

func checkPartitions(partitions int64) error {
	if partitions < minPartitions || partitions > maxPartitions {
		return fmt.Errorf("invalid --partitions %d: must be between %d and %d",
			partitions, minPartitions, maxPartitions)
	}
	return nil
}

func checkRetentionSeconds(seconds int64) error {
	if seconds < minRetentionSeconds || seconds > maxRetentionSeconds {
		return fmt.Errorf("invalid --retention-seconds %d: must be between %d and %d",
			seconds, minRetentionSeconds, maxRetentionSeconds)
	}
	return nil
}

// checkRetentionBytes allows the documented -1 sentinel alongside the range, and
// allows 0 for "the API's own default" — the only two values outside 1..max that
// mean something.
func checkRetentionBytes(bytes int64) error {
	if bytes == unlimitedRetentionBytes || bytes == 0 {
		return nil
	}
	if bytes < 1 || bytes > maxRetentionBytes {
		return fmt.Errorf("invalid --retention-bytes %d: must be between 1 and %d, or -1 for unlimited",
			bytes, maxRetentionBytes)
	}
	return nil
}

// intFlagOr returns the flag when the user set it and the resource's current value
// otherwise. It returns int64 so callers have one type to check; current values
// arrive as float64, since that is what every JSON number decodes to.
func intFlagOr(cmd *cobra.Command, name string, current interface{}) int64 {
	if cmd.Flags().Changed(name) {
		value, _ := cmd.Flags().GetInt(name)
		return int64(value)
	}
	value, _ := current.(float64)
	return int64(value)
}

// optionalInt and optionalInt64 are intFlagOr for a field that may legitimately be
// absent. The second return says whether the field should be sent at all: false means
// the user named no flag AND the resource carries null, so the request omits it
// rather than asserting a zero.
func optionalInt(cmd *cobra.Command, name string, current interface{}) (int64, bool) {
	if cmd.Flags().Changed(name) {
		value, _ := cmd.Flags().GetInt(name)
		return int64(value), true
	}
	value, ok := current.(float64)
	return int64(value), ok
}

func optionalInt64(cmd *cobra.Command, name string, current interface{}) (int64, bool) {
	if cmd.Flags().Changed(name) {
		value, _ := cmd.Flags().GetInt64(name)
		return value, true
	}
	value, ok := current.(float64)
	return int64(value), ok
}
