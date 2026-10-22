package subnet

import (
	"fmt"
	"net"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new subnet",
	RunE:  runCreate,
}

func init() {
	f := createCmd.Flags()

	f.String("vpc-id", "", "VPC (network) ID to create the subnet in (required)")
	f.String("cidr", "", "CIDR block for the subnet, e.g. 10.0.1.0/24 (required)")
	f.String("zone-id", "", "Availability zone ID — run without this flag to see available zones (required)")
	f.String("name", "", "Subnet name")

	for _, name := range []string{"vpc-id", "cidr"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	cidr, _ := cmd.Flags().GetString("cidr")
	zoneID, _ := cmd.Flags().GetString("zone-id")

	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
		return err
	}
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("--cidr must be a valid CIDR")
	}

	body := map[string]interface{}{
		"name":   name,
		"cidr":   cidr,
		"zoneId": zoneID,
	}

	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		if zoneID == "" {
			return fmt.Errorf("--zone-id is required for dry-run")
		}
		cli.PrintDryRun("POST", fmt.Sprintf("/v2/%s/networks/%s/subnets", "<project-id>", vpcID), body)
		return nil
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	if zoneID == "" {
		return suggestZones(apiClient, projectID)
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/networks/%s/subnets", projectID, vpcID), body)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
