package certificate

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a certificate",
	RunE:  runDelete,
}

func init() {
	addCertFlag(deleteCmd)
	f := deleteCmd.Flags()
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
}

func runDelete(cmd *cobra.Command, args []string) error {
	certID, err := certIDOf(cmd)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(vlbclient.Certificate(projectID, certID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch certificate %s: %w", certID, err)
	}
	printDeletePreview(current)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this certificate?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(vlbclient.Certificate(projectID, certID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete certificate %s: %w", certID, err)
	}

	return outputRaw(cmd, cfg, result)
}

func printDeletePreview(response interface{}) {
	c := unwrapData(response)
	fmt.Println("The following certificate will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", c["uuid"])
	fmt.Printf("  Name:    %v\n", c["name"])
	fmt.Printf("  Type:    %v\n", c["certificateType"])
	fmt.Printf("  Domain:  %v\n", c["domainName"])
	fmt.Printf("  Expires: %v\n", c["expiredAt"])
	fmt.Println()
	if inUse, _ := c["inUse"].(bool); inUse {
		fmt.Println("WARNING: this certificate is currently in use by a listener.")
		fmt.Println()
	}
	fmt.Println("This action is irreversible.")
}
