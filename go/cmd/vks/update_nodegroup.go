package vks

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	f.Bool("disable-auto-scale", false, "Disable autoscaling (sends autoScaleConfig: null)")
	f.Bool("dry-run", false, "Preview update without executing")

	updateNodegroupCmd.MarkFlagRequired("cluster-id")
	updateNodegroupCmd.MarkFlagRequired("nodegroup-id")
	// Cobra shows the exclusion in --help and rejects the combination itself;
	// resolveAutoScaleConfig keeps its own check so the body composer stays
	// correct when called outside the command.
	updateNodegroupCmd.MarkFlagsMutuallyExclusive("auto-scale", "disable-auto-scale")
}

func runUpdateNodegroup(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}

	body, err := buildUpdateNodegroupBody(cmd.Flags())
	if err != nil {
		return err
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

// parseNumNodes parses --num-nodes strictly. UpdateNodeGroupDto.numNodes is an
// integer with minimum 0, and 0 is a valid request — so a silently swallowed
// parse error ("abc" -> 0) would scale the node group down to zero instead of
// failing.
func parseNumNodes(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("--num-nodes must be an integer, got %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("--num-nodes must be 0 or greater, got %d", n)
	}
	return n, nil
}

// defaultUpgradeStrategy is the only strategy the API supports today
// (NodeGroupUpgradeConfigDto.strategy).
const defaultUpgradeStrategy = "SURGE"

// upgradeConfigWithDefaults returns a copy of in with the fields the API needs
// filled in: strategy, which NodeGroupUpgradeConfigDto marks *required* (sending
// only maxSurge is a 400), plus maxSurge (default 1) and maxUnavailable
// (default 0). The server applies those two numeric defaults itself, so filling
// them client-side only makes --dry-run show the effective payload. in is not
// mutated.
func upgradeConfigWithDefaults(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in)+3)
	for k, v := range in {
		out[k] = v
	}
	if v, ok := out["strategy"]; !ok || v == nil || v == "" {
		out["strategy"] = defaultUpgradeStrategy
	}
	if v, ok := out["maxSurge"]; !ok || v == nil {
		out["maxSurge"] = 1
	}
	if v, ok := out["maxUnavailable"]; !ok || v == nil {
		out["maxUnavailable"] = 0
	}
	return out
}

// validateUpgradeConfigObject checks the bounds NodeGroupUpgradeConfigDto
// declares: maxSurge 1-100, maxUnavailable 0-100, strategy a non-empty string.
// Run it after upgradeConfigWithDefaults so the defaults are in place.
func validateUpgradeConfigObject(m map[string]interface{}) error {
	bounds := []struct {
		field    string
		min, max int
	}{
		{"maxSurge", 1, 100},
		{"maxUnavailable", 0, 100},
	}
	for _, b := range bounds {
		v, ok, err := integralField(m, b.field)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if v < b.min || v > b.max {
			return fmt.Errorf("%s must be between %d and %d, got %d", b.field, b.min, b.max, v)
		}
	}
	if s, ok := m["strategy"]; ok {
		if str, isStr := s.(string); !isStr || str == "" {
			return fmt.Errorf("strategy must be a non-empty string (the API currently supports only %q)", defaultUpgradeStrategy)
		}
	}
	return nil
}

// integralField reads m[field] as an int. ok reports whether the key is present.
// Shorthand parsing yields int; JSON numbers arrive as float64, so an integral
// float64 (5) is accepted while 2.5 and null are rejected.
func integralField(m map[string]interface{}, field string) (int, bool, error) {
	v, present := m[field]
	if !present {
		return 0, false, nil
	}
	switch n := v.(type) {
	case nil:
		return 0, true, fmt.Errorf("%s must be an integer, got null", field)
	case int:
		return n, true, nil
	case float64:
		if n != float64(int(n)) {
			return 0, true, fmt.Errorf("%s must be an integer, got %v", field, n)
		}
		return int(n), true, nil
	default:
		return 0, true, fmt.Errorf("%s must be an integer, got %T", field, v)
	}
}

