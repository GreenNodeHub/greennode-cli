package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var tableColumns = []string{
	"uuid", "name", "action", "position", "redirectPoolName", "redirectUrl",
	"redirectHttpCode", "displayStatus",
}

// ruleIntFields is empty — L7RuleRequest has no integer fields — but the
// constant keeps the ParseStructFlags call sites uniform.
var ruleIntFields []string

func resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	return vlbclient.Resolve(cmd)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.OutputWithColumns(cmd, cfg, data, tableColumns)
}

func outputRaw(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.Output(cmd, cfg, data)
}

// addParentFlags declares --load-balancer-id and --listener-id, required by
// every policy subcommand.
func addParentFlags(c *cobra.Command) {
	c.Flags().String("load-balancer-id", "", "Load balancer ID (required)")
	c.Flags().String("listener-id", "", "Listener ID (required)")
	for _, name := range []string{"load-balancer-id", "listener-id"} {
		if err := c.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// addPolicyFlag declares --policy-id, required by the per-policy subcommands.
func addPolicyFlag(c *cobra.Command) {
	c.Flags().String("policy-id", "", "L7 policy ID (required)")
	if err := c.MarkFlagRequired("policy-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "policy-id", err))
	}
}

// idsOf reads and validates --load-balancer-id and --listener-id, plus
// --policy-id when requested.
func idsOf(cmd *cobra.Command, withPolicy bool) (lbID, listenerID, policyID string, err error) {
	lbID, _ = cmd.Flags().GetString("load-balancer-id")
	if err = validator.ValidateID(lbID, "load-balancer-id"); err != nil {
		return "", "", "", err
	}
	listenerID, _ = cmd.Flags().GetString("listener-id")
	if err = validator.ValidateID(listenerID, "listener-id"); err != nil {
		return "", "", "", err
	}
	if !withPolicy {
		return lbID, listenerID, "", nil
	}
	policyID, _ = cmd.Flags().GetString("policy-id")
	if err = validator.ValidateID(policyID, "policy-id"); err != nil {
		return "", "", "", err
	}
	return lbID, listenerID, policyID, nil
}

// unwrapData returns the payload of a {"data": ...} envelope, or the response
// itself when it is already the bare object.
func unwrapData(response interface{}) map[string]interface{} {
	m, ok := response.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		return data
	}
	return m
}
