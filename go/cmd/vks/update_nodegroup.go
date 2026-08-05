package vks

import (
	"fmt"
	"os"

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

func toInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// upgradeConfigWithDefaults returns a copy of in with maxSurge and maxUnavailable
// filled to their defaults (1 and 0) when missing or nil. The backend applies
// these same defaults to omitted/null fields; filling client-side keeps the
// payload explicit and makes --dry-run show effective values. in is not mutated.
func upgradeConfigWithDefaults(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	if v, ok := out["maxSurge"]; !ok || v == nil {
		out["maxSurge"] = 1
	}
	if v, ok := out["maxUnavailable"]; !ok || v == nil {
		out["maxUnavailable"] = 0
	}
	return out
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

// validateAutoScaleObject requires both minSize and maxSize present, non-nil,
// and integral. A missing key is "both required"; a present-but-null key is
// "must be an integer" (JsonNullable null is a wrong-type value, not absence).
// Shorthand already coerces to int; JSON numbers arrive as float64, so
// integral float64 (e.g. 5) is accepted and 2.5 rejected.
func validateAutoScaleObject(m map[string]interface{}) error {
	for _, field := range []string{"minSize", "maxSize"} {
		v, ok := m[field]
		if !ok {
			return fmt.Errorf("both minSize and maxSize are required when --auto-scale is an object")
		}
		switch n := v.(type) {
		case nil:
			return fmt.Errorf("%s must be an integer, got null", field)
		case int:
			// shorthand path — fine
		case float64:
			if n != float64(int(n)) {
				return fmt.Errorf("%s must be an integer, got %v", field, n)
			}
		default:
			return fmt.Errorf("%s must be an integer, got %T", field, v)
		}
	}
	return nil
}

// buildUpdateNodegroupBody composes the PUT body from flags. It returns an
// error if nothing is set, or if any field fails validation. The empty-body
// guard runs last so validation errors surface first.
func buildUpdateNodegroupBody(flags *pflag.FlagSet) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	if numNodes, _ := flags.GetString("num-nodes"); numNodes != "" {
		body["numNodes"] = toInt(numNodes)
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
		body["upgradeConfig"] = upgradeConfigWithDefaults(uc)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("nothing to update: provide at least one of --num-nodes, --security-groups, --auto-scale, --disable-auto-scale, or --upgrade-config (use 'update-nodegroup-metadata' for labels/tags/taints)")
	}
	return body, nil
}
