package placementgroup

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new placement group",
	Long: `Create a new placement group. A placement policy is required.

If --policy-id is omitted, the available policies are listed and you are
prompted to pick one by number — so you don't need to remember policy IDs.`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Placement group name (required)")
	f.String("description", "", "Placement group description")
	f.String("policy-id", "", "Policy ID — run 'placement-group list-policies'; if omitted you'll be prompted to choose")
	if err := createCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "name", err))
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	policyID, _ := cmd.Flags().GetString("policy-id")

	if name == "" {
		return fmt.Errorf("flag --name is required")
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	if policyID == "" {
		policyID, err = promptPolicySelection(apiClient, projectID)
		if err != nil {
			return err
		}
	}

	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"policyId":    policyID,
	}

	result, err := apiClient.Post(fmt.Sprintf("/v2/%s/serverGroups", projectID), body)
	if err != nil {
		return fmt.Errorf("failed to create placement group: %w", err)
	}

	return outputResult(cmd, cfg, result)
}

// promptPolicySelection lists the available policies and asks the user to choose
// one by number, returning the selected policy's ID.
func promptPolicySelection(apiClient *client.GreennodeClient, projectID string) (string, error) {
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/serverGroups/policies", projectID), nil)
	if err != nil {
		return "", fmt.Errorf("--policy-id is required (also failed to fetch policies: %w)", err)
	}

	items := extractPolicyItems(result)
	if len(items) == 0 {
		return "", fmt.Errorf("no policies available; pass --policy-id explicitly")
	}

	fmt.Fprintln(os.Stderr, "Available policies:")
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "  [%d] %-22s %s\n", i+1, it.name, it.uuid)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "Select a policy by number [1-%d]: ", len(items))
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("no selection made: %w", err)
		}
		choice, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr != nil || choice < 1 || choice > len(items) {
			fmt.Fprintf(os.Stderr, "Invalid selection. Enter a number between 1 and %d.\n", len(items))
			if err != nil {
				return "", fmt.Errorf("no valid selection made")
			}
			continue
		}
		return items[choice-1].uuid, nil
	}
}
