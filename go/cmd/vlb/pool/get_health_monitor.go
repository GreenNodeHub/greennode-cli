package pool

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var healthMonitorColumns = []string{
	"uuid", "healthCheckProtocol", "healthCheckPath", "healthCheckMethod",
	"interval", "timeout", "healthyThreshold", "unhealthyThreshold", "displayStatus",
}

var getHealthMonitorCmd = &cobra.Command{
	Use:   "get-health-monitor",
	Short: "Get the health monitor of a pool",
	RunE:  runGetHealthMonitor,
}

func init() {
	addLBFlag(getHealthMonitorCmd)
	addPoolFlag(getHealthMonitorCmd)
}

func runGetHealthMonitor(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.HealthMonitor(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to get health monitor of pool %s: %w", poolID, err)
	}

	return vlbclient.OutputWithColumns(cmd, cfg, result, healthMonitorColumns)
}
