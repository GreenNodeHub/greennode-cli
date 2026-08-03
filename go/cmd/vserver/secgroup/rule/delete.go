package rule

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule from a security group",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("secgroup-id", "", "Security group ID (required)")
	f.String("rule-id", "", "Security group rule ID (required)")
	f.Bool("dry-run", false, "Preview the rule deletion without executing")
	f.Bool("force", false, "Skip confirmation prompt")
	deleteCmd.MarkFlagRequired("secgroup-id")
	deleteCmd.MarkFlagRequired("rule-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	ruleID, _ := cmd.Flags().GetString("rule-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(ruleID, "rule-id"); err != nil {
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

	fmt.Println("The following security group rule will be deleted:")
	fmt.Println()
	fmt.Printf("  Rule ID:     %s\n", ruleID)
	fmt.Printf("  Secgroup ID: %s\n", secgroupID)
	fmt.Println()
	fmt.Println("This action is irreversible.")

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this rule?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(
		fmt.Sprintf("/v2/%s/secgroups/%s/secgroupRules/%s", projectID, secgroupID, ruleID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete rule %s from security group %s: %w", ruleID, secgroupID, err)
	}

	return outputResult(cmd, cfg, result)
}
