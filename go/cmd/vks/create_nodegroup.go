package vks

import (
	"fmt"
	"os"
	"regexp"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var createNodegroupCmd = &cobra.Command{
	Use:   "create-nodegroup",
	Short: "Create a new node group",
	RunE:  runCreateNodegroup,
}

func init() {
	f := createNodegroupCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("name", "", "Node group name (required)")
	f.String("os", "ubuntu", "Node group OS image (ubuntu, linux, rocky)")
	f.String("flavor-id", "", "Flavor ID (required)")
	f.String("disk-type", "", "Disk type ID (required)")
	f.String("ssh-key-id", "", "SSH key ID (required)")
	f.String("private-nodes", "disabled", "Private nodes (enabled, disabled)")
	f.Int("num-nodes", 1, "Number of nodes (0-10)")
	f.Int("disk-size", 100, "Disk size in GiB (20-5000)")
	f.String("security-groups", "", "Security group IDs (comma-separated)")
	f.String("subnet-id", "", "Subnet ID for node group")
	f.String("labels", "", "Node labels as key=value pairs (comma-separated)")
	f.String("taints", "", "Node taints as key=value:effect (comma-separated)")
	f.Bool("enable-encryption-volume", false, "Enable volume encryption")
	f.String("tags", "", "Node group tags as key=value pairs (comma-separated)")
	f.String("secondary-subnets", "", "Secondary subnet IDs (comma-separated)")
	f.String("auto-scale", "", "Auto-scale config (shorthand minSize=2,maxSize=10 or JSON)")
	f.String("placement-group", "", "Placement group config (shorthand type=NEW,placementGroupName=pg or JSON)")
	f.String("upgrade-config", "", "Upgrade config (shorthand maxSurge=1,maxUnavailable=0,strategy=SURGE or JSON); default SURGE 1/0")
	f.Bool("dry-run", false, "Validate parameters without creating")

	for _, name := range []string{"cluster-id", "name", "flavor-id", "disk-type", "ssh-key-id"} {
		createNodegroupCmd.MarkFlagRequired(name)
	}
}

func runCreateNodegroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	name, _ := cmd.Flags().GetString("name")
	osImage, _ := cmd.Flags().GetString("os")
	flavorID, _ := cmd.Flags().GetString("flavor-id")
	diskType, _ := cmd.Flags().GetString("disk-type")
	sshKeyID, _ := cmd.Flags().GetString("ssh-key-id")
	privateNodesVal, _ := cmd.Flags().GetString("private-nodes")
	numNodes, _ := cmd.Flags().GetInt("num-nodes")
	diskSize, _ := cmd.Flags().GetInt("disk-size")
	securityGroups, _ := cmd.Flags().GetString("security-groups")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	labelsStr, _ := cmd.Flags().GetString("labels")
	taintsStr, _ := cmd.Flags().GetString("taints")
	enableEncryption, _ := cmd.Flags().GetBool("enable-encryption-volume")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	tagsStr, _ := cmd.Flags().GetString("tags")
	secondarySubnets, _ := cmd.Flags().GetString("secondary-subnets")
	autoScaleStr, _ := cmd.Flags().GetString("auto-scale")
	placementGroupStr, _ := cmd.Flags().GetString("placement-group")
	upgradeConfigStr, _ := cmd.Flags().GetString("upgrade-config")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	enablePrivateNodes, err := parseToggle("private-nodes", privateNodesVal)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"name":                    name,
		"numNodes":                numNodes,
		"os":                      osImage,
		"flavorId":                flavorID,
		"diskSize":                diskSize,
		"diskType":                diskType,
		"enablePrivateNodes":      enablePrivateNodes,
		"sshKeyId":                sshKeyID,
		"enabledEncryptionVolume": enableEncryption,
		"securityGroups":          []string{},
		"upgradeConfig": map[string]interface{}{
			"maxSurge":       1,
			"maxUnavailable": 0,
			"strategy":       "SURGE",
		},
	}

	if securityGroups != "" {
		body["securityGroups"] = parseCommaSeparated(securityGroups)
	}
	if subnetID != "" {
		body["subnetId"] = subnetID
	}
	if labelsStr != "" {
		body["labels"] = parseLabels(labelsStr)
	}
	if taintsStr != "" {
		body["taints"] = parseTaints(taintsStr)
	}
	if tagsStr != "" {
		body["tags"] = parseLabels(tagsStr)
	}
	if secondarySubnets != "" {
		body["secondarySubnets"] = parseCommaSeparated(secondarySubnets)
	}
	if autoScaleStr != "" {
		asc, err := cli.ParseStructFlag(autoScaleStr, "minSize", "maxSize")
		if err != nil {
			return fmt.Errorf("--auto-scale: %w", err)
		}
		body["autoScaleConfig"] = asc
	}
	if placementGroupStr != "" {
		pg, err := cli.ParseStructFlag(placementGroupStr)
		if err != nil {
			return fmt.Errorf("--placement-group: %w", err)
		}
		body["placementGroupConfigDto"] = pg
	}
	if upgradeConfigStr != "" {
		uc, err := cli.ParseStructFlag(upgradeConfigStr, "maxSurge", "maxUnavailable")
		if err != nil {
			return fmt.Errorf("--upgrade-config: %w", err)
		}
		body["upgradeConfig"] = uc
	}

	if dryRun {
		return validateCreateNodegroup(name, diskSize, numNodes)
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(
		fmt.Sprintf("/v1/clusters/%s/node-groups", clusterID), body,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}

func validateCreateNodegroup(name string, diskSize, numNodes int) error {
	ngNameRE := regexp.MustCompile(`^[a-z0-9][a-z0-9-]{3,13}[a-z0-9]$`)
	var errors []string

	if !ngNameRE.MatchString(name) {
		errors = append(errors, fmt.Sprintf(
			"Node group name '%s' is invalid. Must be 5-15 chars, lowercase alphanumeric and hyphens.", name))
	}
	if diskSize < 20 || diskSize > 5000 {
		errors = append(errors, fmt.Sprintf("Disk size %d out of range (20-5000 GiB)", diskSize))
	}
	if numNodes < 0 || numNodes > 10 {
		errors = append(errors, fmt.Sprintf("Number of nodes %d out of range (0-10)", numNodes))
	}

	fmt.Println("=== DRY RUN: Validation results ===")
	fmt.Println()
	if len(errors) > 0 {
		fmt.Printf("Found %d error(s):\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("All parameters are valid. Run without --dry-run to create.")
	return nil
}
