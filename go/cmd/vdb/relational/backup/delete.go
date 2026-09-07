package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a backup",
	Long: "Delete one backup.\n\n" +
		"This is irreversible and can break a chain: an INCREMENTAL backup builds on " +
		"its parent, so deleting a parent may make its children unrestorable. Check " +
		"'backup list --instance-id <id>' for children before deleting a FULL backup.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("backup-id", "", "Backup ID (required)")
	f.Bool("dry-run", false, "Print the request that would be sent without deleting")
	f.Bool("force", false, "Skip the confirmation prompt")

	deleteCmd.MarkFlagRequired("backup-id") //nolint:errcheck

	deleteCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey)) //nolint:errcheck
}

func runDelete(cmd *cobra.Command, args []string) error {
	backupID, _ := cmd.Flags().GetString("backup-id")
	if err := requireBackupID(backupID); err != nil {
		return err
	}

	// The body is a JSON ARRAY and the ID appears in it as well as in the path — one
	// of the six vdb endpoints shaped that way. MemoryStore does the same operation
	// as POST /backups/delete with no ID in the path at all.
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

	// DELETE with a body: the shared client has DeleteWithBody for exactly this.
	result, err := apiClient.DeleteWithBody(basePath+"/"+backupID+"/delete", body)
	if err != nil {
		return fmt.Errorf("failed to delete backup %s: %w", backupID, err)
	}

	return vdbclient.Output(cmd, result)
}

// printDeleteTarget shows what is about to go. Best-effort: a backup that cannot be
// read is reported and the user decides.
func printDeleteTarget(apiClient *vdbclient.Client, backupID string) {
	result, err := apiClient.Get(basePath+"/detail/"+backupID, nil)
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
	if parent, _ := backup["parent"].(string); parent != "" {
		fmt.Printf("  Parent:   %v\n", parent)
	}
	fmt.Println()
}
