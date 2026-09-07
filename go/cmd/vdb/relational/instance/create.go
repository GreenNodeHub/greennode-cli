package instance

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// passwordEnv keeps the master password out of shell history and process listings.
// --password still exists for scripting, but the env var is the documented way.
const passwordEnv = "GRN_VDB_MASTER_PASSWORD"

// Backup retention bounds, from the API docs for backupDuration.
const (
	minBackupDuration = 2
	maxBackupDuration = 14
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Relational Database instance",
	Long: "Create a vDB Relational Database instance (MySQL, MariaDB or PostgreSQL).\n\n" +
		"THIS COSTS MONEY. Creation goes through the order/payment flow: the API answers " +
		"with an order (orderId, orderUrl) and the instance is built asynchronously — a " +
		"successful response means the order was placed, not that the database is ready. " +
		"Poll 'instance get' for status. Use --dry-run to see the exact request first; " +
		"--user-type IAM_USER switches from Checkout to Auto Payment.\n\n" +
		"Values come from 'grn vdb relational catalog'. Two of them are easy to mix up " +
		"with the PostgreSQL Cluster group: --package-id is the numeric flavor id from " +
		"'catalog list-flavors' (e.g. 211), and --volume-type is the volume type NAME " +
		"(e.g. Gen2-NVMe2-IOPS3000), not an ID.\n\n" +
		"The master password is read from $" + passwordEnv + " when --password is omitted. " +
		"It is masked in --dry-run output.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

// createFlags defines the request flags on f. A function rather than inline init()
// code so tests can build a throwaway command with the same flag set instead of
// mutating the one mounted in the command tree.
func createFlags(f *pflag.FlagSet) {
	f.String("name", "", "Instance name (required)")
	f.String("datastore-type", "", "Engine: MySQL, MariaDB or PostgreSQL (required; see 'catalog list-datastores')")
	f.String("datastore-version", "", "Engine version, e.g. 8.0 (required; see 'catalog list-datastores')")
	f.String("package-id", "", "Flavor id, numeric, e.g. 211 (required; the 'id' column of 'catalog list-flavors')")
	f.String("volume-type", "", "Volume type NAME, e.g. Gen2-NVMe2-IOPS3000 (required; the 'type' column of 'catalog list-volume-types')")
	f.Int("volume-size", 0, "Volume size in GB (required)")
	f.String("zone-id", "", "Availability zone, e.g. HCM03-1A (required)")
	f.String("subnet-ids", "", "Subnet ID(s), comma-separated (required; see 'catalog list-subnets')")
	f.String("username", "", "Master username (required)")
	f.String("password", "", "Master password (required; defaults to $"+passwordEnv+")")
	f.String("database-name", "", "Initial database name (required; the API accepts exactly one at creation)")
	f.String("character-set", "", "Character set of the initial database, e.g. utf8mb4 (MySQL/MariaDB)")
	f.String("collate", "", "Collation of the initial database, e.g. utf8mb4_general_ci (MySQL/MariaDB)")
	f.String("config-id", "", "Config group ID to attach (see 'catalog list-config-groups')")
	f.Bool("public-access", false, "Allow public access to the instance")
	f.Bool("backup-auto", false, "Enable daily automatic backup")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d (required with --backup-auto)", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM (required with --backup-auto)")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	createFlags(createCmd.Flags())

	for _, required := range []string{
		"name", "datastore-type", "datastore-version", "package-id",
		"volume-type", "volume-size", "zone-id", "subnet-ids", "username", "database-name",
	} {
		createCmd.MarkFlagRequired(required) //nolint:errcheck
	}

	// Bound here, next to the flags: see the init-order note in completion.go.
	createCmd.RegisterFlagCompletionFunc("datastore-type", datastoreTypeCompletion())       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("datastore-version", datastoreVersionCompletion()) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("volume-type", volumeTypeCompletion())             //nolint:errcheck
	// --package-id completes only once --datastore-type and --datastore-version are
	// on the command line: the flavors endpoint requires them.
	createCmd.RegisterFlagCompletionFunc("package-id", packageIDCompletion()) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("subnet-ids", subnetCompletion())    //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())   //nolint:errcheck
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
		vdbclient.PreviewBody("create", fmt.Sprintf("database instance %q", name), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Create %v %v instance %q? This places a paid order.",
		body["datastoreType"], body["datastoreVersion"], name)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(paymentPath, body)
	if err != nil {
		return fmt.Errorf("failed to create database instance %q: %w", name, err)
	}

	return vdbclient.Output(cmd, result)
}

