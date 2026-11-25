package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCloneMetadataCmd = &cobra.Command{
	Use:   "get-clone-metadata",
	Short: "Get the clone metadata of a load balancer",
	Long:  "Show the suggested name and the per-listener certificates a clone of this load balancer needs.",
	RunE:  runGetCloneMetadata,
}

func init() {
	addLBFlag(getCloneMetadataCmd)
}

func runGetCloneMetadata(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.LoadBalancerSub(projectID, lbID, "cloneMetadata"), nil)
	if err != nil {
		return fmt.Errorf("failed to get clone metadata of load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}
