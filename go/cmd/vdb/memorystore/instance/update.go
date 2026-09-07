package instance

import (
	"fmt"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateSettingsCmd = &cobra.Command{
	Use:   "update-settings",
	Short: "Update the master password, public access or backup schedule of an instance",
	Long: "Change a MemoryStore instance's Redis master password, its public-access " +
		"setting or its automatic-backup schedule.\n\n" +
		"Only the settings you pass change; the one exception is the backup schedule, " +
		"which the API rejects a request without (HTTP 500), so the instance's current " +
		"schedule is read and repeated for you. The password is read from $" + passwordEnv +
		" when --redis-password is omitted, and is masked in --dry-run output.\n\n" +
		"--backup-time and --backup-duration can therefore be changed on their own, " +
		"as long as automatic backup is on.\n\n" +
		"Two API rules are enforced here rather than left to a 400: public access " +
		"requires the master password to be enabled, and any change to the password or to " +
		"whether it is required must travel with editRedisPassword — a flag the CLI sets " +
		"for you.\n\n" +
		"The change is ASYNCHRONOUS and its response body carries nothing useful, so this " +
		"command reports acceptance and points you at 'instance get'.",
	Args: cobra.NoArgs,
	RunE: runUpdateSettings,
}

var updateConfigGroupCmd = &cobra.Command{
	Use:   "update-config-group",
	Short: "Attach or detach the config group of a MemoryStore instance",
	Long: "Attach a config group to an instance, or detach the current one with " +
		"--detach.\n\n" +
		"The group must match the instance's engine and version — see " +
		"'catalog list-config-groups'. Applying parameters can restart Redis, so the " +
		"command confirms first. Asynchronous, like update-settings.",
	Args: cobra.NoArgs,
	RunE: runUpdateConfigGroup,
}

func settingsFlags(f *pflag.FlagSet) {
	f.String("instance-id", "", "MemoryStore instance ID (required)")
	f.String("redis-password", "", "New Redis master password (defaults to $"+passwordEnv+")")
	f.Bool("redis-password-enabled", false, "Whether a master password is required to connect")
	f.Bool("public-access", false, "Whether the instance is reachable publicly")
	f.Bool("backup-auto", false, "Whether daily automatic backup is enabled")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d (with --backup-auto)", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM (with --backup-auto)")
	f.Bool("dry-run", false, "Print the request that would be sent without applying it")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	settingsFlags(updateSettingsCmd.Flags())
	updateSettingsCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	c := updateConfigGroupCmd.Flags()
	c.String("instance-id", "", "MemoryStore instance ID (required)")
	c.String("config-id", "", "Config group ID to attach (required unless --detach)")
	c.Bool("detach", false, "Detach the current config group instead of attaching one")
	c.Bool("dry-run", false, "Print the request that would be sent without applying it")
	c.Bool("force", false, "Skip the confirmation prompt")
	updateConfigGroupCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	updateConfigGroupCmd.MarkFlagsMutuallyExclusive("config-id", "detach")

	// Bound here, next to the flags: see the init-order note in completion.go.
	updateSettingsCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion())    //nolint:errcheck
	updateConfigGroupCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
	updateConfigGroupCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())     //nolint:errcheck
}

