package secgroup

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new security group",
	RunE:  runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Security group name (required)")
	f.String("description", "", "Security group description")
	createCmd.MarkFlagRequired("name")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}
	if err := validateSecgroupDescription(description); err != nil {
		return err
	}

	body := map[string]interface{}{
		"name":        name,
		"description": nilIfEmpty(description),
	}

	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		cli.PrintDryRun("POST", fmt.Sprintf("/v2/%s/secgroups", "<project-id>"), body)
		return nil
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/secgroups", projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create security group: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
