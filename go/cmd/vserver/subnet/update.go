package subnet

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a subnet",
	RunE:  runUpdate,
}

func init() {
	f := updateCmd.Flags()
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("subnet-id", "", "Subnet ID (required)")
	f.String("name", "", "New subnet name (required)")
	f.String("zone-id", "", "Availability zone ID")
	f.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	f.Bool("dry-run", false, "Validate the update without executing")
	for _, name := range []string{"vpc-id", "subnet-id", "name"} {
		if err := updateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{{vpcID, "vpc-id"}, {subnetID, "subnet-id"}} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("update")
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
	body := map[string]any{"name": name, "tags": tags, "zoneId": nilIfEmpty(zoneID)}
	result, err := apiClient.Patch(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s", projectID, vpcID, subnetID), body)
	if err != nil {
		return fmt.Errorf("failed to update subnet %s: %w", subnetID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