func runUpdateSettings(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// The instance is read up front, even for --dry-run: the request needs its current
	// backup schedule (see settingsBody).
	current, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		return err
	}

	body, summary, err := settingsBody(cmd.Flags(), instanceID, current)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewAsyncUpdate("update", fmt.Sprintf("the settings of MemoryStore instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Update %s on MemoryStore instance %s?", summary, instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	warnIfNotActive(apiClient, instanceID)

	// Hyphenated here; the relational API nests the same operation under /update/.
	if _, err := apiClient.Put(instancePath(instanceID, "/update-setting"), body); err != nil {
		return fmt.Errorf("failed to update the settings of MemoryStore instance %s: %w", instanceID, err)
	}

	reportAsyncAccepted(instanceID, summary)
	return nil
}

// settingsBody assembles UpdateMemDbSettingRequest from the flags the user set.
//
// Two Redis-specific rules live here:
//
//   - editRedisPassword must be true whenever redisPasswordEnabled or redisPassword
//     changes. It is a "I meant to touch the password" marker, not a setting, so the
//     CLI sets it rather than exposing it.
//   - publicAccess requires the password to be enabled, so asking for public access
//     without one is refused before the request goes out.
//
// The backup schedule is always sent, taken from `current` when the user names none of
// the backup flags: a body without backupAuto is answered with HTTP 500 (verified live
// on 2026-08-13 — password-only and password+publicAccess bodies both failed, the same
// bodies with backupAuto succeeded). Repeating the instance's own schedule keeps the
// call a no-op for backups while satisfying the API.
func settingsBody(flags *pflag.FlagSet, instanceID string, current map[string]interface{}) (map[string]interface{}, string, error) {
	body := map[string]interface{}{"dbInstanceId": instanceID}
	var changes []string

	password, _ := flags.GetString("redis-password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	passwordEnabledChanged := flags.Changed("redis-password-enabled")
	passwordEnabled, _ := flags.GetBool("redis-password-enabled")

	publicAccessChanged := flags.Changed("public-access")
	publicAccess, _ := flags.GetBool("public-access")

	// Two implications, both so the body cannot contradict itself:
	//   - giving a password means "require it" (otherwise we would send a new password
	//     alongside redisPasswordEnabled false);
	//   - turning public access on means "require it" too, which the API demands.
	// Either can still be overridden by passing --redis-password-enabled explicitly.
	if !passwordEnabledChanged && (password != "" || (publicAccessChanged && publicAccess)) {
		passwordEnabled = true
		passwordEnabledChanged = true
	}

	if password != "" || passwordEnabledChanged {
		if passwordEnabled && password == "" {
			return nil, "", fmt.Errorf(
				"enabling the master password needs one: pass --redis-password or set $%s", passwordEnv)
		}
		if publicAccessChanged && publicAccess && !passwordEnabled {
			return nil, "", fmt.Errorf("--public-access requires the master password to stay enabled")
		}

		// The marker the API demands alongside any password change.
		body["editRedisPassword"] = true
		body["redisPasswordEnabled"] = passwordEnabled
		if password != "" {
			if err := vdbclient.ValidateRedisPassword(password, "redis-password"); err != nil {
				return nil, "", err
			}
			body["redisPassword"] = password
		}

		if password != "" {
			changes = append(changes, "the master password")
		} else {
			changes = append(changes, fmt.Sprintf("password requirement (-> %t)", passwordEnabled))
		}
	}

	if publicAccessChanged {
		body["publicAccess"] = publicAccess
		changes = append(changes, fmt.Sprintf("public access (-> %t)", publicAccess))
	}

	// Start from the instance's own schedule and overwrite only what the user named, so
	// backupAuto is always present without changing anything the user did not ask to
	// change.
	backupAuto := boolField(current, "backupAuto")
	backupTime := stringField(current, "backupTime")
	duration := intField(current, "backupDuration")
	scheduleChanged := false

	if flags.Changed("backup-auto") {
		backupAuto, _ = flags.GetBool("backup-auto")
		scheduleChanged = true
	}
	if flags.Changed("backup-time") {
		backupTime, _ = flags.GetString("backup-time")
		scheduleChanged = true
	}
	if flags.Changed("backup-duration") {
		duration, _ = flags.GetInt("backup-duration")
		scheduleChanged = true
	}

	if !backupAuto && (flags.Changed("backup-time") || flags.Changed("backup-duration")) {
		return nil, "", fmt.Errorf("--backup-time and --backup-duration apply only while automatic backup is on: add --backup-auto")
	}

	body["backupAuto"] = backupAuto
	if backupAuto {
		if duration < minBackupDuration || duration > maxBackupDuration {
			return nil, "", fmt.Errorf("--backup-duration must be between %d and %d days, got %d",
				minBackupDuration, maxBackupDuration, duration)
		}
		if backupTime == "" {
			return nil, "", fmt.Errorf("automatic backup needs a time of day: pass --backup-time, e.g. --backup-time 02:00")
		}
		body["backupDuration"] = duration
		body["backupTime"] = backupTime
	}

	if scheduleChanged {
		// Name the schedule: "automatic backup (-> true)" alone hides the time and
		// retention just set, and the response body carries nothing to check it against.
		if backupAuto {
			changes = append(changes, fmt.Sprintf("automatic backup (-> on, %s, %d days)", backupTime, duration))
		} else {
			changes = append(changes, "automatic backup (-> off)")
		}
	}

	if len(changes) == 0 {
		return nil, "", fmt.Errorf(
			"nothing to update: pass --redis-password (or set $%s), --redis-password-enabled, --public-access or --backup-auto",
			passwordEnv)
	}

	summary := changes[0]
	for _, change := range changes[1:] {
		summary += " and " + change
	}
	return body, summary, nil
}

func runUpdateConfigGroup(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	configID, _ := cmd.Flags().GetString("config-id")
	detach, _ := cmd.Flags().GetBool("detach")

	if !detach && configID == "" {
		return fmt.Errorf("pass --config-id to attach a config group, or --detach to remove the current one")
	}

	// Detach sends null, not "". The relational endpoint rejects the empty string
	// with "The configId  doesn't exist" (verified live) while null works; MemoryStore
	// shares the request schema, so it gets the same treatment — and unlike the
	// PostgreSQL Cluster endpoint, which does accept "".
	body := map[string]interface{}{"dbInstanceId": instanceID}
	if detach {
		body["configId"] = nil
	} else {
		body["configId"] = configID
	}

	summary := fmt.Sprintf("attach config group %s", configID)
	if detach {
		summary = "detach the config group"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewAsyncUpdate("update", fmt.Sprintf("the config group of MemoryStore instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("%s on MemoryStore instance %s? This may restart Redis.",
		capitalize(summary), instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	warnIfNotActive(apiClient, instanceID)

	if _, err := apiClient.Put(instancePath(instanceID, "/update-config-group"), body); err != nil {
		return fmt.Errorf("failed to update the config group of MemoryStore instance %s: %w", instanceID, err)
	}

	reportAsyncAccepted(instanceID, summary)
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// The two update endpoints are asynchronous and their response payload is
// meaningless — see the relational equivalents in
// cmd/vdb/relational/instance/async.go for the measurements behind this.

func previewAsyncUpdate(verb, target string, body map[string]interface{}) {
	vdbclient.PreviewBody(verb, target, body)
	fmt.Println("\nThe API applies this in the background; poll 'instance get' to confirm the result.")
}

func reportAsyncAccepted(instanceID, summary string) {
	fmt.Printf("Accepted: %s on MemoryStore instance %s.\n", summary, instanceID)
	fmt.Println("The API applies this asynchronously — run the following to confirm:")
	fmt.Printf("\n  grn vdb memorystore instance get --instance-id %s\n", instanceID)
}

// warnIfNotActive mirrors the relational guard: while an instance is
// RESTART_REQUIRED these endpoints answer 202 and silently drop the change.
func warnIfNotActive(apiClient *vdbclient.Client, instanceID string) {
	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		return
	}
	status := stringField(instance, "status")
	if strings.EqualFold(status, "ACTIVE") {
		return
	}

	fmt.Printf("Warning: MemoryStore instance %s is %s, not ACTIVE.\n", instanceID, status)
	fmt.Println("The API may accept this change and silently not apply it — this is known to")
	fmt.Println("happen while an instance is RESTART_REQUIRED. Consider 'instance reboot' first.")
	fmt.Println()
}
