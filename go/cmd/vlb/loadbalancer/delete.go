package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a load balancer",
	RunE:  runDelete,
}

func init() {
	addLBFlag(deleteCmd)
	f := deleteCmd.Flags()
	f.Bool("force", false, "Skip confirmation prompt")
	f.Bool("dry-run", false, "Preview what will be deleted without executing")
}

func runDelete(cmd *cobra.Command, args []string) error {
	lbID, err := lbIDOf(cmd)
	if err != nil {
		return err
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	current, err := apiClient.Get(vlbclient.LoadBalancer(projectID, lbID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch load balancer %s: %w", lbID, err)
	}
	printDeletePreview(current)

	if dryRun {
		cli.DryRunNotice("delete")
		return nil
	}
	if !cli.Confirm(force, "Are you sure you want to delete this load balancer?") {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Delete(vlbclient.LoadBalancer(projectID, lbID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer %s: %w", lbID, err)
	}

	return outputRaw(cmd, cfg, result)
}

// printDeletePreview describes the load balancer about to be deleted. The GET
// response wraps the resource in a "data" envelope.
func printDeletePreview(response interface{}) {
	lb := unwrapData(response)
	fmt.Println("The following load balancer will be deleted:")
	fmt.Println()
	fmt.Printf("  ID:      %v\n", lb["uuid"])
	fmt.Printf("  Name:    %v\n", lb["name"])
	fmt.Printf("  Address: %v\n", lb["address"])
	fmt.Printf("  Type:    %v\n", lb["displayType"])
	fmt.Printf("  Status:  %v\n", lb["displayStatus"])
	fmt.Println()
	fmt.Println("Its listeners, pools and members are deleted with it. This action is irreversible.")
}

// unwrapData returns the payload of a {"data": {...}} envelope, or the response
// itself when it is already the bare object. Returns an empty map for anything
// else so preview printing degrades to blanks rather than panicking.
func unwrapData(response interface{}) map[string]interface{} {
	m, ok := response.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		return data
	}
	return m
}
