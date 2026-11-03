package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var resizeInstanceCmd = &cobra.Command{
	Use:   "resize-instance",
	Short: "Change the flavor (vCPU/RAM) of a MemoryStore instance",
	Long: "Move an instance to a different flavor.\n\n" +
		"THIS COSTS MONEY: order/payment flow, asynchronous, and Redis restarts on the " +
		"new flavor. Use --dry-run first.\n\n" +
		"This is the only resize MemoryStore has: an instance holds its data in the " +
		"flavor's RAM and has no volume, so there is no storage resize. Growing capacity " +
		"means a larger flavor.",
	Args: cobra.NoArgs,
	RunE: runResizeInstance,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a MemoryStore instance",
	Long: "Delete an instance and, optionally, its backups.\n\n" +
		"This is irreversible. --create-final-backup takes one last backup first; " +
		"--delete-all-backup also removes the existing ones, which is the only way to " +
		"lose the ability to restore.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

var detachReplicaCmd = &cobra.Command{
	Use:   "detach-replica",
	Short: "Promote a read replica to a standalone instance",
	Long: "Detach a read replica from its source, turning it into an independent " +
		"instance.\n\n" +
		"Pass the REPLICA's ID, not the source's — run 'instance list-replicas' on the " +
		"source to find it. Replication stops and cannot be re-established; the replica " +
		"carries on as a normal instance and keeps costing what it costs.",
	Args: cobra.NoArgs,
	RunE: runDetachReplica,
}

func init() {
	r := resizeInstanceCmd.Flags()
	r.String("instance-id", "", "MemoryStore instance ID (required)")
	r.String("package-id", "", "New flavor id, numeric (required; the 'id' column of 'catalog list-flavors')")
	r.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	r.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	r.Bool("force", false, "Skip the confirmation prompt")
	resizeInstanceCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	resizeInstanceCmd.MarkFlagRequired("package-id")  //nolint:errcheck

	d := deleteCmd.Flags()
	d.String("instance-id", "", "MemoryStore instance ID (required)")
	d.Bool("create-final-backup", false, "Take a final backup before deleting")
	d.Bool("delete-all-backup", false, "Also delete every existing backup of this instance")
	d.Bool("dry-run", false, "Print the request that would be sent without deleting")
	d.Bool("force", false, "Skip the confirmation prompt")
	deleteCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	t := detachReplicaCmd.Flags()
	t.String("instance-id", "", "ID of the replica to detach (required)")
	t.Bool("dry-run", false, "Print the request that would be sent without detaching")
	t.Bool("force", false, "Skip the confirmation prompt")
	detachReplicaCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	for _, cmd := range []*cobra.Command{resizeInstanceCmd, deleteCmd, detachReplicaCmd} {
		cmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
	}
}

func runResizeInstance(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	packageID, _ := cmd.Flags().GetString("package-id")
	poc, _ := cmd.Flags().GetBool("poc")

	// ResizeBody, not ActionBody: the resize endpoints spell the resource-type field
	// "resourceType" where the action endpoints use "resType".
	body := vdbclient.ResizeBody(vdbclient.ResourceTypeInstance, instanceID, "resize",
		map[string]interface{}{
			"packageId": packageID,
			"poc":       poc,
		})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("resize", fmt.Sprintf("MemoryStore instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Move MemoryStore instance %s to flavor %s? This places a paid order and restarts Redis.",
		instanceID, packageID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(instancePath(instanceID, "/resize-instance"), body)
	if err != nil {
		return fmt.Errorf("failed to resize MemoryStore instance %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runDelete(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	createFinalBackup, _ := cmd.Flags().GetBool("create-final-backup")
	deleteAllBackup, _ := cmd.Flags().GetBool("delete-all-backup")

	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, instanceID, "delete",
		map[string]interface{}{
			"createFinalBackup": createFinalBackup,
			"deleteAllBackup":   deleteAllBackup,
		})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	printDeleteTarget(apiClient, instanceID)

	if dryRun {
		vdbclient.PreviewBody("delete", fmt.Sprintf("MemoryStore instance %s", instanceID), body)
		return nil
	}

	prompt := fmt.Sprintf("Delete MemoryStore instance %s? This cannot be undone.", instanceID)
	if deleteAllBackup {
		prompt = fmt.Sprintf(
			"Delete MemoryStore instance %s AND every backup of it? This cannot be undone and leaves nothing to restore from.",
			instanceID)
	}
	if !cli.Confirm(force, prompt) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(instancePath(instanceID, "/delete"), body)
	if err != nil {
		return fmt.Errorf("failed to delete MemoryStore instance %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

func runDetachReplica(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	// detach_replica, with an underscore — the only action in the family that is not
	// a single word.
	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, instanceID, "detach_replica", nil)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("detach", fmt.Sprintf("replica %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Detach replica %s from its source? Replication cannot be re-established.", instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(instancePath(instanceID, "/detach-replica"), body)
	if err != nil {
		return fmt.Errorf("failed to detach replica %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

func printDeleteTarget(apiClient *vdbclient.Client, instanceID string) {
	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		fmt.Printf("Warning: could not read MemoryStore instance %s before deleting (%v).\n", instanceID, err)
		return
	}

	fmt.Println("The following MemoryStore instance will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", instance["id"])
	fmt.Printf("  Name:    %v\n", instance["name"])
	fmt.Printf("  Status:  %v\n", instance["status"])
	fmt.Printf("  Engine:  %v %v\n", instance["datastoreType"], instance["datastoreVersion"])
	fmt.Printf("  Flavor:  %v vCPU / %v GB RAM\n", instance["vcpus"], instance["ram"])
	if replicas, ok := instance["replicas"].([]interface{}); ok && len(replicas) > 0 {
		fmt.Printf("  Replicas: %d — delete or detach them first if the API refuses\n", len(replicas))
	}
	fmt.Println()
}

// replicaFlagsOn is shared by create-replica; kept here so the flag set can be built
// for tests without touching the mounted command.
func replicaFlagsOn(f *pflag.FlagSet) {
	f.String("instance-id", "", "ID of the source instance to replicate (required)")
	f.String("name", "", "Name of the new replica (required)")
	f.String("package-id", "", "Flavor id for the replica (default: same as the source)")
	f.String("zone-id", "", "Availability zone (default: same as the source)")
	f.String("subnet-ids", "", "Subnet ID(s), comma-separated (default: the source's subnet)")
	f.String("config-id", "", "Config group ID to attach (default: the source's, if any)")
	f.Bool("public-access", false, "Allow public access to the replica (default: same as the source)")
	f.Bool("backup-auto", false, "Enable daily automatic backup (default: same as the source)")
	f.Int("backup-duration", 0, fmt.Sprintf("Backup retention in days, %d-%d", minBackupDuration, maxBackupDuration))
	f.String("backup-time", "", "Time of day to run the backup, HH:MM")
	f.String("redis-password", "", "Redis master password for the replica (defaults to $"+passwordEnv+")")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}
