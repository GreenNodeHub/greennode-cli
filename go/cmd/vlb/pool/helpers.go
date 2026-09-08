package pool

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var tableColumns = []string{
	"uuid", "name", "protocol", "loadBalanceMethod", "stickiness", "tlsEncryption", "displayStatus",
}

// healthMonitorIntFields / healthMonitorFields describe the --health-monitor
// struct flag; they mirror CreateHMonitorRequest / UpdateHMonitorRequest.
var healthMonitorIntFields = []string{"healthyThreshold", "unhealthyThreshold", "timeout", "interval"}

func resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	return vlbclient.Resolve(cmd)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.OutputWithColumns(cmd, cfg, data, tableColumns)
}

func outputRaw(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.Output(cmd, cfg, data)
}

// addLBFlag declares --load-balancer-id, required by every pool subcommand.
func addLBFlag(c *cobra.Command) {
	c.Flags().String("load-balancer-id", "", "Load balancer ID (required)")
	if err := c.MarkFlagRequired("load-balancer-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "load-balancer-id", err))
	}
}

// addPoolFlag declares --pool-id, required by the per-pool subcommands.
func addPoolFlag(c *cobra.Command) {
	c.Flags().String("pool-id", "", "Pool ID (required)")
	if err := c.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "pool-id", err))
	}
}

// idsOf reads and validates --load-balancer-id and --pool-id.
func idsOf(cmd *cobra.Command, withPool bool) (lbID, poolID string, err error) {
	lbID, _ = cmd.Flags().GetString("load-balancer-id")
	if err = validator.ValidateID(lbID, "load-balancer-id"); err != nil {
		return "", "", err
	}
	if !withPool {
		return lbID, "", nil
	}
	poolID, _ = cmd.Flags().GetString("pool-id")
	if err = validator.ValidateID(poolID, "pool-id"); err != nil {
		return "", "", err
	}
	return lbID, poolID, nil
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
