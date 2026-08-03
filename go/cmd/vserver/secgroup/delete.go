package secgroup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a security group",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("secgroup-id", "", "Security group ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the security group deletion without executing")
	deleteCmd.MarkFlagRequired("secgroup-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	secgroupID, _ := cmd.Flags().GetString("secgroup-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(secgroupID, "secgroup-id"); err != nil {
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

	response, err := apiClient.Get(fmt.Sprintf("/v2/%s/secgroups/%s", projectID, secgroupID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch security group %s: %w", secgroupID, err)
	}

	if err := printSecgroupDeletePreview(response); err != nil {
		return err
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this security group?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/secgroups/%s", projectID, secgroupID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete security group %s: %w", secgroupID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printSecgroupDeletePreview(sg interface{}) error {
	s, ok := sg.(map[string]interface{})
	if !ok || s == nil {
		return fmt.Errorf("could not parse security group details from API response (type: %T)", sg)
	}
	fmt.Println("The following security group will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:          %v\n", s["id"])
	fmt.Printf("  Name:        %v\n", s["name"])
	fmt.Printf("  Description: %v\n", s["description"])
	fmt.Println()
	fmt.Println("This action is irreversible.")
	return nil
}
