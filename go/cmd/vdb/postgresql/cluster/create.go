package cluster

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// passwordEnv lets the master password stay out of shell history and process
// listings. --password still exists for scripting, but the env var is the
// documented way.
const passwordEnv = "GRN_VDB_MASTER_PASSWORD"

const (
	minNodes = 2
	maxNodes = 10
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a PostgreSQL Cluster",
	Long: "Create a vDB PostgreSQL Cluster.\n\n" +
		"THIS COSTS MONEY. Creation goes through the order/payment flow: the API " +
		"answers with an order (orderId, orderUrl) and the cluster is built " +
		"asynchronously — a successful response means the order was placed, not that the " +
		"cluster is ready. Poll 'cluster get' for status. Use --dry-run to see the exact " +
		"request first; --user-type IAM_USER switches from Checkout to Auto Payment.\n\n" +
		"Every ID comes from 'grn vdb postgresql catalog' — its flavors, volume types, " +
		"versions, config groups, backup locations and policies are cluster-specific and " +
		"the relational ones are not interchangeable.\n\n" +
		"The master password is read from $" + passwordEnv + " when --password is omitted, " +
		"which keeps it out of your shell history. It is masked in --dry-run output.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

// createFlags defines the request flags on f. It is a function rather than inline
// init() code so tests can build a throwaway command with the same flag set
// instead of mutating the one mounted in the command tree.
func createFlags(f *pflag.FlagSet) {
	f.String("name", "", "Cluster name (required)")
	f.String("datastore-version", "", "PostgreSQL version, e.g. 16 (required; see 'catalog list-datastores')")
	f.String("package-id", "", "Flavor ID, 'pgp-...' (required; see 'catalog list-flavors')")
	f.String("volume-type-id", "", "Volume type ID, 'pgst-...' (required; see 'catalog list-volume-types')")
	f.Int("volume-size", 0, "Volume size in GB (required)")
	f.Int("number-of-nodes", 3, fmt.Sprintf("Number of nodes (%d-%d)", minNodes, maxNodes))
	f.String("zone-id", "", "Availability zone, e.g. HCM03-1A (required)")
	f.String("subnet-ids", "", "Subnet ID(s), comma-separated (required; see 'grn vdb relational catalog list-subnets')")
	f.String("username", "", "Master username (required)")
	f.String("password", "", "Master password (required; defaults to $"+passwordEnv+")")
	f.String("database-name", "", "Initial database name (required; the API accepts exactly one at creation)")
	f.String("config-id", "", "Config group ID with deploy type 'cluster' (see 'catalog list-config-groups')")
	f.Bool("public-access", false, "Allow public access to the cluster")
	f.String("backup-location-id", "", "Backup location ID, 'bk-des-...' (see 'catalog list-backup-locations')")
	f.String("backup-policy-id", "", "Backup policy ID, 'bk-pol-...' (see 'catalog list-backup-policies')")
	f.String("backup-point-id", "", "Restore point to build the cluster from, 'bk-db-pt-...' (see 'backup list-restore-points')")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	createFlags(createCmd.Flags())

	for _, required := range []string{
		"name", "datastore-version", "package-id", "volume-type-id",
		"volume-size", "zone-id", "subnet-ids", "username", "database-name",
	} {
		createCmd.MarkFlagRequired(required) //nolint:errcheck
	}

	// Bound here, next to the flags: see the init-order note in completion.go.
	createCmd.RegisterFlagCompletionFunc("datastore-version", datastoreVersionCompletion()) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("package-id", packageIDCompletion())               //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("volume-type-id", volumeTypeIDCompletion())        //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())                     //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())                 //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("backup-location-id", backupLocationCompletion())  //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("backup-policy-id", backupPolicyCompletion())      //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("subnet-ids", subnetCompletion())                  //nolint:errcheck
}

func runCreate(cmd *cobra.Command, args []string) error {
	body, err := createBody(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("PostgreSQL Cluster %q", name), body)
		return nil
	}

	nodes := body["numberOfNodes"]
	if !cli.Confirm(force, fmt.Sprintf(
		"Create PostgreSQL Cluster %q with %v nodes? This places a paid order.", name, nodes)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(pgBase, body)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL Cluster %q: %w", name, err)
	}

	return vdbclient.Output(cmd, result)
}

// createBody assembles CreatePostgreClusterRequest. Split out from runCreate so
// the flag-to-field mapping — including the two nested objects and the node-count
// bounds — is testable without a client.
func createBody(cmd *cobra.Command) (map[string]interface{}, error) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	version, _ := flags.GetString("datastore-version")
	packageID, _ := flags.GetString("package-id")
	volumeTypeID, _ := flags.GetString("volume-type-id")
	volumeSize, _ := flags.GetInt("volume-size")
	nodes, _ := flags.GetInt("number-of-nodes")
	zoneID, _ := flags.GetString("zone-id")
	subnetIDs, _ := flags.GetString("subnet-ids")
	username, _ := flags.GetString("username")
	databaseName, _ := flags.GetString("database-name")
	configID, _ := flags.GetString("config-id")
	publicAccess, _ := flags.GetBool("public-access")
	backupLocationID, _ := flags.GetString("backup-location-id")
	backupPolicyID, _ := flags.GetString("backup-policy-id")
	backupPointID, _ := flags.GetString("backup-point-id")
	poc, _ := flags.GetBool("poc")

	if nodes < minNodes || nodes > maxNodes {
		return nil, fmt.Errorf("invalid --number-of-nodes %d: a PostgreSQL Cluster takes %d to %d nodes",
			nodes, minNodes, maxNodes)
	}
	if volumeSize <= 0 {
		return nil, fmt.Errorf("invalid --volume-size %d: must be greater than 0 "+
			"(see minVolumeSize/maxVolumeSize in 'catalog list-volume-types')", volumeSize)
	}

	password, _ := flags.GetString("password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if password == "" {
		return nil, fmt.Errorf("master password not set: pass --password or set $%s", passwordEnv)
	}

	subnets := cli.ParseCommaSeparated(subnetIDs)
	if len(subnets) == 0 {
		return nil, fmt.Errorf("invalid --subnet-ids: at least one subnet ID is required")
	}

	body := map[string]interface{}{
		"name":             name,
		"locateZoneId":     zoneID,
		"packageId":        packageID,
		"volumeTypeId":     volumeTypeID,
		"volumeSize":       volumeSize,
		"numberOfNodes":    nodes,
		"datastoreVersion": version,
		"netIds":           toInterfaces(subnets),
		"publicAccess":     publicAccess,
		"isPoc":            poc,
		"user": map[string]interface{}{
			"name":     username,
			"password": password,
		},
		// The API takes a list but documents that exactly one database may be
		// created with the cluster.
		"databases": []interface{}{
			map[string]interface{}{"name": databaseName},
		},
	}

	// Optional IDs are omitted rather than sent empty: an empty configId detaches
	// a config group on the update endpoint, so empty strings are not inert in
	// this API.
	for key, value := range map[string]string{
		"configId":         configID,
		"backupLocationId": backupLocationID,
		"backupPolicyId":   backupPolicyID,
		"backupPointId":    backupPointID,
	} {
		if value != "" {
			body[key] = value
		}
	}

	return body, nil
}

func toInterfaces(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
