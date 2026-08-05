package dhcp

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var associateVpcCmd = &cobra.Command{
	Use:   "associate-vpc",
	Short: "Associate a VPC with a DHCP option set (or detach it)",
	Long: `Associate a VPC (network) with a DHCP option set, or detach it.

A VPC can belong to only one DHCP option set. Provide --dhcp-option-id to
associate the VPC with that set (replacing any current association). Use
--detach to remove the VPC's association entirely (the VPC then uses no custom
DHCP option set).`,
	RunE: runAssociateVpc,
}

func init() {
	f := associateVpcCmd.Flags()
	f.String("vpc-id", "", "VPC (network) ID to update (required)")
	f.String("dhcp-option-id", "", "DHCP option set ID to associate the VPC with")
	f.Bool("detach", false, "Detach the VPC from its current DHCP option set")
	associateVpcCmd.MarkFlagRequired("vpc-id") //nolint:errcheck
}

func runAssociateVpc(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	dhcpOptionID, _ := cmd.Flags().GetString("dhcp-option-id")
	detach, _ := cmd.Flags().GetBool("detach")

	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
		return err
	}

	if detach && dhcpOptionID != "" {
		return fmt.Errorf("--detach cannot be combined with --dhcp-option-id")
	}
	if !detach && dhcpOptionID == "" {
		return fmt.Errorf("either --dhcp-option-id (to associate) or --detach (to remove) is required")
	}

	// An empty body removes the association; a body with dhcpOptionId sets it.
	body := map[string]interface{}{}
	if !detach {
		if err := validator.ValidateID(dhcpOptionID, "dhcp-option-id"); err != nil {
			return err
		}
		body["dhcpOptionId"] = dhcpOptionID
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Patch(fmt.Sprintf("/v2/%s/networks/%s/updateDhcpOption", projectID, vpcID), body)
	if err != nil {
		return fmt.Errorf("failed to update DHCP option for VPC %s: %w", vpcID, err)
	}

	return outputResult(cmd, cfg, result)
}
