package member

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var tableColumns = []string{
	"uuid", "name", "address", "protocolPort", "weight", "monitorPort", "backup", "displayStatus",
}

func resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	return vlbclient.Resolve(cmd)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.OutputWithColumns(cmd, cfg, data, tableColumns)
}

// addIDFlags declares --load-balancer-id and --pool-id, required by every
// member subcommand.
func addIDFlags(c *cobra.Command) {
	c.Flags().String("load-balancer-id", "", "Load balancer ID (required)")
	c.Flags().String("pool-id", "", "Pool ID (required)")
	for _, name := range []string{"load-balancer-id", "pool-id"} {
		if err := c.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func idsOf(cmd *cobra.Command) (lbID, poolID string, err error) {
	lbID, _ = cmd.Flags().GetString("load-balancer-id")
	if err = validator.ValidateID(lbID, "load-balancer-id"); err != nil {
		return "", "", err
	}
	poolID, _ = cmd.Flags().GetString("pool-id")
	if err = validator.ValidateID(poolID, "pool-id"); err != nil {
		return "", "", err
	}
	return lbID, poolID, nil
}
