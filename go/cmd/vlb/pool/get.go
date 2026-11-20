package pool

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a pool",
	RunE:  runGet,
}

func init() {
	addLBFlag(getCmd)
	addPoolFlag(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Pool(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to get pool %s: %w", poolID, err)
	}

	return outputResult(cmd, cfg, result)
}
