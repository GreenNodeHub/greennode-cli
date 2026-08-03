package server

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagValueCmd = &cobra.Command{
	Use:   "tag-value",
	Short: "List the possible values for a tag key",
	Long: `List every value that has been used for a given tag key in the project.

Run 'vserver server tag-key' first to see the available keys.`,
	RunE: runTagValue,
}

func init() {
	f := tagValueCmd.Flags()
	f.String("key", "", "Tag key whose values to list (required)")

	if err := tagValueCmd.MarkFlagRequired("key"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "key", err))
	}
}

func runTagValue(cmd *cobra.Command, args []string) error {
	key, _ := cmd.Flags().GetString("key")
	if key == "" {
		return fmt.Errorf("--key is required")
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/tag/tag-key/%s/tag-value", projectID, key), nil)
	if err != nil {
		return fmt.Errorf("failed to list values for tag key %q: %w", key, err)
	}

	return outputResult(cmd, cfg, result)
}
