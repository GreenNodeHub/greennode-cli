package pool

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a pool",
	RunE:  runDelete,
}

func init() {
	addLBFlag(deleteCmd)
	addPoolFlag(deleteCmd)
	f := deleteCmd.Flags()
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
}

func runDelete(cmd *cobra.Command, args []string) error {
	lbID, poolID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(vlbclient.Pool(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch pool %s: %w", poolID, err)
	}
	printDeletePreview(current)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this pool?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(vlbclient.Pool(projectID, lbID, poolID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete pool %s: %w", poolID, err)
	}

	return outputRaw(cmd, cfg, result)
}

func printDeletePreview(response interface{}) {
	p := unwrapData(response)
	fmt.Println("The following pool will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:        %v\n", p["uuid"])
	fmt.Printf("  Name:      %v\n", p["name"])
	fmt.Printf("  Protocol:  %v\n", p["protocol"])
	fmt.Printf("  Algorithm: %v\n", p["loadBalanceMethod"])
	if members, ok := p["members"].([]interface{}); ok {
		fmt.Printf("  Members:   %d\n", len(members))
	}
	fmt.Println()
	fmt.Println("Its members and health monitor are deleted with it. This action is irreversible.")
}
