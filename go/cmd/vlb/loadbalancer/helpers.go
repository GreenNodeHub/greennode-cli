package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// tableColumns are the fields shown for `--output table`. JSON output is
// unaffected and always carries the full payload.
var tableColumns = []string{
	"uuid", "name", "displayStatus", "address", "displayType",
	"loadBalancerSchema", "privateSubnetId", "createdAt",
}

func resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	return vlbclient.Resolve(cmd)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.OutputWithColumns(cmd, cfg, data, tableColumns)
}

func outputRaw(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.Output(cmd, cfg, data)
}

// addLBFlag declares the --load-balancer-id flag shared by most subcommands.
func addLBFlag(c *cobra.Command) {
	c.Flags().String("load-balancer-id", "", "Load balancer ID (required)")
	if err := c.MarkFlagRequired("load-balancer-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "load-balancer-id", err))
	}
}

// lbIDOf reads and validates --load-balancer-id.
func lbIDOf(cmd *cobra.Command) (string, error) {
	id, _ := cmd.Flags().GetString("load-balancer-id")
	if err := validator.ValidateID(id, "load-balancer-id"); err != nil {
		return "", err
	}
	return id, nil
}
