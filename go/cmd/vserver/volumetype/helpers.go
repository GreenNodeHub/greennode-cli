package volumetype

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/client"
	"github.com/vngcloud/greennode-cli/internal/config"
	"github.com/vngcloud/greennode-cli/internal/vserverclient"
)

func createClient(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, error) {
	return vserverclient.BuildClient(cmd)
}

func getProjectID(cfg *config.Config) (string, error) {
	return vserverclient.ProjectID(cfg)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vserverclient.Output(cmd, cfg, data)
}

func suggestZones(apiClient *client.GreennodeClient, projectID string) error {
	return vserverclient.SuggestZoneIDs(apiClient, projectID)
}

// extractVolumeTypeZoneNames returns the "name" of every item in the volume_type_zones response.
func extractVolumeTypeZoneNames(result interface{}) []string {
	var items []interface{}
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, key := range []string{"volumeTypeZones", "data"} {
			if d, ok := v[key].([]interface{}); ok {
				items = d
				break
			}
		}
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			if name, ok := obj["name"].(string); ok && name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

// extractVolumeTypeZoneID pulls the zone ID out of the volume_type_zones response.
// If typeName is non-empty, it matches the item whose "name" field equals typeName.
// Otherwise it returns the ID of the first item.
func extractVolumeTypeZoneID(result interface{}, typeName string) (string, error) {
	tryID := func(obj map[string]interface{}) (string, bool) {
		for _, key := range []string{"id", "uuid", "volumeTypeZoneId", "zoneId"} {
			if v, ok := obj[key].(string); ok && v != "" {
				return v, true
			}
		}
		return "", false
	}

	matchesType := func(obj map[string]interface{}) bool {
		if typeName == "" {
			return true
		}
		name, _ := obj["name"].(string)
		return name == typeName
	}

	var items []interface{}
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, key := range []string{"volumeTypeZones", "data"} {
			if d, ok := v[key].([]interface{}); ok {
				items = d
				break
			}
		}
		if items == nil && matchesType(v) {
			if id, ok := tryID(v); ok {
				return id, nil
			}
		}
	}

	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if matchesType(obj) {
			if id, ok := tryID(obj); ok {
				return id, nil
			}
		}
	}

	if typeName != "" {
		return "", fmt.Errorf("no volume type zone with name %q found", typeName)
	}
	return "", fmt.Errorf("could not find volume type zone ID in response")
}
