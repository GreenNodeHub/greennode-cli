package placementgroup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a placement group's name or description",
	Long: `Update a placement group. Only the flags you provide are changed —
omit a flag to leave that value untouched. Provide --name and/or --description.`,
	RunE: runEdit,
}

func init() {
	f := updateCmd.Flags()
	f.String("placement-group-id", "", "Placement group (server group) ID (required)")
	f.String("name", "", "New name")
	f.String("description", "", "New description")
	if err := updateCmd.MarkFlagRequired("placement-group-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "placement-group-id", err))
	}
}

func runEdit(cmd *cobra.Command, args []string) error {
	groupID, _ := cmd.Flags().GetString("placement-group-id")
	if err := validator.ValidateID(groupID, "placement-group-id"); err != nil {
		return err
	}

	nameChanged := cmd.Flags().Changed("name")
	descChanged := cmd.Flags().Changed("description")
	if !nameChanged && !descChanged {
		return fmt.Errorf("nothing to update: provide --name and/or --description")
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	// serverGroupId is always sent; name/description only when the user set them.
	body := map[string]interface{}{"serverGroupId": groupID}
	if nameChanged {
		name, _ := cmd.Flags().GetString("name")
		body["name"] = name
	}
	if descChanged {
		description, _ := cmd.Flags().GetString("description")
		body["description"] = description
	}

	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/serverGroups/%s", projectID, groupID), body)
	if err != nil {
		return fmt.Errorf("failed to update placement group %s: %w", groupID, err)
	}

	return outputResult(cmd, cfg, result)
}