// resolveAutoScaleConfig decides the autoScaleConfig field value for an
// update. The backend uses JsonNullable, so three states are meaningful:
// omit (keep current), null (disable), or object (set). --disable-auto-scale
// produces null; --auto-scale produces a validated object; both together is
// rejected. set=false means "do not send the field".
func resolveAutoScaleConfig(flags *pflag.FlagSet) (interface{}, bool, error) {
	autoScaleStr, _ := flags.GetString("auto-scale")
	disable, _ := flags.GetBool("disable-auto-scale")

	if autoScaleStr != "" && disable {
		return nil, false, fmt.Errorf("cannot use --auto-scale and --disable-auto-scale together")
	}
	if disable {
		// nil in a map[string]interface{} marshals to JSON null.
		return nil, true, nil
	}
	if autoScaleStr == "" {
		return nil, false, nil
	}

	asc, err := cli.ParseStructFlagTyped(autoScaleStr, []string{"minSize", "maxSize"}, nil)
	if err != nil {
		return nil, false, fmt.Errorf("--auto-scale: %w", err)
	}

	if err := validateAutoScaleObject(asc); err != nil {
		return nil, false, fmt.Errorf("--auto-scale: %w", err)
	}
	return asc, true, nil
}

// validateAutoScaleObject mirrors NodeGroupAutoScaleConfigDto: minSize and
// maxSize are both required and integral, minSize has minimum 0, maxSize has
// minimum 1, and an inverted range is rejected. A missing key is "both
// required"; a present-but-null key is "must be an integer" (a JsonNullable null
// is a wrong-type value here, not absence).
func validateAutoScaleObject(m map[string]interface{}) error {
	sizes := make(map[string]int, 2)
	for _, field := range []string{"minSize", "maxSize"} {
		v, ok, err := integralField(m, field)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("both minSize and maxSize are required when --auto-scale is an object")
		}
		sizes[field] = v
	}
	if sizes["minSize"] < 0 {
		return fmt.Errorf("minSize must be 0 or greater, got %d", sizes["minSize"])
	}
	if sizes["maxSize"] < 1 {
		return fmt.Errorf("maxSize must be 1 or greater, got %d", sizes["maxSize"])
	}
	if sizes["minSize"] > sizes["maxSize"] {
		return fmt.Errorf("minSize (%d) must not exceed maxSize (%d)", sizes["minSize"], sizes["maxSize"])
	}
	return nil
}

// buildUpdateNodegroupBody composes the PUT body from flags. It returns an
// error if nothing is set, or if any field fails validation. The empty-body
// guard runs last so validation errors surface first.
func buildUpdateNodegroupBody(flags *pflag.FlagSet) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	if numNodes, _ := flags.GetString("num-nodes"); numNodes != "" {
		n, err := parseNumNodes(numNodes)
		if err != nil {
			return nil, err
		}
		body["numNodes"] = n
	}
	if sg, _ := flags.GetString("security-groups"); sg != "" {
		body["securityGroups"] = parseCommaSeparated(sg)
	}

	asc, set, err := resolveAutoScaleConfig(flags)
	if err != nil {
		return nil, err
	}
	if set {
		body["autoScaleConfig"] = asc
	}

	if ucStr, _ := flags.GetString("upgrade-config"); ucStr != "" {
		uc, err := cli.ParseStructFlag(ucStr, "maxSurge", "maxUnavailable")
		if err != nil {
			return nil, fmt.Errorf("--upgrade-config: %w", err)
		}
		uc = upgradeConfigWithDefaults(uc)
		if err := validateUpgradeConfigObject(uc); err != nil {
			return nil, fmt.Errorf("--upgrade-config: %w", err)
		}
		body["upgradeConfig"] = uc
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("nothing to update: provide at least one of --num-nodes, --security-groups, --auto-scale, --disable-auto-scale, or --upgrade-config (use 'update-nodegroup-metadata' for labels/tags/taints)")
	}
	return body, nil
}
