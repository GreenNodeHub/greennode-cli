package dhcp

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a DHCP option set",
	Long:  "Show the full details of a single DHCP option set by its ID.",
	RunE:  runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("dhcp-option-id", "", "DHCP option set ID (required)")
	getCmd.MarkFlagRequired("dhcp-option-id") //nolint:errcheck
}

func runGet(cmd *cobra.Command, args []string) error {
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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/dhcp_option/%s", projectID, dhcpOptionID), nil)
	if err != nil {
		return fmt.Errorf("failed to get DHCP option %s: %w", dhcpOptionID, err)
	}

	return outputDhcpDetail(cmd, cfg, result)
}
