package server

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagKeyCmd = &cobra.Command{
	Use:   "tag-key",
	Short: "List all tag keys available in the project",
	Long: `List every tag key that exists in the current project.

Tag keys are used together with their values to label resources — for example
when creating an image with 'vserver server create-image --tag <key>=<value>'.
Use 'vserver server tag-value --key <key>' to see the values for a key.`,
	RunE: runTagKey,
}

func init() {
	// no flags — the endpoint returns the full set of tag keys
}

func runTagKey(cmd *cobra.Command, args []string) error {
	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/tag/tag-key", projectID), nil)
	if err != nil {
		return fmt.Errorf("failed to list tag keys: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
