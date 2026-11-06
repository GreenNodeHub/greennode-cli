package vks

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	f.String("network-type", "", "Network type: TIGERA, CILIUM_OVERLAY, CILIUM_NATIVE_ROUTING (required). TIGERA/CILIUM_OVERLAY need --cidr; CILIUM_NATIVE_ROUTING needs --node-netmask-size")
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("subnet-ids", "", "Subnet IDs for the cluster, comma-separated (required, at least one)")

	for _, name := range []string{"name", "k8s-version", "network-type", "vpc-id"} {
		createClusterCmd.MarkFlagRequired(name)
	}
	// --subnet-ids is required too, but it is enforced in resolveSubnetIDs rather
	// than with MarkFlagRequired: that would reject callers who pass only the
	// deprecated --list-subnet-ids alias.

	// Cluster settings (optional)
	f.String("cidr", "", "CIDR block (required for TIGERA and CILIUM_OVERLAY)")
	f.String("description", "", "Cluster description")
	f.String("private-cluster", "disabled", "Private cluster (enabled, disabled)")
	f.String("release-channel", "STABLE", "Release channel (RAPID, STABLE)")
	f.String("load-balancer-plugin", "enabled", "Load balancer plugin (enabled, disabled)")
	f.String("block-store-csi-plugin", "enabled", "Block store CSI plugin (enabled, disabled)")
	f.String("service-endpoint", "disabled", "Service endpoint (enabled, disabled)")
	f.String("az-strategy", "SINGLE", "Availability zone strategy: SINGLE (exactly one --subnet-ids value) or MULTI")
	f.String("list-subnet-ids", "", "Subnet IDs for the cluster (comma-separated)")
	// Deprecating hides the alias from help and prints a warning when it is used.
	_ = f.MarkDeprecated("list-subnet-ids", "use --subnet-ids instead")
	f.Int("node-netmask-size", 0, "Node netmask size: 24, 25, or 26 (required for CILIUM_NATIVE_ROUTING)")
	f.String("auto-upgrade-config", "", "Auto-upgrade config (shorthand time=03:00,weekdays=Mon or JSON; use JSON for multiple weekdays)")
	f.String("auto-healing-config", "", "Auto-healing config; set exactly one of maxUnhealthy or unhealthyRange (shorthand enableAutoHealing=true,maxUnhealthy=20%,timeoutUnhealthy=10 or JSON)")
	f.Bool("dry-run", false, "Validate parameters without creating the cluster")
}

