package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var configureScalingCmd = &cobra.Command{
	Use:   "configure-scaling",
	Short: "Configure the HA scaling group of a load balancer",
	Long: "Set the instance count bounds and the subnets a load balancer's HA group spans " +
		"(the rebalancing API). --subnets must include the load balancer's primary subnet.",
	RunE: runConfigureScaling,
}

func init() {
	addLBFlag(configureScalingCmd)
	f := configureScalingCmd.Flags()
	f.Int("min-size", 0, "Minimum number of instances (omit to let the backend default it)")
	f.Int("max-size", 0, "Maximum number of instances (required)")
	f.String("subnets", "", "All active subnet IDs, comma-separated, including the primary one (required)")
	f.Bool("dry-run", false, "Print the request body without applying the change")
	for _, name := range []string{"max-size", "subnets"} {
		if err := configureScalingCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// buildScalingBody assembles the ScaleConfigRequest. minSize is only sent when
// the user set it explicitly — the backend treats an absent minSize as "keep
// the current value", which a zero would silently overwrite.
func buildScalingBody(minSize, maxSize int, minSizeSet bool, subnets string) (map[string]interface{}, error) {
	subnetIDs := cli.ParseCommaSeparated(subnets)
	if len(subnetIDs) == 0 {
		return nil, fmt.Errorf("--subnets must list at least one subnet ID")
	}
	if maxSize <= 0 {
		return nil, fmt.Errorf("--max-size must be greater than 0, got %d", maxSize)
	}
	scaling := map[string]interface{}{"maxSize": maxSize}
	if minSizeSet {
		scaling["minSize"] = minSize
	}
	return map[string]interface{}{
		"scaling":    scaling,
		"networking": map[string]interface{}{"subnets": subnetIDs},
	}, nil
}

func runConfigureScaling(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	minSize, _ := f.GetInt("min-size")
	maxSize, _ := f.GetInt("max-size")
	subnets, _ := f.GetString("subnets")
	dryRun, _ := f.GetBool("dry-run")

	body, err := buildScalingBody(minSize, maxSize, f.Changed("min-size"), subnets)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("configure", "scaling of load balancer "+lbID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(vlbclient.LoadBalancerSub(projectID, lbID, "rebalancing"), body)
	if err != nil {
		return fmt.Errorf("failed to configure scaling of load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}
