package cluster

import (
	"fmt"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Kafka cluster",
	Long: "Delete a Kafka cluster and everything on it.\n\n" +
		"This is irreversible and there is nothing to restore from: Kafka is the one vDB " +
		"product with no backup service, so the topics and their data go with the " +
		"cluster.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

var resizeBrokersCmd = &cobra.Command{
	Use:   "resize-brokers",
	Short: "Change the number of brokers in a Kafka cluster",
	Long: "Add or remove brokers.\n\n" +
		"THIS COSTS MONEY: order/payment flow, asynchronous. Use --dry-run first.\n\n" +
		"--rebalance moves existing partitions onto the new broker set. Without it the " +
		"brokers change but the partitions stay where they are, so a newly added broker " +
		"carries no traffic until a topic is created or rebalanced.",
	Args: cobra.NoArgs,
	RunE: runResizeBrokers,
}

var resizeStorageCmd = &cobra.Command{
	Use:   "resize-storage",
	Short: "Change the volume size of a Kafka cluster",
	Long: "Change the volume size of every broker.\n\n" +
		"THIS COSTS MONEY: order/payment flow, asynchronous. Use --dry-run first.\n\n" +
		"The size is PER BROKER, and it is a separate operation from the volume type — " +
		"unlike Relational Database, where one resize carries both. Use " +
		"'cluster update-volume-type' for the type.",
	Args: cobra.NoArgs,
	RunE: runResizeStorage,
}

var updateVolumeTypeCmd = &cobra.Command{
	Use:   "update-volume-type",
	Short: "Change the volume type of a Kafka cluster",
	Long: "Move every broker's volume to a different type.\n\n" +
		"THIS COSTS MONEY: like the resize commands this is an order/payment flow and it " +
		"completes asynchronously. Use --dry-run first.\n\n" +
		"Take the value from 'kafka catalog list-volume-types' — do not reuse a volume " +
		"type name from another vDB product, they are not interchangeable.",
	Args: cobra.NoArgs,
	RunE: runUpdateVolumeType,
}

func init() {
	d := deleteCmd.Flags()
	d.String("cluster-id", "", "Kafka cluster ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")

	b := resizeBrokersCmd.Flags()
	b.String("cluster-id", "", "Kafka cluster ID (required)")
	b.Int("broker-count", 0,
		fmt.Sprintf("New number of brokers, %d-%d (required)", minBrokerCount, maxBrokerCount))
	b.Bool("rebalance", false, "Rebalance existing partitions across the new broker set")
	b.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	b.Bool("force", false, "Skip the confirmation prompt")
	resizeBrokersCmd.MarkFlagRequired("broker-count") //nolint:errcheck

	s := resizeStorageCmd.Flags()
	s.String("cluster-id", "", "Kafka cluster ID (required)")
	s.Int("volume-size", 0, "New volume size per broker in GB (required)")
	s.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	s.Bool("force", false, "Skip the confirmation prompt")
	resizeStorageCmd.MarkFlagRequired("volume-size") //nolint:errcheck

	t := updateVolumeTypeCmd.Flags()
	t.String("cluster-id", "", "Kafka cluster ID (required)")
	t.String("volume-type", "", "New volume type (required; 'kafka catalog list-volume-types')")
	t.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	t.Bool("force", false, "Skip the confirmation prompt")
	updateVolumeTypeCmd.MarkFlagRequired("volume-type")                                   //nolint:errcheck
	updateVolumeTypeCmd.RegisterFlagCompletionFunc("volume-type", volumeTypeCompletion()) //nolint:errcheck

	for _, cmd := range []*cobra.Command{deleteCmd, resizeBrokersCmd, resizeStorageCmd, updateVolumeTypeCmd} {
		cmd.MarkFlagRequired("cluster-id")                                  //nolint:errcheck
		cmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	}
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	printDeleteTarget(apiClient, clusterID)

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would send DELETE %s\n", ClusterPath(clusterID, ""))
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Delete Kafka cluster %s? This cannot be undone and Kafka has no backups.", clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	// The spec declares this response a bare string with no schema; the HTTP status
	// is the whole result. See vdbclient.Client.NoContent.
	if err := apiClient.NoContent("DELETE", BasePath+"/"+clusterID, nil, nil); err != nil {
		return fmt.Errorf("failed to delete Kafka cluster %s: %w", clusterID, err)
	}

	fmt.Printf("Deletion of Kafka cluster %s accepted. Poll 'grn vdb kafka cluster list' until it disappears.\n", clusterID)
	return nil
}

func runResizeBrokers(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	brokerCount, _ := cmd.Flags().GetInt("broker-count")
	if brokerCount < minBrokerCount || brokerCount > maxBrokerCount {
		return fmt.Errorf("invalid --broker-count %d: a Kafka cluster has %d to %d brokers",
			brokerCount, minBrokerCount, maxBrokerCount)
	}
	rebalance, _ := cmd.Flags().GetBool("rebalance")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Refuse a no-op rather than letting it become a paid order that fails in the
	// background: a resize to the value already in place is accepted and then fails
	// asynchronously with an empty change description (verified live on the
	// relational product, and there is no reason Kafka differs).
	if current, err := fetchCluster(apiClient, clusterID); err == nil {
		if have := intField(current, "kafkaBrokerCount"); have == brokerCount {
			return fmt.Errorf("Kafka cluster %s already has %d brokers; nothing to do", clusterID, have)
		}
	}

	params := map[string]string{
		"count":     strconv.Itoa(brokerCount),
		"rebalance": strconv.FormatBool(rebalance),
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("resize", fmt.Sprintf("Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/kafka-broker-count"), params)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Change Kafka cluster %s to %d brokers? This places a paid order.", clusterID, brokerCount)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Request("PUT", ClusterPath(clusterID, "/kafka-broker-count"), params, nil)
	if err != nil {
		return fmt.Errorf("failed to resize brokers of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runResizeStorage(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	volumeSize, _ := cmd.Flags().GetInt("volume-size")
	if volumeSize <= 0 {
		return fmt.Errorf("invalid --volume-size %d: must be greater than 0", volumeSize)
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if current, err := fetchCluster(apiClient, clusterID); err == nil {
		if have := intField(current, "kafkaStorageSize"); have == volumeSize {
			return fmt.Errorf("Kafka cluster %s already has %d GB per broker; nothing to do", clusterID, have)
		}
	}

	params := map[string]string{"size": strconv.Itoa(volumeSize)}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("resize", fmt.Sprintf("Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/kafka-storage-size"), params)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Change Kafka cluster %s to %d GB per broker? This places a paid order.", clusterID, volumeSize)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Request("PUT", ClusterPath(clusterID, "/kafka-storage-size"), params, nil)
	if err != nil {
		return fmt.Errorf("failed to resize storage of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runUpdateVolumeType(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := ValidateClusterID(clusterID); err != nil {
		return err
	}

	volumeType, _ := cmd.Flags().GetString("volume-type")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if current, err := fetchCluster(apiClient, clusterID); err == nil {
		if have := stringField(current, "kafkaStorageType"); have == volumeType {
			return fmt.Errorf("Kafka cluster %s already uses volume type %s; nothing to do", clusterID, have)
		}
	}

	params := map[string]string{"storageType": volumeType}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewQuery("update", fmt.Sprintf("Kafka cluster %s", clusterID),
			"PUT", ClusterPath(clusterID, "/kafka-storage-type"), params)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Move Kafka cluster %s to volume type %s? This places a paid order.", clusterID, volumeType)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Request("PUT", ClusterPath(clusterID, "/kafka-storage-type"), params, nil)
	if err != nil {
		return fmt.Errorf("failed to change the volume type of Kafka cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

func printDeleteTarget(apiClient *vdbclient.Client, clusterID string) {
	clusterObj, err := fetchCluster(apiClient, clusterID)
	if err != nil {
		fmt.Printf("Warning: could not read Kafka cluster %s before deleting (%v).\n", clusterID, err)
		return
	}

	fmt.Println("The following Kafka cluster will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", clusterObj["id"])
	fmt.Printf("  Name:    %v\n", clusterObj["name"])
	fmt.Printf("  Status:  %v\n", clusterObj["status"])
	fmt.Printf("  Version: %v\n", clusterObj["kafkaVersion"])
	fmt.Printf("  Brokers: %v x %v GB (%v)\n",
		clusterObj["kafkaBrokerCount"], clusterObj["kafkaStorageSize"], clusterObj["kafkaStorageType"])
	fmt.Println()
}
