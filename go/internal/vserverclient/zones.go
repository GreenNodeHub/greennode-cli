package vserverclient

import (
	"fmt"
	"os"

	"github.com/vngcloud/greennode-cli/internal/client"
)

// SuggestZoneIDs fetches available zones and prints enabled ones to stderr,
// then returns an error telling the user to set --zone-id.
func SuggestZoneIDs(apiClient *client.GreennodeClient, projectID string) error {
	result, err := apiClient.Get(fmt.Sprintf("/v1/%s/zones", projectID), nil)
	if err != nil {
		return fmt.Errorf("--zone-id is required (also failed to fetch zones: %w)", err)
	}

	var items []interface{}
	if m, ok := result.(map[string]interface{}); ok {
		if d, ok := m["data"].([]interface{}); ok {
			items = d
		}
	} else if arr, ok := result.([]interface{}); ok {
		items = arr
	}

	fmt.Fprintln(os.Stderr, "Flag --zone-id is required. Available zones:")
	for _, item := range items {
		zone, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if enabled, _ := zone["isEnabled"].(bool); !enabled {
			continue
		}
		uuid, _ := zone["uuid"].(string)
		name, _ := zone["name"].(string)
		desc, _ := zone["description"].(string)
		fmt.Fprintf(os.Stderr, "  - %-15s (%s) %s\n", uuid, name, desc)
	}
	return fmt.Errorf("flag --zone-id is required")
}
