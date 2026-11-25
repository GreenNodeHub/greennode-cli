package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var updateBandwidthCmd = &cobra.Command{
	Use:   "update-bandwidth",
	Short: "Update the bandwidth package of a load balancer",
	RunE:  runUpdateBandwidth,
}

func init() {
	addLBFlag(updateBandwidthCmd)
	f := updateBandwidthCmd.Flags()
	f.String("bandwidth-package-id", "", "Target bandwidth package ID (required)")
	f.Bool("dry-run", false, "Print the request body without applying the change")
	if err := updateBandwidthCmd.MarkFlagRequired("bandwidth-package-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "bandwidth-package-id", err))
	}
}

func runUpdateBandwidth(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	bwPackageID, _ := cmd.Flags().GetString("bandwidth-package-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	body := map[string]interface{}{"bandWidthPackageId": bwPackageID}
	if dryRun {
		cli.PrintDryRun("update", "bandwidth package of load balancer "+lbID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(vlbclient.LoadBalancerSub(projectID, lbID, "bandwidth"), body)
	if err != nil {
		return fmt.Errorf("failed to update bandwidth package of load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}
