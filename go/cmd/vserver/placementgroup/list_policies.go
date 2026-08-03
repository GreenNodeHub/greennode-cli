package placementgroup

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listPoliciesCmd = &cobra.Command{
	Use:   "list-policies",
	Short: "List available placement group policies",
	Long: `List the placement group policies that can be used when creating a
placement group. The description is shown in English by default; pass
--language vi for the Vietnamese description.`,
	RunE: runListPolicies,
}

func init() {
	listPoliciesCmd.Flags().String("language", "en", "Description language: en or vi")
	listPoliciesCmd.RegisterFlagCompletionFunc("language", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) { //nolint:errcheck
		return []string{"en\tEnglish", "vi\tVietnamese"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func runListPolicies(cmd *cobra.Command, args []string) error {
	lang, _ := cmd.Flags().GetString("language")

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/serverGroups/policies", projectID), nil)
	if err != nil {
		return fmt.Errorf("failed to list placement group policies: %w", err)
	}

	return outputPolicies(cmd, cfg, result, lang)
}
