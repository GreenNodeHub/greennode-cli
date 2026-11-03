package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a Relational Database instance",
	Long: "Delete an instance and, optionally, its backups.\n\n" +
		"This is irreversible: the instance and its data are gone. " +
		"--create-final-backup takes one last backup first; --delete-all-backup also " +
		"removes the existing ones, which is the only way to lose the ability to " +
		"restore. Both are Relational Database options — a PostgreSQL Cluster accepts " +
		"only the first.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("instance-id", "", "Database instance ID (required)")
	f.Bool("create-final-backup", false, "Take a final backup before deleting")
	f.Bool("delete-all-backup", false, "Also delete every existing backup of this instance")
	f.Bool("dry-run", false, "Print the request that would be sent without deleting")
	f.Bool("force", false, "Skip the confirmation prompt")

	deleteCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	deleteCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
}

func runDelete(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	createFinalBackup, _ := cmd.Flags().GetBool("create-final-backup")
	deleteAllBackup, _ := cmd.Flags().GetBool("delete-all-backup")

	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, instanceID, "delete", map[string]interface{}{
		"createFinalBackup": createFinalBackup,
		"deleteAllBackup":   deleteAllBackup,
	})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Show what is about to be destroyed. Best-effort: if the instance cannot be
	// read (already gone, or a transient error), say so and let the user decide
	// rather than refusing to continue.
	printDeleteTarget(apiClient, instanceID)

	if dryRun {
		vdbclient.PreviewBody("delete", fmt.Sprintf("database instance %s", instanceID), body)
		return nil
	}

	prompt := fmt.Sprintf("Delete database instance %s? This cannot be undone.", instanceID)
	if deleteAllBackup {
		prompt = fmt.Sprintf(
			"Delete database instance %s AND every backup of it? This cannot be undone and leaves nothing to restore from.",
			instanceID)
	}
	if !cli.Confirm(force, prompt) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(instancePath(instanceID, "/delete"), body)
	if err != nil {
		return fmt.Errorf("failed to delete database instance %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

func printDeleteTarget(apiClient *vdbclient.Client, instanceID string) {
	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		fmt.Printf("Warning: could not read database instance %s before deleting (%v).\n", instanceID, err)
		return
	}

	fmt.Println("The following database instance will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", instance["id"])
	fmt.Printf("  Name:    %v\n", instance["name"])
	fmt.Printf("  Status:  %v\n", instance["status"])
	fmt.Printf("  Engine:  %v %v\n", instance["datastoreType"], instance["datastoreVersion"])
	fmt.Printf("  Storage: %v GB (%v)\n", instance["volumeSize"], instance["volumeType"])
	if replicas, ok := instance["replicas"].([]interface{}); ok && len(replicas) > 0 {
		fmt.Printf("  Replicas: %d — delete or detach them first if the API refuses\n", len(replicas))
	}
	fmt.Println()
}
