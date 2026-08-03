package server

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var createImageCmd = &cobra.Command{
	Use:   "create-image",
	Short: "Create a user image from a vServer instance",
	Long: `Create a new user image (custom image) from an existing vServer instance.

Tags are optional key/value pairs. Pass --tag once per pair, in key=value form,
for example: --tag env=prod --tag team=infra. Run 'vserver server tag-key' and
'vserver server tag-value --key <key>' to discover existing keys and values.`,
	RunE: runCreateImage,
}

func init() {
	f := createImageCmd.Flags()
	f.String("server-id", "", "Server ID to create the image from (required)")
	f.String("name", "", "Name of the new image (required)")
	f.StringArray("tag", nil, "Tag to attach, in key=value form (repeatable)")

	for _, name := range []string{"server-id", "name"} {
		if err := createImageCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runCreateImage(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	name, _ := cmd.Flags().GetString("name")
	rawTags, _ := cmd.Flags().GetStringArray("tag")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("--name is required")
	}

	tags, err := parseTags(rawTags)
	if err != nil {
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

	body := map[string]interface{}{
		"name": name,
		"tags": tags,
	}

	result, err := apiClient.Post(
		fmt.Sprintf("/v2/%s/user-images/servers/%s", projectID, serverID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to create image from server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}

// parseTags converts repeated --tag key=value flags into the API tag list.
// Each entry must contain a "=" separator and a non-empty key.
func parseTags(raw []string) ([]interface{}, error) {
	tags := make([]interface{}, 0, len(raw))
	for _, t := range raw {
		key, value, found := strings.Cut(t, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --tag %q: expected key=value form with a non-empty key", t)
		}
		tags = append(tags, map[string]interface{}{
			"key":   key,
			"value": strings.TrimSpace(value),
		})
	}
	return tags, nil
}
