package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an L7 policy",
	RunE:  runDelete,
}

func init() {
	addParentFlags(deleteCmd)
	addPolicyFlag(deleteCmd)
	f := deleteCmd.Flags()
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
}

func runDelete(cmd *cobra.Command, args []string) error {
	lbID, listenerID, policyID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(vlbclient.Policy(projectID, lbID, listenerID, policyID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch policy %s: %w", policyID, err)
	}
	printDeletePreview(current)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this L7 policy?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(vlbclient.Policy(projectID, lbID, listenerID, policyID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete policy %s: %w", policyID, err)
	}

	return outputRaw(cmd, cfg, result)
}

func printDeletePreview(response interface{}) {
	p := unwrapData(response)
	fmt.Println("The following L7 policy will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:       %v\n", p["uuid"])
	fmt.Printf("  Name:     %v\n", p["name"])
	fmt.Printf("  Action:   %v\n", p["action"])
	fmt.Printf("  Position: %v\n", p["position"])
	if rules, ok := p["l7Rules"].([]interface{}); ok {
		fmt.Printf("  Rules:    %d\n", len(rules))
	}
	fmt.Println()
	fmt.Println("Its rules are deleted with it, and the remaining policies are renumbered. This action is irreversible.")
}
