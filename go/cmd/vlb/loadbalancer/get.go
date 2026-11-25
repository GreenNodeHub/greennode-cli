package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a load balancer",
	RunE:  runGet,
}

func init() {
	addLBFlag(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.LoadBalancer(projectID, lbID), nil)
	if err != nil {
		return fmt.Errorf("failed to get load balancer %s: %w", lbID, err)
	}

	return outputResult(cmd, cfg, result)
}
