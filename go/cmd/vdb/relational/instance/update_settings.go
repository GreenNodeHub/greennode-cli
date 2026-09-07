package instance

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateSettingsCmd = &cobra.Command{
	Use:   "update-settings",
	Short: "Update the password, public access or backup schedule of an instance",
	Long: "Change an instance's master password, its public-access setting or its " +
		"automatic-backup schedule.\n\n" +
		"Only the settings you pass change, so updating the password cannot silently " +
		"flip network exposure or turn backups off. The one exception is the backup " +
		"schedule, which the API rejects a request without (HTTP 500), so the instance's " +
		"current schedule is read and repeated for you — which also means --backup-time " +
		"and --backup-duration can be changed on their own while automatic backup is on. " +
		"The new password is read from $" + passwordEnv + " when --password is omitted, and " +
		"is masked in --dry-run output.\n\n" +
		"The change is ASYNCHRONOUS: the API accepts the request and applies it in the " +
		"background, and its response body carries nothing useful, so this command " +
		"reports acceptance and points you at 'instance get' to confirm the result.",
	Args: cobra.NoArgs,
	RunE: runUpdateSettings,
}

func settingsFlags(f *pflag.FlagSet) {
	f.String("instance-id", "", "Database instance ID (required)")
	f.String("password", "", "New master password (defaults to $"+passwordEnv+")")
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

	// Bound here, next to the flag: see the init-order note in completion.go.
	updateSettingsCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
}

func runUpdateSettings(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Read up front, even for --dry-run: the request needs the instance's current backup
	// schedule (see settingsBody).
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
		previewAsyncUpdate("update", fmt.Sprintf("the settings of database instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Update %s on database instance %s?", summary, instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	warnIfNotActive(apiClient, instanceID)

	if _, err := apiClient.Put(instancePath(instanceID, "/update/setting"), body); err != nil {
		return fmt.Errorf("failed to update the settings of database instance %s: %w", instanceID, err)
	}

	reportAsyncAccepted(cmd, instanceID, summary)
	return nil
}

// settingsBody assembles UpdateDbSettingRequest from the flags the user actually
// set, and describes the change for the prompt.
//
// Sending only changed fields matters here: publicAccess and backupAuto are
// booleans whose zero value is a real, destructive setting. A body that always
// carried "publicAccess": false would cut off public connectivity every time
// someone rotated a password, and "backupAuto": false would silently disable
// backups.
//
// backupAuto is the exception, and not by choice: a body without it is answered with
// HTTP 500 (verified live on 2026-08-13 against both this API and its MemoryStore twin,
// with a body that only repeated the instance's own publicAccess). So the schedule is
// taken from `current` and repeated whenever the user names none of the backup flags,
// which keeps the call a no-op for backups while satisfying the API.
func settingsBody(flags *pflag.FlagSet, instanceID string, current map[string]interface{}) (map[string]interface{}, string, error) {
	// dbInstanceId is required in the body as well as the path.
	body := map[string]interface{}{"dbInstanceId": instanceID}
	var changes []string

	password, _ := flags.GetString("password")
	if password == "" {
		password = os.Getenv(passwordEnv)
	}
	if password != "" {
		if err := vdbclient.ValidateDBPassword(password, "password"); err != nil {
			return nil, "", err
		}
		body["password"] = password
		changes = append(changes, "the master password")
	}

	if flags.Changed("public-access") {
		publicAccess, _ := flags.GetBool("public-access")
		body["publicAccess"] = publicAccess
		changes = append(changes, fmt.Sprintf("public access (-> %t)", publicAccess))
	}

	// Start from the instance's own schedule and overwrite only what the user named.
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
		// Name the schedule in the summary: "automatic backup (-> true)" alone hides the
		// time and retention just set, and the response carries nothing to check.
		if backupAuto {
			changes = append(changes, fmt.Sprintf("automatic backup (-> on, %s, %d days)", backupTime, duration))
		} else {
			changes = append(changes, "automatic backup (-> off)")
		}
	}

	// The body always carries the schedule now, so emptiness is judged by what the user
	// actually asked to change.
	if len(changes) == 0 {
		return nil, "", fmt.Errorf("nothing to update: pass --password (or set $%s), --public-access or --backup-auto",
			passwordEnv)
	}

	summary := changes[0]
	for _, change := range changes[1:] {
		summary += " and " + change
	}
	return body, summary, nil
}
