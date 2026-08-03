package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateNodegroupMetadataCmd = &cobra.Command{
	Use:   "update-nodegroup-metadata",
	Short: "Update labels, tags, and taints of a node group",
	RunE:  runUpdateNodegroupMetadata,
}

func init() {
	f := updateNodegroupMetadataCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Node group ID (required)")
	f.String("labels", "", "Node labels as key=value pairs (comma-separated)")
	f.String("tags", "", "Tags as key=value pairs (comma-separated)")
	f.String("taints", "", "Node taints as key=value:effect (comma-separated)")
	f.Bool("dry-run", false, "Preview the metadata update without executing")

	updateNodegroupMetadataCmd.MarkFlagRequired("cluster-id")
	updateNodegroupMetadataCmd.MarkFlagRequired("nodegroup-id")
}

func buildMetadataBody(labelsStr, tagsStr, taintsStr string, changed map[string]bool) map[string]interface{} {
	body := map[string]interface{}{}
	if changed["labels"] {
		body["labels"] = parseLabels(labelsStr)
	}
	if changed["tags"] {
		body["tags"] = parseLabels(tagsStr)
	}
	if changed["taints"] {
		body["taints"] = parseTaints(taintsStr)
	}
	return body
}

func runUpdateNodegroupMetadata(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	labelsStr, _ := cmd.Flags().GetString("labels")
	tagsStr, _ := cmd.Flags().GetString("tags")
	taintsStr, _ := cmd.Flags().GetString("taints")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}

	changed := map[string]bool{
		"labels": cmd.Flags().Changed("labels"),
		"tags":   cmd.Flags().Changed("tags"),
		"taints": cmd.Flags().Changed("taints"),
	}
	if !changed["labels"] && !changed["tags"] && !changed["taints"] {
		return fmt.Errorf("at least one of --labels, --tags, --taints must be provided")
	}
	body := buildMetadataBody(labelsStr, tagsStr, taintsStr, changed)

	if dryRun {
		cli.PrintDryRun("update", fmt.Sprintf("metadata for node group %s", nodegroupID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Patch(
		fmt.Sprintf("/v1/clusters/%s/node-groups/%s/metadata", clusterID, nodegroupID), body,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
