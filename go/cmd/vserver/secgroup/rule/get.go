package rule

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a security group rule",
	RunE:  runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("secgroup-id", "", "Security group ID (required)")
	f.String("rule-id", "", "Security group rule ID (required)")

	for _, name := range []string{"secgroup-id", "rule-id"} {
		if err := getCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	ruleID, _ := cmd.Flags().GetString("rule-id")

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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/secgroups/%s/secgroupRules/%s", projectID, secgroupID, ruleID), nil)
	if err != nil {
		return fmt.Errorf("failed to get rule %s: %w", ruleID, err)
	}

	return outputResult(cmd, cfg, result)
}
