package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteAddonCmd = &cobra.Command{
	Use:   "delete-addon",
	Short: "Delete an add-on from a cluster",
	Long: `Delete an add-on from a cluster.

With --preserve, VKS stops managing the add-on but keeps its Helm release (and
workloads) in the cluster. The add-on goes to DELETING; use
"grn vks wait addon-deleted" to wait until it is gone.`,
	RunE: runDeleteAddon,
}

func init() {
	f := deleteAddonCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("addon-name", "", "Add-on name (required)")
	f.Bool("preserve", false, "Keep the add-on's Helm release in the cluster")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
	f.Bool("force", false, "Skip confirmation prompt")

	deleteAddonCmd.MarkFlagRequired("cluster-id")
	deleteAddonCmd.MarkFlagRequired("addon-name")
	_ = deleteAddonCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchClusterAddonNames))
}

func runDeleteAddon(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	addonName, _ := cmd.Flags().GetString("addon-name")
	preserve, _ := cmd.Flags().GetBool("preserve")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validateAddonName(addonName); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/clusters/%s/addons/%s", clusterID, addonName)
	current, err := apiClient.Get(path, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch add-on: %w", err)
	}
	currentMap, _ := current.(map[string]interface{})
	addon, _ := currentMap["addon"].(map[string]interface{})
	namespace := ""
	if ns, ok := addon["namespaceConfig"].(map[string]interface{}); ok {
		namespace, _ = ns["namespace"].(string)
	}

	fmt.Println("The following add-on will be deleted:")
	fmt.Println()
	fmt.Printf("  Cluster:    %s\n", clusterID)
	fmt.Printf("  Add-on:     %v\n", addon["addonName"])
	fmt.Printf("  Version:    %v\n", addon["addonVersion"])
	fmt.Printf("  Status:     %v\n", addon["status"])
	fmt.Printf("  Namespace:  %s\n", namespace)
	fmt.Println()
	if preserve {
		fmt.Println("VKS will stop managing it; the Helm release stays in the cluster (--preserve).")
	} else {
		fmt.Println("Its Helm release will be uninstalled. This action is irreversible.")
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}

	if !cli.Confirm(force, "Are you sure you want to delete this add-on?") {
		fmt.Println("Aborted.")
		return nil
	}

	var params map[string]string
	if preserve {
		params = map[string]string{"preserve": "true"}
	}
	result, err := apiClient.Delete(path, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
