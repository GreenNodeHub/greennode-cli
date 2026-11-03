package backup

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Backup types the API documents. INCREMENTAL needs a parent backup to build on.
var backupTypes = []string{"FULL", "INCREMENTAL"}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Take a backup of a Relational Database instance now",
	Long: "Create an on-demand backup, in addition to whatever the instance's schedule " +
		"produces.\n\n" +
		"The backup counts towards your backup storage, which is billable beyond the " +
		"free allowance ('backup get-free-storage'), so the command confirms first.\n\n" +
		"--backup-type INCREMENTAL requires --parent-id, the backup to build on; the " +
		"default FULL needs nothing else.\n\n" +
		"--description is REQUIRED even though the API documents it as optional. Without " +
		"a non-empty description the request is accepted (HTTP 200, success true, a " +
		"backup ID handed back) and the backup then fails in the background with " +
		"\"An error occurred when communicating with system\" — measured 4 times out of 4, " +
		"against 2 successes with one. Requiring it here turns a silent, delayed failure " +
		"into an error before anything is sent.",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("instance-id", "", "Instance to back up (required)")
	f.String("name", "", "Backup name (required)")
	f.String("backup-type", "FULL", fmt.Sprintf("Backup type: %s", strings.Join(backupTypes, " or ")))
	f.String("parent-id", "", "Parent backup ID (required with --backup-type INCREMENTAL)")
	f.String("description", "", "Free-text description (required — see the note in --help)")
	f.Bool("dry-run", false, "Print the request that would be sent without taking a backup")
	f.Bool("force", false, "Skip the confirmation prompt")

	createCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	createCmd.MarkFlagRequired("name")        //nolint:errcheck
	createCmd.MarkFlagRequired("description") //nolint:errcheck

	createCmd.RegisterFlagCompletionFunc("instance-id", cli.ResourceCompletion(instanceResourceKey)) //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("backup-type", cli.FlagValues(backupTypes...))              //nolint:errcheck
	createCmd.RegisterFlagCompletionFunc("parent-id", cli.ResourceCompletion(BackupResourceKey))     //nolint:errcheck
}

func runCreate(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := vdbclient.RequireIDWithPrefix(instanceID, "instance-id", instanceIDPrefix,
		"Relational Database instance IDs start with 'db-'; PostgreSQL Cluster backups live under 'grn vdb postgresql backup'"); err != nil {
		return err
	}

	body, err := createBody(cmd, instanceID)
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
		"Take a %v backup %q of database instance %s? It counts towards billable backup storage.",
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
		return fmt.Errorf("failed to create backup %q of database instance %s: %w", name, instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

// createBody assembles CreateBackupRequest and enforces the one rule the API states
// but does not validate for us: an incremental backup needs a parent.
func createBody(cmd *cobra.Command, instanceID string) (map[string]interface{}, error) {
	flags := cmd.Flags()

	name, _ := flags.GetString("name")
	backupType, _ := flags.GetString("backup-type")
	parentID, _ := flags.GetString("parent-id")
	description, _ := flags.GetString("description")

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

	body := map[string]interface{}{
		"dbInstanceId": instanceID,
		"name":         name,
		"backupType":   backupType,
	}
	if parentID != "" {
		body["parentId"] = parentID
	}
	// description is required in practice, not just recommended: an absent OR EMPTY
	// one is accepted by the API and then fails asynchronously (verified live, see the
	// command's Long text). MarkFlagRequired catches an omitted flag; this catches
	// --description "".
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("--description must not be empty: the API accepts a backup " +
			"without one and then fails it in the background")
	}
	body["description"] = description

	return body, nil
}
