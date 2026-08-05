package dhcp

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listVpcsCmd = &cobra.Command{
	Use:   "list-vpcs",
	Short: "List the VPCs associated with a DHCP option set",
	Long:  "List every VPC (network) currently associated with a DHCP option set.",
	RunE:  runListVpcs,
}

func init() {
	f := listVpcsCmd.Flags()
	f.String("dhcp-option-id", "", "DHCP option set ID (required)")
	f.Int("page", 1, "Page number (1-based)")
	f.Int("page-size", 50, "Number of items per page")
	listVpcsCmd.MarkFlagRequired("dhcp-option-id") //nolint:errcheck
}

func runListVpcs(cmd *cobra.Command, args []string) error {
	dhcpOptionID, _ := cmd.Flags().GetString("dhcp-option-id")

	if err := validator.ValidateID(dhcpOptionID, "dhcp-option-id"); err != nil {
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

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	params := map[string]string{
		"dhcpOptionIds": dhcpOptionID,
		"page":          fmt.Sprintf("%d", page),
		"size":          fmt.Sprintf("%d", pageSize),
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks", projectID), params)
	if err != nil {
		return fmt.Errorf("failed to list VPCs associated with DHCP option %s: %w", dhcpOptionID, err)
	}

	return outputVpcList(cmd, cfg, result)
}
