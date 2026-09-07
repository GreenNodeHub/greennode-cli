package cluster

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var updateConfigGroupCmd = &cobra.Command{
	Use:   "update-config-group",
	Short: "Attach or detach the config group of a PostgreSQL Cluster",
	Long: "Attach a config group to a cluster, or detach the current one with --detach.\n\n" +
		"Only config groups with deploy type 'cluster' are accepted — see " +
		"'catalog list-config-groups'. Changing configuration parameters can restart the " +
		"database, so the command confirms first.",
	Args: cobra.NoArgs,
	RunE: runUpdateConfigGroup,
}

func init() {
	f := updateConfigGroupCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.String("config-id", "", "Config group ID to attach (required unless --detach)")
	f.Bool("detach", false, "Detach the current config group instead of attaching one")
	f.Bool("dry-run", false, "Print the request that would be sent without applying it")
	f.Bool("force", false, "Skip the confirmation prompt")

	updateConfigGroupCmd.MarkFlagRequired("cluster-id") //nolint:errcheck
	// Mutually exclusive rather than "empty means detach": the API detaches on an
	// empty string, which is far too easy to send by accident from a script whose
	// variable is unset.
	updateConfigGroupCmd.MarkFlagsMutuallyExclusive("config-id", "detach")

	// Bound here, next to the flags: see the init-order note in completion.go.
	updateConfigGroupCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
	updateConfigGroupCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())   //nolint:errcheck
}

func runUpdateConfigGroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	configID, _ := cmd.Flags().GetString("config-id")
	detach, _ := cmd.Flags().GetBool("detach")

	if !detach && configID == "" {
		return fmt.Errorf("pass --config-id to attach a config group, or --detach to remove the current one")
	}

	// An empty configGroupId is what the API reads as "detach".
	body := map[string]interface{}{"configGroupId": configID}

	action := fmt.Sprintf("attach config group %s to", configID)
	if detach {
		action = "detach the config group of"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("update", fmt.Sprintf("the config group of PostgreSQL Cluster %s", clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("%s PostgreSQL Cluster %s? This may restart the database.",
		capitalize(action), clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(pgPath(clusterID, "/config-group"), body)
	if err != nil {
		return fmt.Errorf("failed to update the config group of PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
