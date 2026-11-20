package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a listener",
	RunE:  runGet,
}

func init() {
	addLBFlag(getCmd)
	addListenerFlag(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	lbID, listenerID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Listener(projectID, lbID, listenerID), nil)
	if err != nil {
		return fmt.Errorf("failed to get listener %s: %w", listenerID, err)
	}

	return outputResult(cmd, cfg, result)
}
