package vks

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/cli"
)

var createClusterCmd = &cobra.Command{
	Use:   "create-cluster",
	Short: "Create a new VKS cluster",
	Long: "Create a new VKS cluster (control plane only). " +
		"Add worker nodes afterwards with 'grn vks create-nodegroup'.",
	RunE: runCreateCluster,
}

func init() {
	f := createClusterCmd.Flags()
	// Cluster settings (required)
	f.String("name", "", "Cluster name (required)")
	f.String("k8s-version", "", "Kubernetes version (required)")
	f.String("network-type", "", "Network type: TIGERA, CILIUM_OVERLAY, CILIUM_NATIVE_ROUTING (required). TIGERA/CILIUM_OVERLAY need --cidr; CILIUM_NATIVE_ROUTING needs --node-netmask-size and --secondary-subnets")
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("subnet-id", "", "Subnet ID")

	for _, name := range []string{"name", "k8s-version", "network-type", "vpc-id"} {
		createClusterCmd.MarkFlagRequired(name)
	}

	// Cluster settings (optional)
	f.String("cidr", "", "CIDR block (required for TIGERA and CILIUM_OVERLAY)")
	f.String("description", "", "Cluster description")
	f.String("private-cluster", "disabled", "Private cluster (enabled, disabled)")
	f.String("release-channel", "STABLE", "Release channel (RAPID, STABLE)")
	f.String("load-balancer-plugin", "enabled", "Load balancer plugin (enabled, disabled)")
	f.String("block-store-csi-plugin", "enabled", "Block store CSI plugin (enabled, disabled)")
	f.String("service-endpoint", "disabled", "Service endpoint (enabled, disabled)")
	f.String("az-strategy", "SINGLE", "Availability zone strategy")
	f.String("secondary-subnets", "", "Secondary subnet CIDRs, comma-separated, e.g. 10.5.60.0/22 (required for CILIUM_NATIVE_ROUTING, at least one, max 10). NOT subnet IDs")
	f.String("list-subnet-ids", "", "Subnet IDs for the cluster (comma-separated)")
	f.Int("node-netmask-size", 0, "Node netmask size: 24, 25, or 26 (required for CILIUM_NATIVE_ROUTING)")
	f.String("auto-upgrade-config", "", "Auto-upgrade config (shorthand time=03:00,weekdays=Mon or JSON; use JSON for multiple weekdays)")
	f.String("auto-healing-config", "", "Auto-healing config; set exactly one of maxUnhealthy or unhealthyRange (shorthand enableAutoHealing=true,maxUnhealthy=20%,timeoutUnhealthy=10 or JSON)")
	f.Bool("dry-run", false, "Validate parameters without creating the cluster")
}

