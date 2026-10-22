package vpc

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
	Short: "Update a VPC",
	RunE:  runUpdate,
}

func init() {
	f := updateCmd.Flags()
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("name", "", "New VPC name (required)")
	f.String("zone-id", "", "Availability zone ID")
	f.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	f.Bool("dry-run", false, "Validate the update without executing")
	for _, name := range []string{"vpc-id", "name"} {
		if err := updateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	body := map[string]any{"name": name, "tags": tags, "zoneId": nilIfEmpty(zoneID)}
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
	result, err := apiClient.Patch(fmt.Sprintf("/v2/%s/networks/%s", projectID, vpcID), body)
	if err != nil {
		return fmt.Errorf("failed to update VPC %s: %w", vpcID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
