package secgroup

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{Use: "update", Short: "Update a security group", RunE: runUpdate}

func init() {
	f := updateCmd.Flags()
	f.String("secgroup-id", "", "Security group ID (required)")
	f.String("name", "", "New security group name (required)")
	f.String("description", "", "Security group description")
	f.String("zone-id", "", "Availability zone ID")
	f.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	f.Bool("dry-run", false, "Validate the update without executing")
	for _, name := range []string{"secgroup-id", "name"} {
		if err := updateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	if err := validateSecgroupDescription(description); err != nil {
		return err
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
	body := map[string]any{"name": name, "description": nilIfEmpty(description), "zoneId": nilIfEmpty(zoneID), "tags": tags}
	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/secgroups/%s", projectID, secgroupID), body)
	if err != nil {
		return fmt.Errorf("failed to update security group %s: %w", secgroupID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
