package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of an L7 policy",
	RunE:  runGet,
}

func init() {
	addParentFlags(getCmd)
	addPolicyFlag(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	lbID, listenerID, policyID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Policy(projectID, lbID, listenerID, policyID), nil)
	if err != nil {
		return fmt.Errorf("failed to get policy %s: %w", policyID, err)
	}

	return outputResult(cmd, cfg, result)
}
