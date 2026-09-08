package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listBySubnetCmd = &cobra.Command{
	Use:   "list-by-subnet",
	Short: "List the load balancers attached to a subnet",
	RunE:  runListBySubnet,
}

func init() {
	f := listBySubnetCmd.Flags()
	f.String("subnet-id", "", "Subnet ID (required)")
	if err := listBySubnetCmd.MarkFlagRequired("subnet-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "subnet-id", err))
	}
}

func runListBySubnet(cmd *cobra.Command, args []string) error {
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	if err := validator.ValidateID(subnetID, "subnet-id"); err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("%s/subnet/%s", vlbclient.LoadBalancers(projectID), subnetID), nil)
	if err != nil {
		return fmt.Errorf("failed to list load balancers in subnet %s: %w", subnetID, err)
	}

	return outputResult(cmd, cfg, result)
}
