package vpc

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a VPC",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("vpc-id", "", "VPC (network) ID (required)")
	getCmd.MarkFlagRequired("vpc-id")
}

func runGet(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/%s", projectID, vpcID), nil)
	if err != nil {
		return fmt.Errorf("failed to get VPC %s: %w", vpcID, err)
	}

	return outputResult(cmd, cfg, result)
}
