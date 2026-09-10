package rule

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listSamplesCmd = &cobra.Command{Use: "list-samples", Short: "List security-group rule samples", RunE: runListSamples}

func init() {
	listSamplesCmd.Flags().String("secgroup-id", "", "Security group ID (required)")
	if err := listSamplesCmd.MarkFlagRequired("secgroup-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "secgroup-id", err))
	}
}

func runListSamples(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
		return err
	}
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/secgroups/%s/secgroupRules/samples", projectID, secgroupID), nil)
	if err != nil {
		return fmt.Errorf("failed to list rule samples for security group %s: %w", secgroupID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
