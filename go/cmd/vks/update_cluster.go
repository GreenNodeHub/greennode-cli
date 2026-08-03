package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateClusterCmd = &cobra.Command{
	Use:   "update-cluster",
	Short: "Update a VKS cluster",
	RunE:  runUpdateCluster,
}

func init() {
	f := updateClusterCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("k8s-version", "", "New Kubernetes version; unset = unchanged")
	f.String("whitelist-node-cidrs", "", "Whitelist CIDRs, comma-separated; unset = unchanged")
	f.String("load-balancer-plugin", "", "Load balancer plugin (enabled, disabled); unset = unchanged")
	f.String("block-store-csi-plugin", "", "Block store CSI plugin (enabled, disabled); unset = unchanged")
	f.Bool("dry-run", false, "Validate parameters without updating")

	updateClusterCmd.MarkFlagRequired("cluster-id")
}

func runUpdateCluster(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	// All body fields are optional (partial update) — send only what the user set.
	body := map[string]any{}

	if cmd.Flags().Changed("k8s-version") {
		v, _ := cmd.Flags().GetString("k8s-version")
		body["version"] = v
	}
	if cmd.Flags().Changed("whitelist-node-cidrs") {
		v, _ := cmd.Flags().GetString("whitelist-node-cidrs")
		body["whitelistNodeCIDRs"] = parseCommaSeparated(v)
	}
	if cmd.Flags().Changed("load-balancer-plugin") {
		v, _ := cmd.Flags().GetString("load-balancer-plugin")
		enabled, err := parseToggle("load-balancer-plugin", v)
		if err != nil {
			return err
		}
		body["enabledLoadBalancerPlugin"] = enabled
	}
	if cmd.Flags().Changed("block-store-csi-plugin") {
		v, _ := cmd.Flags().GetString("block-store-csi-plugin")
		enabled, err := parseToggle("block-store-csi-plugin", v)
		if err != nil {
			return err
		}
		body["enabledBlockStoreCsiPlugin"] = enabled
	}

	if len(body) == 0 {
		return fmt.Errorf("nothing to update: provide at least one of --k8s-version, --whitelist-node-cidrs, --load-balancer-plugin, or --block-store-csi-plugin")
	}

	if dryRun {
		fmt.Println("=== DRY RUN: Update cluster ===")
		fmt.Println()
		fmt.Printf("Cluster ID: %s\n", clusterID)
		for key, value := range body {
			fmt.Printf("  %s: %v\n", key, value)
		}
		fmt.Println("\nRun without --dry-run to update.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(fmt.Sprintf("/v1/clusters/%s", clusterID), body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
