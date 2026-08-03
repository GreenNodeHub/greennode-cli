package secgroup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a security group",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("secgroup-id", "", "Security group ID (required)")
	if err := getCmd.MarkFlagRequired("secgroup-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "secgroup-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")

	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/secgroups/%s", projectID, secgroupID), nil)
	if err != nil {
		return fmt.Errorf("failed to get security group %s: %w", secgroupID, err)
	}

	return outputResult(cmd, cfg, result)
}
