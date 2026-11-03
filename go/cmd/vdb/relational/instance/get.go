package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a Relational Database instance",
	Long: "Show the full details of a single vDB Relational Database instance.\n\n" +
		"This also accepts the PostgreSQL Cluster IDs ('pg-') that appear in " +
		"'instance list' — verified against the live API. Cluster records answer with " +
		"deployType 'cluster', numberOfNodes and the privateRwIp/publicRwIp fields " +
		"instead of a single instance's ip list.",
	RunE: runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("instance-id", "", "Database instance ID (required)")
	getCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	getCmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck
}

func runGet(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")

	if err := validator.ValidateID(instanceID, "instance-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(detailPath(instanceID), nil)
	if err != nil {
		return fmt.Errorf("failed to get database instance %s: %w", instanceID, err)
	}

	// Output, not OutputWithColumns: a single instance carries four nested arrays
	// (ip, securityGroup, replicas, sharedActions) and table extraction would pick
	// one of them at random. See internal/vdbclient/output.go.
	return vdbclient.Output(cmd, result)
}
