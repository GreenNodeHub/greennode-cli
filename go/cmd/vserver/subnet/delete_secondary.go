package subnet

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var deleteSecondaryCmd = &cobra.Command{
	Use:   "delete-secondary",
	Short: "Delete a secondary subnet",
	RunE:  runDeleteSecondary,
}

func init() {
	f := deleteSecondaryCmd.Flags()
	f.String("vpc-id", "", "VPC ID (required)")
	f.String("subnet-id", "", "Primary subnet ID (required)")
	f.String("secondary-subnet-id", "", "Secondary subnet ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the deletion without executing")
	for _, name := range []string{"vpc-id", "subnet-id", "secondary-subnet-id"} {
		if err := deleteSecondaryCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runDeleteSecondary(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	secondaryID, _ := cmd.Flags().GetString("secondary-subnet-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{
		{vpcID, "vpc-id"}, {subnetID, "subnet-id"}, {secondaryID, "secondary-subnet-id"},
	} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("Delete secondary subnet %s?", secondaryID)) {
		return cli.ConfirmationError()
	}

	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s/secondary-subnets/%s", projectID, vpcID, subnetID, secondaryID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete secondary subnet %s: %w", secondaryID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
