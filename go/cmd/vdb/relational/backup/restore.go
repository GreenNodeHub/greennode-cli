package backup

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// passwordEnv keeps the master password out of shell history. A restore may need
// one, since it builds a new instance.
const passwordEnv = "GRN_VDB_MASTER_PASSWORD"

const (
	minBackupDuration = 2
	maxBackupDuration = 14
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Create a new Relational Database instance from a backup",
	Long: "Restore a backup.\n\n" +
		"**This does not restore in place.** The API builds a NEW instance from the " +
		"backup and leaves the original untouched — so a restore is a create: it goes " +
		"through the order/payment flow, costs money, completes asynchronously, and " +
		"needs a name for the new instance.\n\n" +
		"You choose where it lands: --zone-id and --subnet-ids are required, because a " +
		"restore is a placement decision — the new instance need not sit where the " +
		"original did.\n\n" +
		"Everything else defaults to the backup's own record: engine, version, flavor, " +
		"storage type and size, and config group are read from it. Any of those flags " +
		"overrides one; --dry-run shows the resolved request.\n\n" +
		"IMPORTANT when restoring into a DIFFERENT zone from the original: flavor ids and " +
		"volume type names are zone-specific (id 180 and " +
		"'Gen2-NVMe2-IOPS3000-HCM03-1B' exist only in HCM03-1B), so pass --package-id and " +
		"--volume-type from that zone's 'grn vdb relational catalog' too. Without a " +
		"matching zone the API rejects the request with 'Package ID … is invalid; Volume " +
		"type … is invalid'.",
	Args: cobra.NoArgs,
	RunE: runRestore,
}

func restoreFlags(f *pflag.FlagSet) {
	f.String("backup-id", "", "Backup to restore (required)")
	f.String("name", "", "Name of the new instance (required)")
	f.String("package-id", "", "Flavor id for the new instance (default: the backup's)")
	f.String("volume-type", "", "Volume type NAME (default: the backup's)")
	f.Int("volume-size", 0, "Volume size in GB (default: the backup's)")
	f.String("zone-id", "", "Availability zone for the new instance (required)")
	f.String("subnet-ids", "", "Subnet ID(s) for the new instance, comma-separated (required)")
	f.String("config-id", "", "Config group to attach (default: the backup's, if any)")
	f.String("password", "", "Master password for the new instance (defaults to $"+passwordEnv+")")
	f.Bool("public-access", false, "Allow public access to the new instance")
	f.Bool("backup-auto", false, "Enable daily automatic backup on the new instance")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d (with --backup-auto)", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM (with --backup-auto)")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	restoreFlags(restoreCmd.Flags())

	restoreCmd.MarkFlagRequired("backup-id")  //nolint:errcheck
	restoreCmd.MarkFlagRequired("name")       //nolint:errcheck
	restoreCmd.MarkFlagRequired("zone-id")    //nolint:errcheck
	restoreCmd.MarkFlagRequired("subnet-ids") //nolint:errcheck

	restoreCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey))       //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("volume-type", cli.ResourceCompletion(volumeTypeResourceKey)) //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("zone-id", cli.ResourceCompletion(zoneResourceKey))           //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("subnet-ids", cli.ResourceCompletion(subnetResourceKey))      //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResourceKey))  //nolint:errcheck
}

// Completion keys owned by the relational catalog package.
const (
	volumeTypeResourceKey  = "vdb:relational-volume-type"
	zoneResourceKey        = "vdb:relational-zone"
	subnetResourceKey      = "vdb:relational-subnet"
	configGroupResourceKey = "vdb:relational-config-group"
)

