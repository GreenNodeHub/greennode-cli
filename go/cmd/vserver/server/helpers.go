package server

import (
	"fmt"
	"os"
	"strings"
	"time"

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

// serverListColumns defines the columns shown in table mode for server list.
var serverListColumns = []string{"name", "status", "privateIp", "publicIp", "zone", "created", "app", "uuid"}

// serverDetailColumns defines the columns shown in table mode for a single server.
var serverDetailColumns = []string{"uuid", "name", "status", "privateIp", "publicIp", "zone", "created"}

// serverListFields defines which fields are included in list output (JSON and table).
var serverListFields = map[string]bool{
	"name":      true,
	"status":    true,
	"privateIp": true,
	"publicIp":  true,
	"zone":      true,
	"created":   true,
	"app":       true,
	"uuid":      true,
}

func outputServerList(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vserverclient.OutputWithColumns(cmd, cfg, data, serverListColumns)
}

func outputServerDetail(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vserverclient.OutputWithColumns(cmd, cfg, data, serverDetailColumns)
}

func parseCommaSeparated(s string) []string {
	result := []string{}
	if s == "" {
		return result
	}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func suggestZones(apiClient *client.GreennodeClient, projectID string) error {
	return vserverclient.SuggestZoneIDs(apiClient, projectID)
}

func suggestVPCs(apiClient *client.GreennodeClient, projectID string) error {
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks", projectID), map[string]string{"page": "1", "size": "50"})
	if err != nil {
		return fmt.Errorf("--network-id is required (also failed to fetch VPCs: %w)", err)
	}
	fmt.Fprintln(os.Stderr, "Flag --network-id is required. Available VPCs:")
	printItems(result, []string{"listData"}, func(obj map[string]interface{}) {
		fmt.Fprintf(os.Stderr, "  - %-40s  name: %v  cidr: %v\n", obj["id"], obj["displayName"], obj["cidr"])
	})
	return fmt.Errorf("flag --network-id is required")
}

func suggestSubnets(apiClient *client.GreennodeClient, projectID, networkID string) error {
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/networks/%s/subnets", projectID, networkID), map[string]string{"page": "1", "size": "50"})
	if err != nil {
		return fmt.Errorf("--subnet-id is required (also failed to fetch subnets: %w)", err)
	}
	fmt.Fprintln(os.Stderr, "Flag --subnet-id is required. Available subnets for VPC "+networkID+":")
	printItems(result, []string{"data"}, func(obj map[string]interface{}) {
		id := obj["uuid"]
		if id == nil {
			id = obj["id"]
		}
		fmt.Fprintf(os.Stderr, "  - %-40v  name: %v  cidr: %v\n", id, obj["name"], obj["cidr"])
	})
	return fmt.Errorf("flag --subnet-id is required")
}

func suggestImages() error {
	fmt.Fprintln(os.Stderr, "Flag --image-id is required. To see available images, run:")
	fmt.Fprintln(os.Stderr, "  grn vserver image list --type os")
	fmt.Fprintln(os.Stderr, "  grn vserver image list --type gpu")
	return fmt.Errorf("flag --image-id is required")
}

func suggestFlavors() error {
	fmt.Fprintln(os.Stderr, "Flag --flavor-id is required. To see available flavors, run:")
	fmt.Fprintln(os.Stderr, "  grn vserver flavor list-families          # see instance families")
	fmt.Fprintln(os.Stderr, "  grn vserver flavor list-codes             # see CPU platform codes")
	fmt.Fprintln(os.Stderr, "  grn vserver flavor list --family <family> --code <code>")
	return fmt.Errorf("flag --flavor-id is required")
}

func suggestRootDiskTypes(zoneID string) error {
	fmt.Fprintln(os.Stderr, "Flag --root-disk-type-id is required. To see available volume types, run:")
	fmt.Fprintf(os.Stderr, "  grn vserver volume-type list --zone-id %s\n", zoneID)
	return fmt.Errorf("flag --root-disk-type-id is required")
}

var serverRemoveKeys = map[string]bool{
	"zone":              true,
	"stopBeforeMigrate": true,
	"migrationStatus":   true,
	"migrateState":      true,
	"enableLog":         true,
	"enableMetric":      true,
	"metadata":          true,
}

func formatDateOnly(v interface{}) interface{} {
	s, ok := v.(string)
	if !ok || s == "" {
		return v
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC().Format("2006-01-02")
		}
	}
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func transformServer(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		if serverRemoveKeys[k] {
			continue
		}
		switch k {
		case "image":
			if img, ok := v.(map[string]interface{}); ok {
				result["imageId"] = img["id"]
			} else {
				result["imageId"] = v
			}
		case "flavor":
			if flv, ok := v.(map[string]interface{}); ok {
				result["flavorId"] = flv["flavorId"]
			} else {
				result["flavorId"] = v
			}
		case "internalInterfaces":
			if ifaces, ok := v.([]interface{}); ok && len(ifaces) > 0 {
				if iface, ok := ifaces[0].(map[string]interface{}); ok {
					result["privateIp"] = iface["fixedIp"]
					result["publicIp"] = iface["floatingIp"]
				}
			}
		case "zoneId":
			result["zone"] = v
		case "createdAt":
			result["created"] = formatDateOnly(v)
		case "product":
			if prod, ok := v.(map[string]interface{}); ok {
				if name, ok := prod["name"].(string); ok && name != "" {
					result["app"] = name
				}
			}
		default:
			result[k] = v
		}
	}
	return result
}

func filterServerListFields(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(serverListFields))
	for k, v := range obj {
		if serverListFields[k] && v != nil {
			result[k] = v
		}
	}
	return result
}

func transformServerList(items []interface{}) []interface{} {
	out := make([]interface{}, len(items))
	for i, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			out[i] = filterServerListFields(transformServer(obj))
		} else {
			out[i] = item
		}
	}
	return out
}

// transformServerResult applies field removals and renames to API server responses.
// Handles envelopes: {"data": {...}}, {"listData": [...]}, plain object, and plain array.
func transformServerResult(result interface{}) interface{} {
	switch v := result.(type) {
	case map[string]interface{}:
		// Single-object envelope: {"data": {...}}
		if data, ok := v["data"].(map[string]interface{}); ok {
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["data"] = transformServer(data)
			return out
		}
		// List envelope: {"listData": [...]}
		if listData, ok := v["listData"].([]interface{}); ok {
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["listData"] = transformServerList(listData)
			return out
		}
		// Plain server object
		return transformServer(v)
	case []interface{}:
		return transformServerList(v)
	}
	return result
}

// printItems iterates the items array from a response envelope and calls fn for each object.
func printItems(result interface{}, keys []string, fn func(map[string]interface{})) {
	var items []interface{}
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, key := range keys {
			if d, ok := v[key].([]interface{}); ok {
				items = d
				break
			}
		}
	}
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			fn(obj)
		}
	}
}
