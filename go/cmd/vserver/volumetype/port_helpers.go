package volumetype

import "fmt"

func defaultAvailabilityZoneID(result any) (string, error) {
	var items []any
	switch v := result.(type) {
	case []any:
		items = v
	case map[string]any:
		if data, ok := v["data"].([]any); ok {
			items = data
		}
	}

	for _, item := range items {
		zone, ok := item.(map[string]any)
		if !ok {
			continue
		}
		isDefault, _ := zone["isDefault"].(bool)
		isEnabled, _ := zone["isEnabled"].(bool)
		zoneID, _ := zone["uuid"].(string)
		if isDefault && isEnabled && zoneID != "" {
			return zoneID, nil
		}
	}

	return "", fmt.Errorf("could not find an enabled default availability zone; pass --zone-id")
}
