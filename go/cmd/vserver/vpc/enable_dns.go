package vpc

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var enableDNSCmd = &cobra.Command{
	Use:   "enable-dns",
	Short: "Enable DNS for a VPC",
	RunE:  runEnableDNS,
}

func init() {
	enableDNSCmd.Flags().String("vpc-id", "", "VPC ID (required)")
	enableDNSCmd.Flags().Bool("dry-run", false, "Preview the DNS change without executing")
	if err := enableDNSCmd.MarkFlagRequired("vpc-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "vpc-id", err))
	}
}

func runEnableDNS(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("enable DNS")
		return nil
	}

	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Patch(fmt.Sprintf("/v2/%s/networks/%s/enableDns", projectID, vpcID), nil)
	if err != nil {
		return fmt.Errorf("failed to enable DNS for VPC %s: %w", vpcID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
