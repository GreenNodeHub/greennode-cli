package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the listeners of a load balancer",
	RunE:  runList,
}

func init() {
	addLBFlag(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	lbID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Listeners(projectID, lbID), nil)
	if err != nil {
		return fmt.Errorf("failed to list listeners of load balancer %s: %w", lbID, err)
	}

	return outputResult(cmd, cfg, result)
}