func runCreateCluster(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	networkType, _ := cmd.Flags().GetString("network-type")
	cidr, _ := cmd.Flags().GetString("cidr")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	body, err := buildCreateClusterBody(cmd.Flags())
	if err != nil {
		return err
	}

	// Field requirements enforced client-side so both dry-run and real creates
	// fail fast with a clear message instead of an opaque server error.
	netmaskSet := cmd.Flags().Changed("node-netmask-size")
	nodeNetmaskSize, _ := cmd.Flags().GetInt("node-netmask-size")
	azStrategy, _ := body["azStrategy"].(string)
	subnetIDs, _ := body["listSubnetIds"].([]string)

	netErrs := validateNetworkRequirements(networkType, cidr, netmaskSet)
	netErrs = append(netErrs, validateNodeNetmaskSize(netmaskSet, nodeNetmaskSize)...)
	netErrs = append(netErrs, validateAZStrategy(azStrategy, subnetIDs)...)

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

// buildCreateClusterBody composes the POST body from flags. Every key it sets
// exists on CreateClusterComboDto, the schema POST /v1/clusters accepts. The
// deprecated single-subnet "subnetId" is never sent: one subnet and many subnets
// both travel in "listSubnetIds".
func buildCreateClusterBody(flags *pflag.FlagSet) (map[string]interface{}, error) {
	name, _ := flags.GetString("name")
	k8sVersion, _ := flags.GetString("k8s-version")
	networkType, _ := flags.GetString("network-type")
	vpcID, _ := flags.GetString("vpc-id")
	cidr, _ := flags.GetString("cidr")
	description, _ := flags.GetString("description")
	releaseChannel, _ := flags.GetString("release-channel")
	azStrategy, _ := flags.GetString("az-strategy")
	autoUpgradeStr, _ := flags.GetString("auto-upgrade-config")
	autoHealingStr, _ := flags.GetString("auto-healing-config")

	// Parse enabled/disabled toggle flags.
	privateClusterVal, _ := flags.GetString("private-cluster")
	lbPluginVal, _ := flags.GetString("load-balancer-plugin")
	csiPluginVal, _ := flags.GetString("block-store-csi-plugin")
	serviceEndpointVal, _ := flags.GetString("service-endpoint")
	enablePrivateCluster, err := parseToggle("private-cluster", privateClusterVal)
	if err != nil {
		return nil, err
	}
	enabledLBPlugin, err := parseToggle("load-balancer-plugin", lbPluginVal)
	if err != nil {
		return nil, err
	}
	enabledCSIPlugin, err := parseToggle("block-store-csi-plugin", csiPluginVal)
	if err != nil {
		return nil, err
	}
	enabledServiceEndpoint, err := parseToggle("service-endpoint", serviceEndpointVal)
	if err != nil {
		return nil, err
	}

	subnetIDs, err := resolveSubnetIDs(flags)
	if err != nil {
		return nil, err
	}

	// Node groups are created separately via 'grn vks create-nodegroup'.
	body := map[string]interface{}{
		"name":                       name,
		"version":                    k8sVersion,
		"networkType":                networkType,
		"vpcId":                      vpcID,
		"listSubnetIds":              subnetIDs,
		"enablePrivateCluster":       enablePrivateCluster,
		"releaseChannel":             releaseChannel,
		"enabledBlockStoreCsiPlugin": enabledCSIPlugin,
		"enabledLoadBalancerPlugin":  enabledLBPlugin,
		"enabledServiceEndpoint":     enabledServiceEndpoint,
		"azStrategy":                 azStrategy,
	}

	if cidr != "" {
		body["cidr"] = cidr
	}
	if description != "" {
		body["description"] = description
	}
	if flags.Changed("node-netmask-size") {
		nodeNetmaskSize, _ := flags.GetInt("node-netmask-size")
		body["nodeNetmaskSize"] = nodeNetmaskSize
	}
	if autoUpgradeStr != "" {
		uc, err := cli.ParseStructFlag(autoUpgradeStr)
		if err != nil {
			return nil, fmt.Errorf("--auto-upgrade-config: %w", err)
		}
		body["autoUpgradeConfig"] = uc
	}
	if autoHealingStr != "" {
		hc, err := cli.ParseStructFlagTyped(autoHealingStr, []string{"timeoutUnhealthy"}, []string{"enableAutoHealing"})
		if err != nil {
			return nil, fmt.Errorf("--auto-healing-config: %w", err)
		}
		if enabled, _ := hc["enableAutoHealing"].(bool); enabled {
			_, hasMax := hc["maxUnhealthy"]
			_, hasRange := hc["unhealthyRange"]
			if hasMax == hasRange {
				return nil, fmt.Errorf("--auto-healing-config: set exactly one of maxUnhealthy or unhealthyRange")
			}
		}
		body["autoHealingConfig"] = hc
	}

	return body, nil
}

// resolveSubnetIDs returns the cluster's subnet IDs. The API still accepts the
// single-subnet "subnetId" as a deprecated fallback, but the CLI stops sending it
// ahead of its removal: one subnet and many subnets both travel in
// "listSubnetIds" — hence a single required list flag. The deprecated
// --list-subnet-ids alias is still accepted, but not together with --subnet-ids,
// where there would be no safe way to guess which one the caller meant.
func resolveSubnetIDs(flags *pflag.FlagSet) ([]string, error) {
	subnetIDs, _ := flags.GetString("subnet-ids")
	deprecated, _ := flags.GetString("list-subnet-ids")

	if subnetIDs != "" && deprecated != "" {
		return nil, fmt.Errorf("--subnet-ids and --list-subnet-ids set the same field; pass only --subnet-ids")
	}
	if deprecated != "" {
		subnetIDs = deprecated
	}

	ids := parseCommaSeparated(subnetIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("--subnet-ids is required (at least one subnet ID, comma-separated)")
	}
	return ids, nil
}

// validateNetworkRequirements checks the fields each network type requires.
// The API mandates --cidr for TIGERA/CILIUM_OVERLAY and
// --node-netmask-size for CILIUM_NATIVE_ROUTING; validating here turns opaque
// server errors into actionable messages.
func validateNetworkRequirements(networkType, cidr string, nodeNetmaskSet bool) []string {
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
	}
	return errs
}

// validateNodeNetmaskSize checks the range CreateClusterDto.nodeNetmaskSize
// allows (24-26). Only checked when the flag was explicitly set, since the field
// is otherwise omitted and the server applies its own default.
func validateNodeNetmaskSize(nodeNetmaskSet bool, size int) []string {
	if nodeNetmaskSet && (size < 24 || size > 26) {
		return []string{fmt.Sprintf("--node-netmask-size must be 24, 25, or 26, got %d", size)}
	}
	return nil
}

// validateAZStrategy checks --az-strategy against the subnet list. The API allows
// SINGLE or MULTI, and documents listSubnetIds as "a single-element list for
// SINGLE" — so several subnets under the default SINGLE strategy is caught here
// instead of coming back as an opaque server rejection.
func validateAZStrategy(azStrategy string, subnetIDs []string) []string {
	switch azStrategy {
	case "SINGLE":
		if len(subnetIDs) > 1 {
			return []string{fmt.Sprintf(
				"--az-strategy SINGLE takes exactly one --subnet-ids value, got %d; use --az-strategy MULTI for a multi-subnet cluster",
				len(subnetIDs))}
		}
	case "MULTI":
	default:
		return []string{fmt.Sprintf("--az-strategy must be SINGLE or MULTI, got %q", azStrategy)}
	}
	return nil
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
