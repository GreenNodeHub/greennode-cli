package cluster

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a PostgreSQL Cluster",
	Long: "Delete a cluster.\n\n" +
		"Served by the Relational Database delete endpoint. This is irreversible: the " +
		"cluster and its data are gone. --create-final-backup takes one last backup " +
		"first.\n\n" +
		"A cluster takes only that one option. The shared request schema also carries " +
		"deleteAllBackup, which applies to Relational Database instances and not here, " +
		"so existing backups survive the deletion and stay restorable.",
	Args: cobra.NoArgs,
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.Bool("create-final-backup", false, "Take a final backup before deleting")
	f.Bool("dry-run", false, "Print the request that would be sent without deleting")
	f.Bool("force", false, "Skip the confirmation prompt")

	deleteCmd.MarkFlagRequired("cluster-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	deleteCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion()) //nolint:errcheck
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	createFinalBackup, _ := cmd.Flags().GetBool("create-final-backup")

	// createFinalBackup is the only per-cluster option this endpoint honours. The
	// shared DeleteDbInstanceConfig schema also has deleteAllBackup, but that is a
	// Relational Database instance option, so it is not sent here.
	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, clusterID, "delete", map[string]interface{}{
		"createFinalBackup": createFinalBackup,
	})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Show what is about to be destroyed. A best-effort lookup: if the cluster
	// cannot be read (already gone, or a transient error), say so and let the user
	// decide rather than refusing to continue.
	printDeleteTarget(apiClient, clusterID)

	if dryRun {
		vdbclient.PreviewBody("delete", fmt.Sprintf("PostgreSQL Cluster %s", clusterID), body)
		return nil
	}

	if !cli.Confirm(force, fmt.Sprintf(
		"Delete PostgreSQL Cluster %s? This cannot be undone.", clusterID)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(relPath(clusterID, "/delete"), body)
	if err != nil {
		return fmt.Errorf("failed to delete PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

func printDeleteTarget(apiClient interface {
	Get(string, map[string]string) (interface{}, error)
}, clusterID string) {
	result, err := apiClient.Get(detailPath(clusterID), nil)
	if err != nil {
		fmt.Printf("Warning: could not read PostgreSQL Cluster %s before deleting (%v).\n", clusterID, err)
		return
	}
	cluster, ok := vdbclient.Unwrap(result).(map[string]interface{})
	if !ok {
		return
	}

	fmt.Println("The following PostgreSQL Cluster will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", cluster["id"])
	fmt.Printf("  Name:    %v\n", cluster["name"])
	fmt.Printf("  Status:  %v\n", cluster["status"])
	fmt.Printf("  Version: PostgreSQL %v\n", cluster["datastoreVersion"])
	fmt.Printf("  Nodes:   %v\n", cluster["numberOfNodes"])
	fmt.Printf("  Storage: %v GB (%v)\n", cluster["volumeSize"], cluster["volumeType"])
	fmt.Println()
}
