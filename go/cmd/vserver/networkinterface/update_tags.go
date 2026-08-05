package networkinterface

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateTagsCmd = &cobra.Command{
	Use:   "update-tags",
	Short: "Update the tag list of a network interface",
	Long: `Replace the key/value tag list of an elastic network interface.

The tags you provide become the complete list for the interface — any tag not
included is removed. Tags are passed in key=value form:

  --tag key=value         a tag whose value was not changed (isEdited=false)
  --edited-tag key=value  a tag whose value was changed   (isEdited=true)

The isEdited marker is sent per tag so the backend can tell which entries were
modified.`,
	RunE: runUpdateTags,
}

func init() {
	f := updateTagsCmd.Flags()
	f.String("network-interface-id", "", "Network interface ID whose tags to update (required)")
	f.StringArray("tag", nil, "Unchanged tag in key=value form, isEdited=false (repeatable)")
	f.StringArray("edited-tag", nil, "Changed tag in key=value form, isEdited=true (repeatable)")

	if err := updateTagsCmd.MarkFlagRequired("network-interface-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "network-interface-id", err))
	}
}

func runUpdateTags(cmd *cobra.Command, args []string) error {
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	rawEdited, _ := cmd.Flags().GetStringArray("edited-tag")

	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
		return err
	}

	tagList := make([]interface{}, 0, len(rawTags)+len(rawEdited))
	tags, err := parseTagRequests(rawTags, false)
	if err != nil {
		return err
	}
	tagList = append(tagList, tags...)
	edited, err := parseTagRequests(rawEdited, true)
	if err != nil {
		return err
	}
	tagList = append(tagList, edited...)

	if len(tagList) == 0 {
		return fmt.Errorf("at least one tag is required (--tag or --edited-tag)")
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"resourceId":     interfaceID,
		"resourceType":   "NETWORK-INTERFACE",
		"tagRequestList": tagList,
	}

	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/tag/resource/%s", projectID, interfaceID), body)
	if err != nil {
		return fmt.Errorf("failed to update tags for network interface %s: %w", interfaceID, err)
	}

	return outputResult(cmd, cfg, result)
}
