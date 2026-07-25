package vserverclient

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/client"
)

// extractCompletions pulls "id\tname" pairs from an API response.
// listKeys: envelope keys to try in order (e.g. "listData", "data", "images").
// idKey: field used as the completion value; nameKey: shown as the tab description.
func extractCompletions(result interface{}, listKeys []string, idKey, nameKey string) []string {
	var items []interface{}
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, key := range listKeys {
			if d, ok := v[key].([]interface{}); ok {
				items = d
				break
			}
		}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := obj[idKey].(string)
		if id == "" {
			continue
		}
		if name, _ := obj[nameKey].(string); name != "" {
			out = append(out, id+"\t"+name)
		} else {
			out = append(out, id)
		}
	}
	return out
}

func buildCompleter(fetch func(*client.GreennodeClient, string) ([]string, error)) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		c, cfg, err := BuildClient(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		projectID, err := ProjectID(cfg)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		completions, err := fetch(c, projectID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteServerIDs completes --server-id flags.
var CompleteServerIDs = buildCompleter(func(c *client.GreennodeClient, projectID string) ([]string, error) {
	result, err := c.Get(fmt.Sprintf("/v2/%s/servers", projectID), map[string]string{"page": "1", "size": "100"})
	if err != nil {
		return nil, err
	}
	return extractCompletions(result, []string{"listData"}, "uuid", "name"), nil
})

// CompleteVolumeIDs completes --volume-id flags.
var CompleteVolumeIDs = buildCompleter(func(c *client.GreennodeClient, projectID string) ([]string, error) {
	result, err := c.Get(fmt.Sprintf("/v2/%s/volumes", projectID), map[string]string{"page": "1", "size": "100"})
	if err != nil {
		return nil, err
	}
	return extractCompletions(result, []string{"listData"}, "uuid", "name"), nil
})

// CompleteVPCIDs completes --network-id and --vpc-id flags.
var CompleteVPCIDs = buildCompleter(func(c *client.GreennodeClient, projectID string) ([]string, error) {
	result, err := c.Get(fmt.Sprintf("/v2/%s/networks", projectID), map[string]string{"page": "1", "size": "100"})
	if err != nil {
		return nil, err
	}
	return extractCompletions(result, []string{"listData"}, "id", "displayName"), nil
})

// CompleteSecgroupIDs completes --secgroup-id and --security-group flags.
var CompleteSecgroupIDs = buildCompleter(func(c *client.GreennodeClient, projectID string) ([]string, error) {
	result, err := c.Get(fmt.Sprintf("/v2/%s/secgroups", projectID), map[string]string{"page": "1", "size": "100"})
	if err != nil {
		return nil, err
	}
	return extractCompletions(result, []string{"listData"}, "id", "name"), nil
})

// CompleteImageIDs completes --image-id flags by combining OS and GPU images.
var CompleteImageIDs = buildCompleter(func(c *client.GreennodeClient, projectID string) ([]string, error) {
	var completions []string
	for _, imageType := range []string{"os", "gpu"} {
		result, err := c.Get(fmt.Sprintf("/v1/%s/images/%s", projectID, imageType), map[string]string{"page": "1", "size": "100"})
		if err != nil {
			continue
		}
		completions = append(completions, extractCompletions(result, []string{"images", "data"}, "id", "name")...)
	}
	return completions, nil
})

// CompleteZoneIDs completes --zone-id flags, showing only enabled zones.
func CompleteZoneIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c, cfg, err := BuildClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	projectID, err := ProjectID(cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	result, err := c.Get(fmt.Sprintf("/v1/%s/zones", projectID), nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var items []interface{}
	if m, ok := result.(map[string]interface{}); ok {
		if d, ok := m["data"].([]interface{}); ok {
			items = d
		}
	}
	var completions []string
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
		if uuid == "" {
			continue
		}
		if name != "" {
			completions = append(completions, uuid+"\t"+name)
		} else {
			completions = append(completions, uuid)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// CompleteSubnetIDs completes --subnet-id flags.
// Reads VPC from --network-id (server create) or --vpc-id (subnet commands).
func CompleteSubnetIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	networkID, _ := cmd.Flags().GetString("network-id")
	if networkID == "" {
		networkID, _ = cmd.Flags().GetString("vpc-id")
	}
	if networkID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, cfg, err := BuildClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	projectID, err := ProjectID(cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	result, err := c.Get(fmt.Sprintf("/v2/%s/networks/%s/subnets", projectID, networkID), map[string]string{"page": "1", "size": "100"})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return extractCompletions(result, []string{"data", "listData"}, "uuid", "name"), cobra.ShellCompDirectiveNoFileComp
}

// CompleteVolumeTypeIDs completes --volume-type-id flags.
// Reads the zone from --zone-id to perform the two-step lookup.
func CompleteVolumeTypeIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	if zoneID == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, cfg, err := BuildClient(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	projectID, err := ProjectID(cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	// Step 1: resolve volume-type-zone IDs for this zone
	zoneResult, err := c.Get(fmt.Sprintf("/v1/%s/volume_type_zones", projectID), map[string]string{"zoneId": zoneID})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var vzItems []interface{}
	switch v := zoneResult.(type) {
	case []interface{}:
		vzItems = v
	case map[string]interface{}:
		for _, key := range []string{"volumeTypeZones", "data"} {
			if d, ok := v[key].([]interface{}); ok {
				vzItems = d
				break
			}
		}
	}
	// Step 2: fetch volume types for each volume-type-zone
	var completions []string
	for _, item := range vzItems {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var vzID string
		for _, key := range []string{"id", "uuid", "volumeTypeZoneId"} {
			if v, ok := obj[key].(string); ok && v != "" {
				vzID = v
				break
			}
		}
		if vzID == "" {
			continue
		}
		result, err := c.Get(fmt.Sprintf("/v1/%s/%s/volume_types", projectID, vzID), nil)
		if err != nil {
			continue
		}
		completions = append(completions, extractCompletions(result, []string{"data", "volumeTypes"}, "id", "name")...)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
