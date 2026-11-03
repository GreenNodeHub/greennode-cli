package instance

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var createReplicaCmd = &cobra.Command{
	Use:   "create-replica",
	Short: "Create a read replica of a MemoryStore instance",
	Long: "Create a read replica from an existing instance.\n\n" +
		"THIS COSTS MONEY: a replica is a full instance and goes through the same " +
		"order/payment flow as 'instance create', asynchronously. Use --dry-run first.\n\n" +
		"Every property defaults to the SOURCE instance's — engine, version, flavor, " +
		"zone, subnet, public access and backup settings are read from it, so normally " +
		"only --instance-id and --name are needed. Any other flag overrides one; " +
		"--dry-run shows the resolved request. Engine and version are deliberately not " +
		"overridable: a replica must match its source.",
	Args: cobra.NoArgs,
	RunE: runCreateReplica,
}

func init() {
	replicaFlagsOn(createReplicaCmd.Flags())

	createReplicaCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	createReplicaCmd.MarkFlagRequired("name")        //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	createReplicaCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())         //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("subnet-ids", subnetCompletion())      //nolint:errcheck
	createReplicaCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())     //nolint:errcheck
}

func runCreateReplica(cmd *cobra.Command, args []string) error {
	sourceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(sourceID); err != nil {
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

	body, err := replicaBody(cmd.Flags(), sourceID, source)
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
		"Create replica %q of %s (flavor %v)? This places a paid order.",
		name, sourceID, body["packageId"])) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(instancePath(sourceID, "/create-replicas"), body)
	if err != nil {
		return fmt.Errorf("failed to create replica %q of MemoryStore instance %s: %w", name, sourceID, err)
	}

	return vdbclient.Output(cmd, result)
}

// replicaBody assembles CreateMemDbInstanceReplicaRequest, defaulting every unset
// field to the source. Like the relational version, the flavor is `quotaPackageId`
// on an instance but `packageId` in the request, and an attached config group is only
// reported in the nested `configuration.id`.
func replicaBody(flags *pflag.FlagSet, sourceID string, source map[string]interface{}) (map[string]interface{}, error) {
	name, _ := flags.GetString("name")
	poc, _ := flags.GetBool("poc")

	body := map[string]interface{}{
		"name":             name,
		"replicaSourceId":  sourceID,
		"poc":              poc,
		"datastoreType":    stringField(source, "datastoreType"),
		"datastoreVersion": stringField(source, "datastoreVersion"),
		"packageId":        stringOrDefault(flags, "package-id", stringField(source, "quotaPackageId")),
		"locateZoneId":     stringOrDefault(flags, "zone-id", stringField(source, "zoneId")),
		"publicAccess":     boolOrDefault(flags, "public-access", boolField(source, "publicAccess")),
	}

	subnets := cli.ParseCommaSeparated(mustString(flags, "subnet-ids"))
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

	// A replica is a new instance and needs its own password when the source has one
	// enabled, since the source's cannot be read back.
	if boolField(source, "redisPasswordEnabled") || flags.Changed("redis-password") {
		password, _ := flags.GetString("redis-password")
		if password == "" {
			password = os.Getenv(passwordEnv)
		}
		if password == "" {
			return nil, fmt.Errorf(
				"the source instance requires a master password, so the replica needs one too: pass --redis-password or set $%s",
				passwordEnv)
		}
		if err := vdbclient.ValidateRedisPassword(password, "redis-password"); err != nil {
			return nil, err
		}
		body["redisPasswordEnabled"] = true
		body["redisPassword"] = password
	} else {
		body["redisPasswordEnabled"] = false
	}

	if err := applyReplicaBackupFlags(flags, source, body); err != nil {
		return nil, err
	}

	if body["packageId"] == "" {
		return nil, fmt.Errorf("could not determine a flavor from the source instance: pass --package-id")
	}
	return body, nil
}

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

// sourceConfigID reads the attached config group. The flat configId is null even
// when one is attached; the nested configuration object is where it appears.
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
// fallback. Changed() rather than emptiness, so an explicit false is an override.

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

func mustString(flags *pflag.FlagSet, name string) string {
	value, _ := flags.GetString(name)
	return value
}
