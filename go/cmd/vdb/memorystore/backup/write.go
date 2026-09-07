package backup

import (
	"fmt"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BackupResourceKey is the completion key for a MemoryStore backup ID.
const BackupResourceKey = "vdb:memorystore-backup"

// passwordEnv is the same variable the other groups use for a master password.
const passwordEnv = "GRN_VDB_MASTER_PASSWORD"

var backupTypes = []string{"FULL", "INCREMENTAL"}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Take a backup of a MemoryStore instance now",
	Long: "Create an on-demand backup, in addition to whatever the instance's schedule " +
		"produces.\n\n" +
		"The backup counts towards your backup storage, which is billable beyond the free " +
		"allowance, so the command confirms first.\n\n" +
		"--description is REQUIRED: on the Relational Database API a backup without one is " +
		"accepted and then fails in the background (measured), and the two share this " +
		"request schema — so the CLI requires it here too rather than risk the same silent " +
		"failure.\n\n" +
		"--backup-type INCREMENTAL requires --parent-id, the backup to build on.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a backup",
	Long: "Delete one backup.\n\n" +
		"Irreversible, and it can break a chain: an INCREMENTAL backup builds on its " +
		"parent, so deleting a parent may leave its children unrestorable.\n\n" +
		"Note the request shape: this is a POST to /backups/delete with a JSON ARRAY body " +
		"and no ID in the path at all — the relational API uses DELETE with the ID in the " +
		"path instead.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Create a new MemoryStore instance from a backup",
	Long: "Restore a backup.\n\n" +
		"**This does not restore in place.** The API builds a NEW instance from the backup " +
		"and leaves the original untouched — so a restore is a create: order/payment flow, " +
		"real cost, asynchronous, and it needs a name.\n\n" +
		"You choose where it lands: --zone-id and --subnet-ids are required, because a " +
		"restore is a placement decision. Everything else defaults to the backup's record " +
		"(engine, version, flavor, config group). A master password is needed if the new " +
		"instance should require one.",
	Args: cobra.NoArgs,
	RunE: runRestore,
}

func createFlags(f *pflag.FlagSet) {
	f.String("instance-id", "", "Instance to back up (required)")
	f.String("name", "", "Backup name (required)")
	f.String("description", "", "Free-text description (required — see the note in --help)")
	f.String("backup-type", "FULL", fmt.Sprintf("Backup type: %s", strings.Join(backupTypes, " or ")))
	f.String("parent-id", "", "Parent backup ID (required with --backup-type INCREMENTAL)")
	f.Bool("dry-run", false, "Print the request that would be sent without taking a backup")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func restoreFlags(f *pflag.FlagSet) {
	f.String("backup-id", "", "Backup to restore (required)")
	f.String("name", "", "Name of the new instance (required)")
	f.String("zone-id", "", "Availability zone for the new instance (required)")
	f.String("subnet-ids", "", "Subnet ID(s) for the new instance, comma-separated (required)")
	f.String("package-id", "", "Flavor id (default: the backup's)")
	f.String("config-id", "", "Config group to attach (default: the backup's, if any)")
	f.String("redis-password", "", "Master password for the new instance (defaults to $"+passwordEnv+")")
	f.Bool("public-access", false, "Allow public access — requires a master password")
	f.Bool("backup-auto", false, "Enable daily automatic backup on the new instance")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d (with --backup-auto)", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM (with --backup-auto)")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	createFlags(createCmd.Flags())
	for _, required := range []string{"instance-id", "name", "description"} {
		createCmd.MarkFlagRequired(required) //nolint:errcheck
	}
	createCmd.RegisterFlagCompletionFunc("instance-id", cli.ResourceCompletion(instanceResourceKey)) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("backup-type", cli.FlagValues(backupTypes...))              //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("parent-id", cli.ResourceCompletion(BackupResourceKey))     //nolint:errcheck

	d := deleteCmd.Flags()
	d.String("backup-id", "", "Backup ID (required)")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteCmd.MarkFlagRequired("backup-id")                                                      //nolint:errcheck
	deleteCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey)) //nolint:errcheck

	restoreFlags(restoreCmd.Flags())
	for _, required := range []string{"backup-id", "name", "zone-id", "subnet-ids"} {
		restoreCmd.MarkFlagRequired(required) //nolint:errcheck
	}
	restoreCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey))   //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("zone-id", cli.ResourceCompletion(zoneResourceKey))       //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("subnet-ids", cli.ResourceCompletion(subnetResourceKey))  //nolint:errcheck
	restoreCmd.RegisterFlagCompletionFunc("config-id", cli.ResourceCompletion(configGroupResource)) //nolint:errcheck
}

