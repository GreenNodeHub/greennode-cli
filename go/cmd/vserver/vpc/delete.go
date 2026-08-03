package vpc

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a VPC",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("vpc-id", "", "VPC (network) ID (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the VPC deletion without executing")
	deleteCmd.MarkFlagRequired("vpc-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(vpcID, "vpc-id"); err != nil {
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

	response, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/%s", projectID, vpcID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch VPC %s: %w", vpcID, err)
	}

	vpcData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type from API: %T", response)
	}

	if err := printVpcDeletePreview(vpcData); err != nil {
		return err
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this VPC?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/networks/%s", projectID, vpcID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete VPC %s: %w", vpcID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printVpcDeletePreview(vpc interface{}) error {
	v, ok := vpc.(map[string]interface{})
	if !ok || v == nil {
		return fmt.Errorf("could not parse VPC details from API response (type: %T)", vpc)
	}

	fmt.Println("The following VPC will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:   %v\n", v["id"])
	fmt.Printf("  Name: %v\n", v["displayName"])
	fmt.Printf("  CIDR: %v\n", v["cidr"])
	fmt.Printf("  Status: %v\n", v["status"])
	fmt.Printf("  Dns Status: %v\n", v["dnsStatus"])
	fmt.Println()
	fmt.Println("This action is irreversible.")
	return nil
}
