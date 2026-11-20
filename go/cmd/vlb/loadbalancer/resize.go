package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a load balancer to a different package",
	RunE:  runResize,
}

func init() {
	addLBFlag(resizeCmd)
	f := resizeCmd.Flags()
	f.String("package-id", "", "Target package ID — run 'grn vlb package list' to see options (required)")
	f.Bool("dry-run", false, "Print the request body without applying the resize")
	if err := resizeCmd.MarkFlagRequired("package-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "package-id", err))
	}
}

func runResize(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	packageID, _ := cmd.Flags().GetString("package-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	body := map[string]interface{}{"packageId": packageID}
	if dryRun {
		cli.PrintDryRun("resize", "load balancer "+lbID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(vlbclient.LoadBalancerSub(projectID, lbID, "resize"), body)
	if err != nil {
		return fmt.Errorf("failed to resize load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}