func runCreate(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	body, err := createBody(cmd.Flags(), instanceID)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("create", fmt.Sprintf("backup %q of %s", name, instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Take a %v backup %q of MemoryStore instance %s? It counts towards billable backup storage.",
		body["backupType"], name, instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(basePath+"/create", body)
	if err != nil {
		return fmt.Errorf("failed to create backup %q of MemoryStore instance %s: %w", name, instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

// createBody assembles CreateBackupRequest — the same schema the relational API uses,
// including the description rule enforced there.
func createBody(flags *pflag.FlagSet, instanceID string) (map[string]interface{}, error) {
	name, _ := flags.GetString("name")
	description, _ := flags.GetString("description")
	backupType, _ := flags.GetString("backup-type")
	parentID, _ := flags.GetString("parent-id")

	backupType = strings.ToUpper(backupType)
	if backupType != "FULL" && backupType != "INCREMENTAL" {
		return nil, fmt.Errorf("invalid --backup-type %q: must be %s", backupType, strings.Join(backupTypes, " or "))
	}
	if backupType == "INCREMENTAL" && parentID == "" {
		return nil, fmt.Errorf("--backup-type INCREMENTAL requires --parent-id, the backup to build on")
	}
	if parentID != "" {
		if err := requireBackupID(parentID); err != nil {
			return nil, err
		}
		if backupType != "INCREMENTAL" {
			return nil, fmt.Errorf("--parent-id only applies to --backup-type INCREMENTAL")
		}
	}
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("--description must not be empty: on the relational API, which shares " +
			"this request schema, a backup without one is accepted and then fails in the background")
	}

	body := map[string]interface{}{
		"dbInstanceId": instanceID,
		"name":         name,
		"backupType":   backupType,
		"description":  description,
	}
	if parentID != "" {
		body["parentId"] = parentID
	}
	return body, nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	backupID, _ := cmd.Flags().GetString("backup-id")
	if err := requireBackupID(backupID); err != nil {
		return err
	}

	// The whole request is the array; the path carries no ID.
	body := []interface{}{
		map[string]interface{}{"backupId": backupID},
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	printDeleteTarget(apiClient, backupID)

	if dryRun {
		vdbclient.PreviewBody("delete", fmt.Sprintf("backup %s", backupID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Delete backup %s? This cannot be undone.", backupID)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(basePath+"/delete", body)
	if err != nil {
		return fmt.Errorf("failed to delete backup %s: %w", backupID, err)
	}

	return vdbclient.Output(cmd, result)
}

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
		"Restore backup %s into a NEW MemoryStore instance %q? This places a paid order; the original instance is not touched.",
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

// restoreBody assembles RestoreMemBackupRequest: the same envelope as the relational
// restore — action "restore_backup", resourceType "dbaas-backup", a detail carrying
// only a config — with the Redis password fields in place of user/databases and no
// volume fields.
func restoreBody(flags *pflag.FlagSet, backupID string, backup map[string]interface{}) (map[string]interface{}, error) {
	name, _ := flags.GetString("name")
	poc, _ := flags.GetBool("poc")
	publicAccess, _ := flags.GetBool("public-access")

	zoneID, _ := flags.GetString("zone-id")
	if zoneID == "" {
		return nil, fmt.Errorf("--zone-id must not be empty: a restored instance needs a zone to be placed in")
	}

	// The relational restore rejects a request whose flavor and volume type do not
	// match the zone, blaming those fields rather than the zone. Sending the zone
	// explicitly is what avoids that class of error.
	config := map[string]interface{}{
		"backupId":         backupID,
		"name":             name,
		"poc":              poc,
		"locateZoneId":     zoneID,
		"publicAccess":     publicAccess,
		"datastoreType":    stringField(backup, "datastoreType"),
		"datastoreVersion": stringField(backup, "datastoreVersion"),
		"packageId":        stringOrDefault(flags, "package-id", stringField(backup, "packageId")),
	}

	subnets := cli.ParseCommaSeparated(mustString(flags, "subnet-ids"))
	if len(subnets) == 0 {
		return nil, fmt.Errorf("--subnet-ids must not be empty: pass a subnet in %s", zoneID)
	}
	config["netIds"] = toInterfaces(subnets)

	if configID := stringOrDefault(flags, "config-id", stringField(backup, "configId")); configID != "" {
		config["configId"] = configID
	}

	password, _ := flags.GetString("redis-password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if publicAccess && password == "" {
		return nil, fmt.Errorf("--public-access requires a master password: pass --redis-password or set $%s", passwordEnv)
	}
	config["redisPasswordEnabled"] = password != ""
	if password != "" {
		if err := vdbclient.ValidateRedisPassword(password, "redis-password"); err != nil {
			return nil, err
		}
		config["redisPassword"] = password
	}

	if err := applyBackupSchedule(flags, config); err != nil {
		return nil, err
	}
	if config["packageId"] == "" {
		return nil, fmt.Errorf("could not determine a flavor from backup %s: pass --package-id", backupID)
	}

	return vdbclient.ResizeConfigBody(vdbclient.ResourceTypeBackup, "restore_backup", config), nil
}

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

func fetchBackup(apiClient *vdbclient.Client, backupID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(basePath+"/"+backupID+"/detail", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup %s: %w", backupID, err)
	}
	backup, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("backup %s not found or unusable (the API returned an empty payload; "+
			"a backup whose creation failed reads this way)", backupID)
	}
	return backup, nil
}

func printDeleteTarget(apiClient *vdbclient.Client, backupID string) {
	result, err := apiClient.Get(basePath+"/"+backupID+"/detail", nil)
	if err != nil {
		fmt.Printf("Warning: could not read backup %s before deleting (%v).\n", backupID, err)
		return
	}
	backup, ok := vdbclient.PayloadObject(result)
	if !ok {
		fmt.Printf("Warning: backup %s returned an empty payload — it may already be gone.\n", backupID)
		return
	}

	fmt.Println("The following backup will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:       %v\n", backup["id"])
	fmt.Printf("  Name:     %v\n", backup["name"])
	fmt.Printf("  Type:     %v\n", backup["backupType"])
	fmt.Printf("  Instance: %v (%v)\n", backup["instanceName"], backup["dbInstanceId"])
	fmt.Printf("  Created:  %v\n", backup["created"])
	fmt.Println()
}

// Payload readers and flag defaulting, as in the other groups.

func stringField(payload map[string]interface{}, key string) string {
	value, _ := payload[key].(string)
	return value
}

func stringOrDefault(flags *pflag.FlagSet, name, fallback string) string {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetString(name)
	return value
}

func mustString(flags *pflag.FlagSet, name string) string {
	value, _ := flags.GetString(name)
	return value
}

func toInterfaces(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
