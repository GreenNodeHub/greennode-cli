package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateNodegroupCmd = &cobra.Command{
	Use:   "update-nodegroup",
	Short: "Update a node group",
	RunE:  runUpdateNodegroup,
}

func init() {
	f := updateNodegroupCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Node group ID (required)")
	f.String("num-nodes", "", "New number of nodes")
	f.String("security-groups", "", "Security group IDs (comma-separated)")
	f.String("auto-scale", "", "Auto-scale config (shorthand minSize=2,maxSize=10 or JSON)")
	f.String("upgrade-config", "", "Upgrade config (shorthand maxSurge=1,maxUnavailable=0,strategy=SURGE or JSON)")
	f.Bool("dry-run", false, "Preview update without executing")

	updateNodegroupCmd.MarkFlagRequired("cluster-id")
	updateNodegroupCmd.MarkFlagRequired("nodegroup-id")
}

func runUpdateNodegroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	numNodes, _ := cmd.Flags().GetString("num-nodes")
	securityGroups, _ := cmd.Flags().GetString("security-groups")
	autoScaleStr, _ := cmd.Flags().GetString("auto-scale")
	upgradeConfigStr, _ := cmd.Flags().GetString("upgrade-config")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}

	body := map[string]interface{}{}

	if numNodes != "" {
		body["numNodes"] = toInt(numNodes)
	}
	if securityGroups != "" {
		body["securityGroups"] = parseCommaSeparated(securityGroups)
	}
	if autoScaleStr != "" {
		asc, err := cli.ParseStructFlag(autoScaleStr, "minSize", "maxSize")
		if err != nil {
			return fmt.Errorf("--auto-scale: %w", err)
		}
		body["autoScaleConfig"] = asc
	}
	if upgradeConfigStr != "" {
		uc, err := cli.ParseStructFlag(upgradeConfigStr, "maxSurge", "maxUnavailable")
		if err != nil {
			return fmt.Errorf("--upgrade-config: %w", err)
		}
		body["upgradeConfig"] = uc
	}

	if len(body) == 0 {
		return fmt.Errorf("nothing to update: provide at least one of --num-nodes, --security-groups, --auto-scale, or --upgrade-config (use 'update-nodegroup-metadata' for labels/tags/taints)")
	}

	if dryRun {
		fmt.Println("=== DRY RUN: Update node group ===")
		fmt.Println()
		fmt.Printf("Cluster ID: %s\n", clusterID)
		fmt.Printf("Node group ID: %s\n", nodegroupID)
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

	result, err := apiClient.Put(
		fmt.Sprintf("/v1/clusters/%s/node-groups/%s", clusterID, nodegroupID), body,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}

func toInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
