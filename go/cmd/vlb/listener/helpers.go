package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// defaultPoolName, not defaultPoolId: the service does populate the name when a
// default pool is attached (verified live — an earlier note here claimed it was
// always null, which was wrong), and a name reads better than a UUID. It is null
// only when the listener genuinely has no default pool. Scripts wanting the id
// can still reach it with --query, and `listener delete` prints the name too.
var tableColumns = []string{
	"uuid", "name", "protocol", "protocolPort", "defaultPoolName",
	"defaultAction", "displayStatus",
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

// addLBFlag declares --load-balancer-id, required by every listener subcommand.
func addLBFlag(c *cobra.Command) {
	c.Flags().String("load-balancer-id", "", "Load balancer ID (required)")
	if err := c.MarkFlagRequired("load-balancer-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "load-balancer-id", err))
	}
}

// addListenerFlag declares --listener-id, required by the per-listener subcommands.
func addListenerFlag(c *cobra.Command) {
	c.Flags().String("listener-id", "", "Listener ID (required)")
	if err := c.MarkFlagRequired("listener-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "listener-id", err))
	}
}

// idsOf reads and validates --load-balancer-id and, when requested, --listener-id.
func idsOf(cmd *cobra.Command, withListener bool) (lbID, listenerID string, err error) {
	lbID, _ = cmd.Flags().GetString("load-balancer-id")
	if err = validator.ValidateID(lbID, "load-balancer-id"); err != nil {
		return "", "", err
	}
	if !withListener {
		return lbID, "", nil
	}
	listenerID, _ = cmd.Flags().GetString("listener-id")
	if err = validator.ValidateID(listenerID, "listener-id"); err != nil {
		return "", "", err
	}
	return lbID, listenerID, nil
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
