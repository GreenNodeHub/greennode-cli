package subnet

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a subnet",
	RunE:  runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.String("subnet-id", "", "Subnet ID (required)")
	f.String("vpc-id", "", "VPC (network) ID the subnet belongs to (required)")
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview the subnet deletion without executing")
	deleteCmd.MarkFlagRequired("subnet-id")
	deleteCmd.MarkFlagRequired("vpc-id")
}

func runDelete(cmd *cobra.Command, args []string) error {
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	vpcID, _ := cmd.Flags().GetString("vpc-id")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(subnetID, "subnet-id"); err != nil {
		return err
	}
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

	response, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s", projectID, vpcID, subnetID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch subnet %s: %w", subnetID, err)
	}

	subnetData, ok := response.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response type from API: %T", response)
	}

	if err := printSubnetDeletePreview(subnetData); err != nil {
		return err
	}

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this subnet?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(fmt.Sprintf("/v2/%s/networks/%s/subnets/%s", projectID, vpcID, subnetID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete subnet %s: %w", subnetID, err)
	}

	return outputResult(cmd, cfg, result)
}

func printSubnetDeletePreview(subnet interface{}) error {
	s, ok := subnet.(map[string]interface{})
	if !ok || s == nil {
		return fmt.Errorf("could not parse subnet details from API response (type: %T)", subnet)
	}

	fmt.Println("The following subnet will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:   %v\n", s["uuid"])
	fmt.Printf("  Name: %v\n", s["name"])
	fmt.Printf("  CIDR: %v\n", s["cidr"])
	fmt.Println()
	fmt.Println("This action is irreversible.")
	return nil
}
