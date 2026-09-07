package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listReplicasCmd = &cobra.Command{
	Use:   "list-replicas",
	Short: "List the read replicas of a Relational Database instance",
	Long: "List the read replicas created from an instance.\n\n" +
		"The API answers with a reduced view of each replica — no ip, port or security " +
		"group — so run 'instance get' on a replica ID for its full details.",
	Args: cobra.NoArgs,
	RunE: runListReplicas,
}

func init() {
	f := listReplicasCmd.Flags()
	f.String("instance-id", "", "ID of the source instance whose replicas to list (required)")
	listReplicasCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	listReplicasCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
}

func runListReplicas(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := validateInstanceID(instanceID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// The path parameter is named replicaSourceId here — same instance, a third
	// name for it after dbInstanceId and instanceId.
	result, err := apiClient.Get(instancePath(instanceID, "/replicas"), nil)
	if err != nil {
		return fmt.Errorf("failed to list replicas of database instance %s: %w", instanceID, err)
	}

	return vdbclient.OutputWithColumns(cmd, result, replicaColumns)
}
