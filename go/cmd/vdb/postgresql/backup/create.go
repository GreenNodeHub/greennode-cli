package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Take an on-demand backup of a PostgreSQL Cluster",
	Long: "Take a backup of a cluster now, in addition to whatever its policy schedules.\n\n" +
		"The backup is stored in the cluster's backup location and counts towards its " +
		"backup storage, which is chargeable beyond the free allowance. The API takes no " +
		"parameters — the location, policy and retention come from the cluster's backup " +
		"record ('backup get').",
	Args: cobra.NoArgs,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.Bool("dry-run", false, "Print the request that would be sent without taking a backup")
	f.Bool("force", false, "Skip the confirmation prompt")

	createCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in
	// cmd/vdb/relational/instance/completion.go.
	createCmd.RegisterFlagCompletionFunc("cluster-id", cli.ResourceCompletion(clusterResourceKey)) //nolint:errcheck
}

func runCreate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	path := clusterPath(clusterID, "/backup-now")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Printf("Would POST %s (no request body).\n", path)
		cli.DryRunNotice("take the backup")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Take a backup of PostgreSQL Cluster %s now? It counts towards billable backup storage.", clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// The endpoint takes no body; the API answers with a plain string inside the
	// envelope rather than an object.
	result, err := apiClient.Post(path, nil)
	if err != nil {
		return fmt.Errorf("failed to back up PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}
