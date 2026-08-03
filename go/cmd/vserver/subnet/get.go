package subnet

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a subnet",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("subnet-id", "", "Subnet ID (required)")
	getCmd.Flags().String("vpc-id", "", "VPC (network) ID the subnet belongs to (required)")
	getCmd.MarkFlagRequired("subnet-id")
	getCmd.MarkFlagRequired("vpc-id")
}

func runGet(cmd *cobra.Command, args []string) error {
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	vpcID, _ := cmd.Flags().GetString("vpc-id")

	if err := validator.ValidateID(subnetID, "subnet-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s", projectID, vpcID, subnetID), nil)
	if err != nil {
		return fmt.Errorf("failed to get subnet %s: %w", subnetID, err)
	}

	return outputResult(cmd, cfg, result)
}
