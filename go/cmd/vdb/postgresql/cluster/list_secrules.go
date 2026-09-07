package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listSecrulesCmd = &cobra.Command{
	Use:   "list-secrules",
	Short: "List the security rules of a PostgreSQL Cluster",
	Long: "List the security group rules controlling access to a cluster.\n\n" +
		"Served by the Relational Database secrules endpoint. Use " +
		"'cluster update-secrule' to change them — note that update REPLACES the whole " +
		"rule set, so start from this listing.",
	Args: cobra.NoArgs,
	RunE: runListSecrules,
}

func init() {
	f := listSecrulesCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	listSecrulesCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	listSecrulesCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runListSecrules(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := fetchSecrules(apiClient, clusterID)
	if err != nil {
		return err
	}

	return vdbclient.OutputWithColumns(cmd, result, vdbclient.SecurityRuleColumns)
}

// fetchSecrules is shared with update-secrule, which reads the current rules to
// show what a replacement would change.
func fetchSecrules(apiClient interface {
	Get(string, map[string]string) (interface{}, error)
}, clusterID string) (interface{}, error) {
	result, err := apiClient.Get(relPath(clusterID, "/secrules"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list security rules of PostgreSQL Cluster %s: %w", clusterID, err)
	}
	return result, nil
}
