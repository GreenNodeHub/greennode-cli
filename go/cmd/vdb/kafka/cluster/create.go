package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Broker-count bounds, from the API's own description of kafkaBrokerCount
// ("Minimum 3 brokers and maximum 10 brokers"). Checked client-side so the error
// names the flag instead of coming back as a 400 on an order request.
const (
	minBrokerCount = 3
	maxBrokerCount = 10
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Kafka cluster",
	Long: "Create a vDB Kafka cluster.\n\n" +
		"THIS COSTS MONEY: this is an order/payment flow and it completes " +
		"asynchronously. Use --dry-run to see the request first, then poll " +
		"'cluster get' until the status is active.\n\n" +
		"A cluster has 3-10 brokers, each with its own volume of the given type and " +
		"size. --config-group-version-id takes a config group VERSION, not a group: run " +
		"'grn vdb kafka configuration list' and use the id of the version you want.\n\n" +
		"Authentication is off unless --mtls-authen or --sasl-authen is passed. A " +
		"cluster with neither is reachable by anyone who can reach its network, so at " +
		"least one is normally wanted; users and their per-topic permissions are then " +
		"managed with 'grn vdb kafka user'.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Name of the new cluster (required)")
	f.String("kafka-version", "", "Kafka version (required; see 'kafka catalog list-flavors')")
	f.String("flavor-id", "", "Flavor of each broker (required; the 'id' column of 'kafka catalog list-flavors')")
	f.Int("broker-count", minBrokerCount,
		fmt.Sprintf("Number of brokers, %d-%d", minBrokerCount, maxBrokerCount))
	f.String("volume-type", "", "Volume type for each broker (required; 'kafka catalog list-volume-types')")
	f.Int("volume-size", 0, "Volume size per broker in GB (required)")
	f.String("vserver-project-id", "", "vServer project ID (default: the profile's project_id)")
	f.String("network-id", "", "Network (VPC) ID to attach the cluster to (required)")
	f.String("subnet-id", "", "Subnet ID within that network (required)")
	f.Bool("mtls-authen", false, "Enable mTLS authentication")
	f.Bool("sasl-authen", false, "Enable SASL authentication")
	f.String("config-group-version-id", "", "Config group VERSION to apply (optional)")
	f.Bool("encryption-volume", false, "Encrypt the brokers' volumes")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")

	for _, name := range []string{"name", "kafka-version", "flavor-id", "volume-type", "volume-size", "network-id", "subnet-id"} {
		createCmd.MarkFlagRequired(name) //nolint:errcheck
	}

	// Bound here, next to the flags — see the init-order note in completion.go.
	createCmd.RegisterFlagCompletionFunc("kafka-version", kafkaVersionCompletion())                 //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("flavor-id", flavorIDCompletion())                         //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("volume-type", volumeTypeCompletion())                     //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("network-id", cli.ResourceCompletion(networkResource))     //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("subnet-id", cli.ResourceCompletion(subnetResource))       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("config-group-version-id", configGroupVersionCompletion()) //nolint:errcheck
}

// networkResource and subnetResource are the RELATIONAL keys, reused deliberately:
// Kafka has no networks or subnets endpoint of its own, and both are project-wide
// infrastructure. MemoryStore reuses the same pair for the same reason.
const (
	networkResource = "vdb:relational-network"
	subnetResource  = "vdb:relational-subnet"
)

func runCreate(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()

	brokerCount, _ := f.GetInt("broker-count")
	if brokerCount < minBrokerCount || brokerCount > maxBrokerCount {
		return fmt.Errorf("invalid --broker-count %d: a Kafka cluster has %d to %d brokers",
			brokerCount, minBrokerCount, maxBrokerCount)
	}

	volumeSize, _ := f.GetInt("volume-size")
	if volumeSize <= 0 {
		return fmt.Errorf("invalid --volume-size %d: must be greater than 0", volumeSize)
	}

	name, _ := f.GetString("name")
	kafkaVersion, _ := f.GetString("kafka-version")
	flavorID, _ := f.GetString("flavor-id")
	volumeType, _ := f.GetString("volume-type")
	networkID, _ := f.GetString("network-id")
	subnetID, _ := f.GetString("subnet-id")
	mtls, _ := f.GetBool("mtls-authen")
	sasl, _ := f.GetBool("sasl-authen")
	configGroupVersionID, _ := f.GetString("config-group-version-id")
	encryptionVolume, _ := f.GetBool("encryption-volume")

	body := map[string]interface{}{
		"name":             name,
		"kafkaVersion":     kafkaVersion,
		"serverFlavorId":   flavorID,
		"kafkaBrokerCount": brokerCount,
		"kafkaStorageType": volumeType,
		"kafkaStorageSize": volumeSize,
		"vserverProjectId": vserverProjectID(cmd),
		"networkId":        networkID,
		"subnetId":         subnetID,
		"mtlsAuthen":       mtls,
		"saslAuthen":       sasl,
		"encryptionVolume": encryptionVolume,
	}
	// Omitted rather than sent empty: an empty string is a value the API may read as
	// "this version", and there is no live evidence it treats "" as "none" here.
	if configGroupVersionID != "" {
		body["configGroupVersionId"] = configGroupVersionID
	}

	dryRun, _ := f.GetBool("dry-run")
	force, _ := f.GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("Kafka cluster %q", name), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Create Kafka cluster %q with %d brokers of %d GB? This places a paid order.",
		name, brokerCount, volumeSize)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(BasePath, body)
	if err != nil {
		return fmt.Errorf("failed to create Kafka cluster %q: %w", name, err)
	}

	// One of the 9 wrapped Kafka operations: OrderResponse[] inside the usual
	// envelope, whose resourceId is the new cluster's ID.
	return vdbclient.Output(cmd, result)
}

// vserverProjectID falls back to the profile's project_id. The API needs the field
// and the CLI already knows it, so making the user retype it would be busywork —
// the same reason relational's resize-storage reads the instance for its current
// values.
func vserverProjectID(cmd *cobra.Command) string {
	if explicit, _ := cmd.Flags().GetString("vserver-project-id"); explicit != "" {
		return explicit
	}
	profile, _ := cmd.Flags().GetString("profile")
	cfg, err := config.LoadConfig(profile)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.ProjectID
}
