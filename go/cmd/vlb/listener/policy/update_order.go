package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var updateOrderCmd = &cobra.Command{
	Use:   "update-order",
	Short: "Reorder the L7 policies of a listener",
	Long: "Set the evaluation order of a listener's L7 policies. Positions start " +
		"at 1; pass one --policy per policy being moved.",
	RunE: runUpdateOrder,
}

func init() {
	addParentFlags(updateOrderCmd)
	f := updateOrderCmd.Flags()
	f.StringArray("policy", nil,
		"Policy placement, shorthand (policyId=<id>,position=1) or JSON (repeatable, required); positions start at 1")
	f.Bool("dry-run", false, "Print the request body without applying the new order")
	if err := updateOrderCmd.MarkFlagRequired("policy"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "policy", err))
	}
}

// buildUpdateOrderBody assembles the ReorderPoliciesRequest.
//
// Positions are rejected below 1. The spec documents `position` as zero-based,
// but the service reports its own policy positions starting at 1, and a reorder
// sent with position=0 was accepted with HTTP 200, changed no position, and left
// the load balancer in UPDATING for ~30 minutes — refusing further changes and
// even deletion ("400 ... is updating") — before recovering on its own.
// Observed once against HCM-3 on 2026-09-08, so the causal link is not proven;
// but 0 contradicts every position the API itself returns, and the observed cost
// of sending it is half an hour of an unusable load balancer.
func buildUpdateOrderBody(policies []string) (map[string]interface{}, error) {
	parsed, err := cli.ParseStructFlags("policy", policies, []string{"position"}, nil)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("--policy must list at least one policy")
	}
	for i, p := range parsed {
		if _, ok := p["policyId"]; !ok {
			return nil, fmt.Errorf("--policy[%d]: policyId is required", i)
		}
		pos, ok := p["position"]
		if !ok {
			return nil, fmt.Errorf("--policy[%d]: position is required", i)
		}
		n, ok := pos.(int)
		if !ok {
			// JSON input decodes numbers as float64.
			f, isFloat := pos.(float64)
			if !isFloat {
				return nil, fmt.Errorf("--policy[%d]: position must be a number, got %v", i, pos)
			}
			n = int(f)
			p["position"] = n
		}
		if n < 1 {
			return nil, fmt.Errorf("--policy[%d]: position must be 1 or greater "+
				"(the API numbers policies from 1; sending 0 has been observed to "+
				"leave the load balancer stuck in UPDATING for ~30 minutes), got %d", i, n)
		}
	}
	return map[string]interface{}{"policies": parsed}, nil
}

func runUpdateOrder(cmd *cobra.Command, args []string) error {
	lbID, listenerID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}
	policies, _ := cmd.Flags().GetStringArray("policy")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	body, err := buildUpdateOrderBody(policies)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("update", "policy order of listener "+listenerID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(vlbclient.ReorderPolicies(projectID, lbID, listenerID), body)
	if err != nil {
		return fmt.Errorf("failed to reorder the policies of listener %s: %w", listenerID, err)
	}

	return outputRaw(cmd, cfg, result)
}
