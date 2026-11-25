package member

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the members of a pool",
	RunE:  runList,
}

func init() {
	addIDFlags(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Members(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to list members of pool %s: %w", poolID, err)
	}

	return outputResult(cmd, cfg, result)
}
