package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var scaleHistoryColumns = []string{"id", "eventType", "adjustmentNumber", "total", "scaledAt"}

var listScaleHistoryCmd = &cobra.Command{
	Use:   "list-scale-history",
	Short: "List the scale history of a load balancer",
	RunE:  runListScaleHistory,
}

func init() {
	addLBFlag(listScaleHistoryCmd)
}

func runListScaleHistory(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.LoadBalancerSub(projectID, lbID, "scale-history"), nil)
	if err != nil {
		return fmt.Errorf("failed to list scale history of load balancer %s: %w", lbID, err)
	}

	return vlbclient.OutputWithColumns(cmd, cfg, result, scaleHistoryColumns)
}