func runRestore(cmd *cobra.Command, args []string) error {
	backupID, _ := cmd.Flags().GetString("backup-id")
	if err := requireBackupID(backupID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	backup, err := fetchBackup(apiClient, backupID)
	if err != nil {
		return err
	}

	body, err := restoreBody(cmd.Flags(), backupID, backup)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("restore", fmt.Sprintf("backup %s into a new instance %q", backupID, name), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Restore backup %s into a NEW instance %q? This places a paid order; the original instance is not touched.",
		backupID, name)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(basePath+"/"+backupID+"/restore", body)
	if err != nil {
		return fmt.Errorf("failed to restore backup %s: %w", backupID, err)
	}

	return vdbclient.Output(cmd, result)
}

func fetchBackup(apiClient *vdbclient.Client, backupID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(basePath+"/detail/"+backupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup %s: %w", backupID, err)
	}
	backup, ok := vdbclient.PayloadObject(result)
	if !ok {
		// HTTP 200 with a null payload is how this API says "not found"; without this
		// check the restore would be built from an empty record.
		return nil, fmt.Errorf("backup %s not found or unusable (the API returned an empty payload; "+
			"a backup whose creation failed reads this way — check 'backup list' and 'instance list-histories')", backupID)
	}
	return backup, nil
}

// restoreBody assembles RestoreBackupRequest.
//
// The shape is unlike every other action request: the detail carries ONLY a config —
// no instancesId — because the thing being acted on is named inside it, and the rest
// of the config describes the instance to create. The action is "restore_backup" and
// the resource type "dbaas-backup", not "dbaas".
//
// Field names differ between a backup record and this request, which is the main
// reason the defaulting is centralised here:
//
//	backup.storageType -> volumeType     backup.storageSize -> volumeSize
//	backup.packageId   -> packageId      backup.netIds      -> netIds
func restoreBody(flags *pflag.FlagSet, backupID string, backup map[string]interface{}) (map[string]interface{}, error) {
	name, _ := flags.GetString("name")
	poc, _ := flags.GetBool("poc")

	config := map[string]interface{}{
		"backupId":         backupID,
		"name":             name,
		"poc":              poc,
		"datastoreType":    stringField(backup, "datastoreType"),
		"datastoreVersion": stringField(backup, "datastoreVersion"),
		"packageId":        stringOrDefault(flags, "package-id", stringField(backup, "packageId")),
		"volumeType":       stringOrDefault(flags, "volume-type", stringField(backup, "storageType")),
		"volumeSize":       intOrDefault(flags, "volume-size", intField(backup, "storageSize")),
		"publicAccess":     boolOrDefault(flags, "public-access", false),
	}

	// locateZoneId must be sent, though the spec marks it optional: without it the API
	// validates the flavor and volume type against some other zone and rejects both
	// with "Package ID … is invalid; Volume type … is invalid", even when they are
	// exactly what the source instance runs (verified live 2026-08-13). It is a required
	// flag rather than a value inherited from the backup, because where a restored
	// instance lands is the user's decision.
	zoneID, _ := flags.GetString("zone-id")
	if zoneID == "" {
		return nil, fmt.Errorf("--zone-id must not be empty: a restored instance needs a zone to be placed in")
	}
	config["locateZoneId"] = zoneID

	// The request wants SUBNET ids ("sub-…"), but a backup record's netIds field can
	// hold NETWORK ids ("net-…") — verified on a real backup, whose instance was
	// created with a subnet yet whose record reports the network. Defaulting blindly
	// would send a network id and fail, so only subnet-shaped values are reused.
	// Subnets come from the flag only. The backup's own netIds field is no help: it
	// records the NETWORK ("net-…") the original instance sat in, not a subnet, and a
	// subnet belongs to one zone anyway — which the user has just chosen.
	subnets := cli.ParseCommaSeparated(mustString(flags, "subnet-ids"))
	if len(subnets) == 0 {
		return nil, fmt.Errorf("--subnet-ids must not be empty: pass a subnet in %s", zoneID)
	}
	config["netIds"] = toInterfaces(subnets)

	if configID := stringOrDefault(flags, "config-id", stringField(backup, "configId")); configID != "" {
		config["configId"] = configID
	}

	// A restored instance is a new instance, so it needs master credentials. The
	// backup records the username; the password cannot be recovered from it.
	password, _ := flags.GetString("password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if password != "" {
		if err := vdbclient.ValidateDBPassword(password, "password"); err != nil {
			return nil, err
		}
		config["password"] = password
	}

	if err := applyBackupSchedule(flags, config); err != nil {
		return nil, err
	}

	if config["volumeSize"].(int) <= 0 {
		return nil, fmt.Errorf("could not determine a volume size from backup %s: pass --volume-size", backupID)
	}
	if config["packageId"] == "" {
		return nil, fmt.Errorf("could not determine a flavor from backup %s: pass --package-id", backupID)
	}

	return vdbclient.ResizeConfigBody(vdbclient.ResourceTypeBackup, "restore_backup", config), nil
}

// applyBackupSchedule mirrors the create rules: the three backup fields travel
// together or not at all.
func applyBackupSchedule(flags *pflag.FlagSet, config map[string]interface{}) error {
	backupAuto, _ := flags.GetBool("backup-auto")
	duration, _ := flags.GetInt("backup-duration")
	backupTime, _ := flags.GetString("backup-time")

	config["backupAuto"] = backupAuto

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

	config["backupDuration"] = duration
	config["backupTime"] = backupTime
	return nil
}
