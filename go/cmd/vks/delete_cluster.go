package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteClusterCmd = &cobra.Command{
	Use:   "delete-cluster",
	Short: "Delete a VKS cluster",
	RunE:  runDeleteCluster,
}

func init() {
	f := deleteClusterCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
	f.Bool("force", false, "Skip confirmation prompt")

	deleteClusterCmd.MarkFlagRequired("cluster-id")
}

func runDeleteCluster(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Fetch cluster info for preview
	cluster, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch cluster: %w", err)
	}

	nodegroups, err := apiClient.Get(
		fmt.Sprintf("/v1/clusters/%s/node-groups", clusterID),
		map[string]string{"page": "0", "pageSize": "50"},
	)
	if err != nil {
		return fmt.Errorf("failed to fetch node groups: %w", err)
	}

	// Show preview
	printClusterPreview(cluster, nodegroups)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}

	if !cli.Confirm(force, "Are you sure you want to delete this cluster?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v1/clusters/%s", clusterID), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}

func printClusterPreview(cluster, nodegroups interface{}) {
	clusterMap, _ := cluster.(map[string]interface{})
	fmt.Println("The following resources will be deleted:")
	fmt.Println()
	fmt.Println("Cluster:")
	fmt.Printf("  ID:      %v\n", clusterMap["id"])
	fmt.Printf("  Name:    %v\n", clusterMap["name"])
	fmt.Printf("  Status:  %v\n", clusterMap["status"])
	fmt.Printf("  Version: %v\n", clusterMap["version"])
	fmt.Printf("  Nodes:   %v\n", clusterMap["numNodes"])
	fmt.Println()

	ngMap, _ := nodegroups.(map[string]interface{})
	items, _ := ngMap["items"].([]interface{})
	if len(items) > 0 {
		fmt.Printf("Node groups (%d):\n", len(items))
		for _, item := range items {
			ng, _ := item.(map[string]interface{})
			fmt.Printf("  - %v (ID: %v, nodes: %v)\n", ng["name"], ng["id"], ng["numNodes"])
		}
	} else {
		fmt.Println("Node groups: none")
	}

	fmt.Println("\nThis action is irreversible.")
}