// createBody assembles CreateDbInstanceRequest. Split out from runCreate so the
// flag-to-field mapping and the backup rules are testable without a client.
func createBody(cmd *cobra.Command) (map[string]interface{}, error) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	datastoreType, _ := flags.GetString("datastore-type")
	datastoreVersion, _ := flags.GetString("datastore-version")
	packageID, _ := flags.GetString("package-id")
	volumeType, _ := flags.GetString("volume-type")
	volumeSize, _ := flags.GetInt("volume-size")
	zoneID, _ := flags.GetString("zone-id")
	subnetIDs, _ := flags.GetString("subnet-ids")
	username, _ := flags.GetString("username")
	databaseName, _ := flags.GetString("database-name")
	characterSet, _ := flags.GetString("character-set")
	collate, _ := flags.GetString("collate")
	configID, _ := flags.GetString("config-id")
	publicAccess, _ := flags.GetBool("public-access")
	poc, _ := flags.GetBool("poc")

	if volumeSize <= 0 {
		return nil, fmt.Errorf("invalid --volume-size %d: must be greater than 0 "+
			"(see minVolumeSize/maxVolumeSize in 'catalog list-volume-types')", volumeSize)
	}

	password, err := masterPassword(flags)
	if err != nil {
		return nil, err
	}

	subnets := cli.ParseCommaSeparated(subnetIDs)
	if len(subnets) == 0 {
		return nil, fmt.Errorf("invalid --subnet-ids: at least one subnet ID is required")
	}

	database := map[string]interface{}{"name": databaseName}
	if characterSet != "" {
		database["characterSet"] = characterSet
	}
	if collate != "" {
		database["collate"] = collate
	}

	body := map[string]interface{}{
		"name":             name,
		"datastoreType":    datastoreType,
		"datastoreVersion": datastoreVersion,
		"packageId":        packageID,
		"volumeType":       volumeType,
		"volumeSize":       volumeSize,
		"locateZoneId":     zoneID,
		"netIds":           toInterfaces(subnets),
		"publicAccess":     publicAccess,
		"poc":              poc,
		"user": map[string]interface{}{
			"name":     username,
			"password": password,
		},
		// The API takes a list but documents that exactly one database may be
		// created with the instance.
		"databases": []interface{}{database},
	}

	if configID != "" {
		// Omitted rather than sent empty: an empty configId means "detach" on the
		// update endpoint, so empty strings are not inert in this API.
		body["configId"] = configID
	}

	if err := applyBackupFlags(flags, body); err != nil {
		return nil, err
	}

	return body, nil
}

// applyBackupFlags adds the three backup fields. The API requires duration and
// time whenever backupAuto is on, and ignores them otherwise, so they are only
// sent together.
func applyBackupFlags(flags *pflag.FlagSet, body map[string]interface{}) error {
	backupAuto, _ := flags.GetBool("backup-auto")
	duration, _ := flags.GetInt("backup-duration")
	backupTime, _ := flags.GetString("backup-time")

	body["backupAuto"] = backupAuto

	if !backupAuto {
		if duration != 0 || backupTime != "" {
			return fmt.Errorf("--backup-duration and --backup-time only apply with --backup-auto")
		}
		return nil
	}

	if duration < minBackupDuration || duration > maxBackupDuration {
		return fmt.Errorf("--backup-auto requires --backup-duration between %d and %d days, got %d",
			minBackupDuration, maxBackupDuration, duration)
	}
	if backupTime == "" {
		return fmt.Errorf("--backup-auto requires --backup-time, e.g. --backup-time 02:00")
	}

	body["backupDuration"] = duration
	body["backupTime"] = backupTime
	return nil
}

// masterPassword resolves --password, falling back to the environment, and checks it
// against the product's rules before it can reach a request body.
func masterPassword(flags *pflag.FlagSet) (string, error) {
	password, _ := flags.GetString("password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if password == "" {
		return "", fmt.Errorf("master password not set: pass --password or set $%s", passwordEnv)
	}
	if err := vdbclient.ValidateDBPassword(password, "password"); err != nil {
		return "", err
	}
	return password, nil
}

func toInterfaces(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
