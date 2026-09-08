package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the L7 policies of a listener",
	RunE:  runList,
}

func init() {
	addParentFlags(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	lbID, listenerID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Policies(projectID, lbID, listenerID), nil)
	if err != nil {
		return fmt.Errorf("failed to list the policies of listener %s: %w", listenerID, err)
	}

	return outputResult(cmd, cfg, result)
}
