package subnet

import (
	"fmt"
	"net"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var createSecondaryCmd = &cobra.Command{
	Use:   "create-secondary",
	Short: "Create a secondary subnet",
	RunE:  runCreateSecondary,
}

func init() {
	f := createSecondaryCmd.Flags()
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("subnet-id", "", "Primary subnet ID (required)")
	f.String("name", "", "Secondary subnet name (required)")
	f.String("cidr", "", "Secondary subnet CIDR (required)")
	f.String("uuid", "", "Existing secondary subnet UUID, when required by the service")
	f.Bool("dry-run", false, "Validate parameters without creating the secondary subnet")
	for _, name := range []string{"vpc-id", "subnet-id", "name", "cidr"} {
		if err := createSecondaryCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runCreateSecondary(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	name, _ := cmd.Flags().GetString("name")
	cidr, _ := cmd.Flags().GetString("cidr")
	uuid, _ := cmd.Flags().GetString("uuid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{{vpcID, "vpc-id"}, {subnetID, "subnet-id"}} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return fmt.Errorf("invalid --cidr %q: %w", cidr, err)
	}
	if uuid != "" {
		if err := validator.ValidateID(uuid, "uuid"); err != nil {
			return err
		}
	}
	if dryRun {
		cli.DryRunNotice("create")
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
	body := map[string]any{"name": name, "cidr": cidr}
	if uuid != "" {
		body["uuid"] = uuid
	}
	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s/secondary-subnets", projectID, vpcID, subnetID), body)
	if err != nil {
		return fmt.Errorf("failed to create secondary subnet: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
