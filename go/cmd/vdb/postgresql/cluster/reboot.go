package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Reboot a PostgreSQL Cluster",
	Long: "Restart the database on a cluster.\n\n" +
		"Served by the Relational Database reboot endpoint. Connections are dropped " +
		"while the nodes restart, so the command confirms first.",
	Args: cobra.NoArgs,
	RunE: runReboot,
}

func init() {
	f := rebootCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.Bool("dry-run", false, "Print the request that would be sent without rebooting")
	f.Bool("force", false, "Skip the confirmation prompt")

	rebootCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	rebootCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runReboot(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, clusterID, "reboot", nil)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("reboot", fmt.Sprintf("PostgreSQL Cluster %s", clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Reboot PostgreSQL Cluster %s? Open connections will be dropped.", clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(relPath(clusterID, "/reboot"), body)
	if err != nil {
		return fmt.Errorf("failed to reboot PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}
