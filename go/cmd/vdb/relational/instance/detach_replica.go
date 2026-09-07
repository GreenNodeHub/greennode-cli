package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var detachReplicaCmd = &cobra.Command{
	Use:   "detach-replica",
	Short: "Promote a read replica to a standalone instance",
	Long: "Detach a read replica from its source, turning it into an independent " +
		"instance.\n\n" +
		"Pass the REPLICA's ID, not the source's. Replication stops and cannot be " +
		"re-established — the replica keeps its data and carries on as a normal " +
		"instance, and it keeps costing what it costs. Run 'instance list-replicas' on " +
		"the source to see which IDs are replicas.",
	Args: cobra.NoArgs,
	RunE: runDetachReplica,
}

func init() {
	f := detachReplicaCmd.Flags()
	f.String("instance-id", "", "ID of the replica to detach (required)")
	f.Bool("dry-run", false, "Print the request that would be sent without detaching")
	f.Bool("force", false, "Skip the confirmation prompt")

	detachReplicaCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	detachReplicaCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
}

func runDetachReplica(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	// The API's action value is detach_replica, with an underscore — the only action
	// in the family that is not a single word.
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