func runCreateCluster(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	k8sVersion, _ := cmd.Flags().GetString("k8s-version")
	networkType, _ := cmd.Flags().GetString("network-type")
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	cidr, _ := cmd.Flags().GetString("cidr")
	description, _ := cmd.Flags().GetString("description")
	releaseChannel, _ := cmd.Flags().GetString("release-channel")
	azStrategy, _ := cmd.Flags().GetString("az-strategy")
	secondarySubnets, _ := cmd.Flags().GetString("secondary-subnets")
	listSubnetIDs, _ := cmd.Flags().GetString("list-subnet-ids")
	autoUpgradeStr, _ := cmd.Flags().GetString("auto-upgrade-config")
	autoHealingStr, _ := cmd.Flags().GetString("auto-healing-config")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Parse enabled/disabled toggle flags.
	privateClusterVal, _ := cmd.Flags().GetString("private-cluster")
	lbPluginVal, _ := cmd.Flags().GetString("load-balancer-plugin")
	csiPluginVal, _ := cmd.Flags().GetString("block-store-csi-plugin")
	serviceEndpointVal, _ := cmd.Flags().GetString("service-endpoint")
	enablePrivateCluster, err := parseToggle("private-cluster", privateClusterVal)
	if err != nil {
		return err
	}
	enabledLBPlugin, err := parseToggle("load-balancer-plugin", lbPluginVal)
	if err != nil {
		return err
	}
	enabledCSIPlugin, err := parseToggle("block-store-csi-plugin", csiPluginVal)
	if err != nil {
		return err
	}
	enabledServiceEndpoint, err := parseToggle("service-endpoint", serviceEndpointVal)
	if err != nil {
		return err
	}

	// Build cluster body. Node groups are created separately via
	// 'grn vks create-nodegroup'.
	body := map[string]interface{}{
		"name":                       name,
		"version":                    k8sVersion,
		"networkType":                networkType,
		"vpcId":                      vpcID,
		"enablePrivateCluster":       enablePrivateCluster,
		"releaseChannel":             releaseChannel,
		"enabledBlockStoreCsiPlugin": enabledCSIPlugin,
		"enabledLoadBalancerPlugin":  enabledLBPlugin,
		"enabledServiceEndpoint":     enabledServiceEndpoint,
		"azStrategy":                 azStrategy,
	}

	if subnetID != "" {
		body["subnetId"] = subnetID
	}
	if cidr != "" {
		body["cidr"] = cidr
	}
	if description != "" {
		body["description"] = description
	}
	if secondarySubnets != "" {
		body["secondarySubnets"] = parseCommaSeparated(secondarySubnets)
	}
	if listSubnetIDs != "" {
		body["listSubnetIds"] = parseCommaSeparated(listSubnetIDs)
	}
	if cmd.Flags().Changed("node-netmask-size") {
		nodeNetmaskSize, _ := cmd.Flags().GetInt("node-netmask-size")
		body["nodeNetmaskSize"] = nodeNetmaskSize
	}
	if autoUpgradeStr != "" {
		uc, err := cli.ParseStructFlag(autoUpgradeStr)
		if err != nil {
			return fmt.Errorf("--auto-upgrade-config: %w", err)
		}
		body["autoUpgradeConfig"] = uc
	}
	if autoHealingStr != "" {
		hc, err := cli.ParseStructFlagTyped(autoHealingStr, []string{"timeoutUnhealthy"}, []string{"enableAutoHealing"})
		if err != nil {
			return fmt.Errorf("--auto-healing-config: %w", err)
		}
		if enabled, _ := hc["enableAutoHealing"].(bool); enabled {
			_, hasMax := hc["maxUnhealthy"]
			_, hasRange := hc["unhealthyRange"]
			if hasMax == hasRange {
				return fmt.Errorf("--auto-healing-config: set exactly one of maxUnhealthy or unhealthyRange")
			}
		}
		body["autoHealingConfig"] = hc
	}

	// Network-type-specific requirements (enforced client-side so both dry-run
	// and real creates fail fast with a clear message).
	netErrs := validateNetworkRequirements(networkType, cidr, cmd.Flags().Changed("node-netmask-size"), secondarySubnets)

	if dryRun {
		return validateCreateCluster(name, netErrs)
	}
	if len(netErrs) > 0 {
		return fmt.Errorf("%s", strings.Join(netErrs, "; "))
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post("/v1/clusters", body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}

// validateNetworkRequirements checks the fields each network type requires.
// The API mandates --cidr for TIGERA/CILIUM_OVERLAY and both
// --node-netmask-size and at least one --secondary-subnets for
// CILIUM_NATIVE_ROUTING; validating here turns opaque server errors into
// actionable messages.
func validateNetworkRequirements(networkType, cidr string, nodeNetmaskSet bool, secondarySubnets string) []string {
	var errs []string
	switch networkType {
	case "TIGERA", "CILIUM_OVERLAY":
		if cidr == "" {
			errs = append(errs, fmt.Sprintf("--cidr is required when --network-type is %s", networkType))
		}
	case "CILIUM_NATIVE_ROUTING":
		if !nodeNetmaskSet {
			errs = append(errs, "--node-netmask-size is required when --network-type is CILIUM_NATIVE_ROUTING (allowed: 24, 25, 26)")
		}
		if secondarySubnets == "" {
			errs = append(errs, "--secondary-subnets is required when --network-type is CILIUM_NATIVE_ROUTING (at least one subnet)")
		}
	}
	return errs
}

func validateCreateCluster(name string, networkErrors []string) error {
	clusterNameRE := regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{3,18}[a-z0-9]$`)

	var errors []string

	if !clusterNameRE.MatchString(name) {
		errors = append(errors, fmt.Sprintf(
			"Cluster name '%s' is invalid. Must be 5-20 chars, lowercase alphanumeric and hyphens, start/end with alphanumeric.", name))
	}

	errors = append(errors, networkErrors...)

	fmt.Println("=== DRY RUN: Validation results ===")
	fmt.Println()
	if len(errors) > 0 {
		fmt.Printf("Found %d error(s):\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("All parameters are valid. Run without --dry-run to create the cluster.")
	fmt.Println()
	fmt.Println("Note: dry-run performs local checks only. Whether the --k8s-version is")
	fmt.Println("available on the selected --release-channel, that the VPC/subnets exist,")
	fmt.Println("and quota availability are validated by the server on the actual create.")
	return nil
}
