package userimage

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateTagsCmd = &cobra.Command{
	Use:   "update-tags",
	Short: "Update the tag list of a user image",
	Long: `Replace the key/value tag list of a user image.

The tags you provide become the complete list for the image — any tag not
included is removed. Tags are passed in key=value form:

  --tag key=value         a tag whose value was not changed (isEdited=false)
  --edited-tag key=value  a tag whose value was changed   (isEdited=true)

The isEdited marker is sent per tag so the backend can tell which entries were
modified. Pass each flag once per pair, for example:
  --tag env=prod --edited-tag vks-cluster-ids=k8s-...`,
	RunE: runUpdateTags,
}

func init() {
	f := updateTagsCmd.Flags()
	f.String("user-image-id", "", "User image ID whose tags to update (required)")
	f.StringArray("tag", nil, "Unchanged tag in key=value form, isEdited=false (repeatable)")
	f.StringArray("edited-tag", nil, "Changed tag in key=value form, isEdited=true (repeatable)")

	if err := updateTagsCmd.MarkFlagRequired("user-image-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "user-image-id", err))
	}
}

func runUpdateTags(cmd *cobra.Command, args []string) error {
	imageID, _ := cmd.Flags().GetString("user-image-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	rawEdited, _ := cmd.Flags().GetStringArray("edited-tag")

	if err := validator.ValidateID(imageID, "user-image-id"); err != nil {
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
		"resourceId":     imageID,
		"resourceType":   "IMAGE",
		"tagRequestList": tagList,
	}

	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/tag/resource/%s", projectID, imageID), body)
	if err != nil {
		return fmt.Errorf("failed to update tags for user image %s: %w", imageID, err)
	}

	return outputResult(cmd, cfg, result)
}

// parseTagRequests converts key=value flags into tag request entries, stamping
// each with the given isEdited marker.
func parseTagRequests(raw []string, isEdited bool) ([]interface{}, error) {
	out := make([]interface{}, 0, len(raw))
	for _, t := range raw {
		key, value, found := strings.Cut(t, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid tag %q: expected key=value form with a non-empty key", t)
		}
		out = append(out, map[string]interface{}{
			"isEdited": isEdited,
			"key":      key,
			"value":    strings.TrimSpace(value),
		})
	}
	return out, nil
}
