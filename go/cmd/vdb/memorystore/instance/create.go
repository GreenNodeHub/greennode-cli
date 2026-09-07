package instance

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// passwordEnv keeps the Redis master password out of shell history. The relational
// group uses the same variable for its master password.
const passwordEnv = "GRN_VDB_MASTER_PASSWORD"

const (
	minBackupDuration = 2
	maxBackupDuration = 14
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a MemoryStore instance",
	Long: "Create a vDB MemoryStore (Redis) instance.\n\n" +
		"THIS COSTS MONEY. Creation goes through the order/payment flow: the API answers " +
		"with an order (orderId, orderUrl) and builds the instance asynchronously — a " +
		"successful response means the order was placed, not that Redis is ready. Poll " +
		"'instance get'. Use --dry-run first; --user-type IAM_USER switches from Checkout " +
		"to Auto Payment.\n\n" +
		"No storage flags: a MemoryStore instance has no volume, its capacity is the " +
		"flavor's RAM. --package-id is the numeric flavor id from " +
		"'catalog list-flavors'.\n\n" +
		"Access is a master password on the instance, not a database user: " +
		"--redis-password (or $" + passwordEnv + ") sets it, and it is REQUIRED whenever " +
		"--public-access is on. The password is masked in --dry-run output.\n\n" +
		"Zones and subnets come from the Relational Database catalog — MemoryStore has " +
		"none of its own — so use 'grn vdb relational catalog list-zones' and " +
		"'list-subnets'.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

func createFlags(f *pflag.FlagSet) {
	f.String("name", "", "Instance name (required)")
	f.String("datastore-type", "Redis", "Engine (required; see 'catalog list-datastores')")
	f.String("datastore-version", "", "Engine version, e.g. 7.2 (required)")
	f.String("package-id", "", "Flavor id, numeric (required; the 'id' column of 'catalog list-flavors')")
	f.String("zone-id", "", "Availability zone, e.g. HCM03-1A (required)")
	f.String("subnet-ids", "", "Subnet ID(s), comma-separated (required)")
	f.String("redis-password", "", "Redis master password (defaults to $"+passwordEnv+")")
	f.Bool("redis-password-enabled", false, "Require a master password to connect (implied by --public-access)")
	f.String("config-id", "", "Config group ID to attach (see 'catalog list-config-groups')")
	f.Bool("public-access", false, "Allow public access — requires a master password")
	f.Bool("backup-auto", false, "Enable daily automatic backup")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d (required with --backup-auto)", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM (required with --backup-auto)")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	createFlags(createCmd.Flags())

	for _, required := range []string{"name", "datastore-version", "package-id", "zone-id", "subnet-ids"} {
		createCmd.MarkFlagRequired(required) //nolint:errcheck
	}

	// Bound here, next to the flags: see the init-order note in completion.go.
	createCmd.RegisterFlagCompletionFunc("datastore-type", datastoreTypeCompletion())       //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("datastore-version", datastoreVersionCompletion()) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())                     //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("subnet-ids", subnetCompletion())                  //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())                 //nolint:errcheck
	// The flavor completer reads --datastore-version (and --zone-id when given) off this
	// same command line, so it yields nothing until the version is set.
	createCmd.RegisterFlagCompletionFunc("package-id", packageIDCompletion()) //nolint:errcheck
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
		vdbclient.PreviewBody("create", fmt.Sprintf("MemoryStore instance %q", name), body)
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
		return fmt.Errorf("failed to create MemoryStore instance %q: %w", name, err)
	}

	return vdbclient.Output(cmd, result)
}

// createBody assembles CreateMemDbInstanceRequest.
//
// Compared with the relational create it has no `user`, no `databases`, no
// `volumeType` and no `volumeSize` — Redis has neither users nor a volume — and adds
// the two password fields.
func createBody(cmd *cobra.Command) (map[string]interface{}, error) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	datastoreType, _ := flags.GetString("datastore-type")
	datastoreVersion, _ := flags.GetString("datastore-version")
	packageID, _ := flags.GetString("package-id")
	zoneID, _ := flags.GetString("zone-id")
	subnetIDs, _ := flags.GetString("subnet-ids")
	configID, _ := flags.GetString("config-id")
	publicAccess, _ := flags.GetBool("public-access")
	poc, _ := flags.GetBool("poc")

	subnets := cli.ParseCommaSeparated(subnetIDs)
	if len(subnets) == 0 {
		return nil, fmt.Errorf("invalid --subnet-ids: at least one subnet ID is required")
	}

	body := map[string]interface{}{
		"name":             name,
		"datastoreType":    datastoreType,
		"datastoreVersion": datastoreVersion,
		"packageId":        packageID,
		"locateZoneId":     zoneID,
		"netIds":           toInterfaces(subnets),
		"publicAccess":     publicAccess,
		"poc":              poc,
	}
	if configID != "" {
		body["configId"] = configID
	}

	if err := applyPasswordFlags(flags, body, publicAccess); err != nil {
		return nil, err
	}
	if err := applyBackupFlags(flags, body); err != nil {
		return nil, err
	}

	return body, nil
}

// applyPasswordFlags handles the Redis authentication pair and the one cross-field
// rule the API states: public access requires the master password to be enabled.
// Enforcing it here turns a 400 into a message that names the flags.
func applyPasswordFlags(flags *pflag.FlagSet, body map[string]interface{}, publicAccess bool) error {
	password, _ := flags.GetString("redis-password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}

	enabled, _ := flags.GetBool("redis-password-enabled")
	// A password given without the flag plainly means "enable it"; and public access
	// requires it whether or not the user thought about it.
	if password != "" || publicAccess {
		enabled = true
	}

	if enabled && password == "" {
		if publicAccess {
			return fmt.Errorf("--public-access requires a master password: pass --redis-password or set $%s", passwordEnv)
		}
		return fmt.Errorf("--redis-password-enabled requires a password: pass --redis-password or set $%s", passwordEnv)
	}

	body["redisPasswordEnabled"] = enabled
	if password != "" {
		if err := vdbclient.ValidateRedisPassword(password, "redis-password"); err != nil {
			return err
		}
		body["redisPassword"] = password
	}
	return nil
}

// applyBackupFlags adds the three backup fields, which the API wants together.
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

func toInterfaces(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
