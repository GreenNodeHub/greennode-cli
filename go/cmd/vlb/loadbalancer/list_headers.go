package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listHeadersCmd = &cobra.Command{
	Use:   "list-headers",
	Short: "List the headers a listener can insert",
	Long:  "Show the header names supported by 'grn vlb listener create --insert-header'.",
	RunE:  runListHeaders,
}

func runListHeaders(cmd *cobra.Command, args []string) error {
	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("%s/headers", vlbclient.LoadBalancers(projectID)), nil)
	if err != nil {
		return fmt.Errorf("failed to list headers: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}
