package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a listener",
	RunE:  runDelete,
}

func init() {
	addLBFlag(deleteCmd)
	addListenerFlag(deleteCmd)
	f := deleteCmd.Flags()
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
}

func runDelete(cmd *cobra.Command, args []string) error {
	lbID, listenerID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(vlbclient.Listener(projectID, lbID, listenerID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch listener %s: %w", listenerID, err)
	}
	printDeletePreview(current)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this listener?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(vlbclient.Listener(projectID, lbID, listenerID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete listener %s: %w", listenerID, err)
	}

	return outputRaw(cmd, cfg, result)
}

func printDeletePreview(response interface{}) {
	l := unwrapData(response)
	fmt.Println("The following listener will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:           %v\n", l["uuid"])
	fmt.Printf("  Name:         %v\n", l["name"])
	fmt.Printf("  Protocol:     %v:%v\n", l["protocol"], l["protocolPort"])
	fmt.Printf("  Default pool: %v\n", l["defaultPoolName"])
	fmt.Println()
	fmt.Println("Its L7 policies are deleted with it. This action is irreversible.")
}
