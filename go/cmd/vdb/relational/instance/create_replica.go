package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var createReplicaCmd = &cobra.Command{
	Use:   "create-replica",
	Short: "Create a read replica of a Relational Database instance",
	Long: "Create a read replica from an existing instance.\n\n" +
		"THIS COSTS MONEY: a replica is a full instance and goes through the same " +
		"order/payment flow as 'instance create', asynchronously. Use --dry-run first.\n\n" +
		"Every property of the replica defaults to the SOURCE instance's — engine, " +
		"version, flavor, volume type and size, zone, subnet, public access and backup " +
		"settings are read from it, so normally only --instance-id and --name are needed. " +
		"Pass any of the other flags to override one; --dry-run shows the resolved " +
		"request. A replica must run the same engine and version as its source, so those " +
		"two are deliberately not overridable.",
	Args: cobra.NoArgs,
	RunE: runCreateReplica,
}

func createReplicaFlags(f *pflag.FlagSet) {
	f.String("instance-id", "", "ID of the source instance to replicate (required)")
	f.String("name", "", "Name of the new replica (required)")
	f.String("package-id", "", "Flavor id for the replica (default: same as the source)")
	f.String("volume-type", "", "Volume type NAME for the replica (default: same as the source)")
	f.Int("volume-size", 0, "Volume size in GB (default: same as the source)")
	f.String("zone-id", "", "Availability zone to place the replica in (default: same as the source)")
	f.String("subnet-ids", "", "Subnet ID(s), comma-separated (default: the source's subnet)")
	f.String("config-id", "", "Config group ID to attach (default: the source's, if any)")
	f.Bool("public-access", false, "Allow public access to the replica (default: same as the source)")
	f.Bool("backup-auto", false, "Enable daily automatic backup (default: same as the source)")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	createReplicaFlags(createReplicaCmd.Flags())

	createReplicaCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	createReplicaCmd.MarkFlagRequired("name")        //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	createReplicaCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("volume-type", volumeTypeCompletion())      //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())              //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("subnet-ids", subnetCompletion())           //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())          //nolint:errcheck
}

func runCreateReplica(cmd *cobra.Command, args []string) error {
	sourceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(sourceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	source, err := fetchInstance(apiClient, sourceID)
	if err != nil {
		return err
	}

	body, err := replicaBody(cmd, sourceID, source)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("replica %q of %s", name, sourceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Create replica %q of %s (%v GB, flavor %v)? This places a paid order.",
		name, sourceID, body["volumeSize"], body["packageId"])) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(instancePath(sourceID, "/create-replicas"), body)
	if err != nil {
		return fmt.Errorf("failed to create replica %q of database instance %s: %w", name, sourceID, err)
	}

	return vdbclient.Output(cmd, result)
}

// replicaBody assembles CreateDbInstanceReplicaRequest, defaulting every unset
// field to the source instance. Split out from runCreateReplica so the defaulting
// rules are testable against a captured instance payload.
func replicaBody(cmd *cobra.Command, sourceID string, source map[string]interface{}) (map[string]interface{}, error) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	poc, _ := flags.GetBool("poc")

	body := map[string]interface{}{
		"name":            name,
		"replicaSourceId": sourceID,
		"poc":             poc,
		// A replica must match its source's engine, so these are taken from it and
		// have no flags.
		"datastoreType":    stringField(source, "datastoreType"),
		"datastoreVersion": stringField(source, "datastoreVersion"),
		"packageId":        stringOrDefault(flags, "package-id", sourcePackageID(source)),
		"volumeType":       stringOrDefault(flags, "volume-type", stringField(source, "volumeType")),
		"volumeSize":       intOrDefault(flags, "volume-size", intField(source, "volumeSize")),
		"locateZoneId":     stringOrDefault(flags, "zone-id", stringField(source, "zoneId")),
		"publicAccess":     boolOrDefault(flags, "public-access", boolField(source, "publicAccess")),
	}

	subnetIDs, _ := flags.GetString("subnet-ids")
	subnets := cli.ParseCommaSeparated(subnetIDs)
	if len(subnets) == 0 {
		if sourceSubnet := stringField(source, "subnetId"); sourceSubnet != "" {
			subnets = []string{sourceSubnet}
		}
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("could not determine a subnet: the source instance reports none, so pass --subnet-ids")
	}
	body["netIds"] = toInterfaces(subnets)

	if configID := stringOrDefault(flags, "config-id", sourceConfigID(source)); configID != "" {
		body["configId"] = configID
	}

	if err := applyReplicaBackupFlags(flags, source, body); err != nil {
		return nil, err
	}

	if body["volumeSize"].(int) <= 0 {
		return nil, fmt.Errorf("could not determine a volume size from the source instance: pass --volume-size")
	}
	if body["packageId"] == "" {
		return nil, fmt.Errorf("could not determine a flavor from the source instance: pass --package-id")
	}

	return body, nil
}

// applyReplicaBackupFlags mirrors the source's backup settings unless the user
// changes them. When backup is on, the API demands duration and time together, so
// the defaults come from the source as a set.
func applyReplicaBackupFlags(flags *pflag.FlagSet, source, body map[string]interface{}) error {
	backupAuto := boolOrDefault(flags, "backup-auto", boolField(source, "backupAuto"))
	body["backupAuto"] = backupAuto

	if !backupAuto {
		return nil
	}

	duration := intOrDefault(flags, "backup-duration", intField(source, "backupDuration"))
	backupTime := stringOrDefault(flags, "backup-time", stringField(source, "backupTime"))

	if duration < minBackupDuration || duration > maxBackupDuration {
		return fmt.Errorf("backup retention must be between %d and %d days, got %d: pass --backup-duration",
			minBackupDuration, maxBackupDuration, duration)
	}
	if backupTime == "" {
		return fmt.Errorf("automatic backup needs a time of day: pass --backup-time, e.g. --backup-time 02:00")
	}

	body["backupDuration"] = duration
	body["backupTime"] = backupTime
	return nil
}

// sourcePackageID is the source's flavor. The instance payload calls it
// quotaPackageId while the create request calls it packageId.
func sourcePackageID(source map[string]interface{}) string {
	return stringField(source, "quotaPackageId")
}

// sourceConfigID reads the attached config group. The instance payload leaves the
// flat configId null even when one is attached and reports it in the nested
// configuration object instead (verified live), so check both.
func sourceConfigID(source map[string]interface{}) string {
	if configID := stringField(source, "configId"); configID != "" {
		return configID
	}
	if configuration, ok := source["configuration"].(map[string]interface{}); ok {
		return stringField(configuration, "id")
	}
	return ""
}

// stringOrDefault and friends return the flag when the user set it, otherwise the
// fallback. Changed() rather than emptiness, so "--public-access=false" is a
// deliberate override rather than an unset flag.
func stringOrDefault(flags *pflag.FlagSet, name, fallback string) string {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetString(name)
	return value
}

func intOrDefault(flags *pflag.FlagSet, name string, fallback int) int {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetInt(name)
	return value
}

func boolOrDefault(flags *pflag.FlagSet, name string, fallback bool) bool {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetBool(name)
	return value
}
