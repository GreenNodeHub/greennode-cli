package rule

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{Use: "update", Short: "Update a security-group rule", RunE: runUpdate}

func init() {
	f := updateCmd.Flags()
	f.String("secgroup-id", "", "Security group ID (required)")
	f.String("rule-id", "", "Security group rule ID (required)")
	f.String("description", "", "Rule description")
	f.String("zone-id", "", "Availability zone ID")
	f.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	f.Bool("dry-run", false, "Validate the update without executing")
	for _, name := range []string{"secgroup-id", "rule-id"} {
		if err := updateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	ruleID, _ := cmd.Flags().GetString("rule-id")
	description, _ := cmd.Flags().GetString("description")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{{secgroupID, "secgroup-id"}, {ruleID, "rule-id"}} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if description == "" && zoneID == "" && len(tags) == 0 {
		return fmt.Errorf("at least one of --description, --zone-id, or --tag is required")
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
	body := map[string]any{"description": nilIfEmpty(description), "zoneId": nilIfEmpty(zoneID), "tags": tags}
	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/secgroups/%s/secgroupRules/%s", projectID, secgroupID, ruleID), body)
	if err != nil {
		return fmt.Errorf("failed to update security group rule %s: %w", ruleID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
