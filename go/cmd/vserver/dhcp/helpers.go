package dhcp

import (
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
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

// resolveOutput returns the effective output format, mirroring vserverclient.Output:
// the --output flag, falling back to the configured default, then "json".
func resolveOutput(cmd *cobra.Command, cfg *config.Config) string {
	output, _ := cmd.Flags().GetString("output")
	if output == "" && cfg != nil {
		output = cfg.Output
	}
	if output == "" {
		output = "json"
	}
	return output
}

// uuidPreviewLen is how many runes of the uuid are shown in table output.
const uuidPreviewLen = 20

// tableColumns is the column order shown in table output. Fields not listed here
// are hidden from the table but remain in JSON. "associatedVpcs" is a derived count.
var tableColumns = []string{"uuid", "name", "status", "dnsServers", "associatedVpcs", "createdAt"}

// transformListForTable rebuilds the list envelope for table output. Each DHCP option
// becomes one main row plus an extra row per additional DNS server (so every address is
// shown); the extra rows carry only the dnsServers column. JSON output is untouched.
func transformListForTable(result interface{}) interface{} {
	switch v := result.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = val
		}
		for _, key := range []string{"listData", "data"} {
			if arr, ok := v[key].([]interface{}); ok {
				out[key] = expandRows(arr)
				return out
			}
		}
		return out
	case []interface{}:
		return expandRows(v)
	default:
		return result
	}
}

// expandRows turns each DHCP option object into one or more table rows.
func expandRows(items []interface{}) []interface{} {
	rows := make([]interface{}, 0, len(items))
	for _, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			rows = append(rows, it)
			continue
		}

		base := transformItem(obj)
		dns := dnsServers(obj["dnsServers"])
		if len(dns) == 0 {
			base["dnsServers"] = ""
			rows = append(rows, base)
			continue
		}

		base["dnsServers"] = dns[0]
		rows = append(rows, base)
		// One extra row per remaining DNS server — only the dnsServers cell is filled.
		for _, ip := range dns[1:] {
			rows = append(rows, map[string]interface{}{"dnsServers": ip})
		}
	}
	return rows
}

// transformItem applies the per-row transforms (excluding dnsServers, set by the caller):
// shortened uuid, compact timestamp, and the derived associatedVpcs count.
func transformItem(obj map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(obj)+1)
	for k, val := range obj {
		switch {
		case k == "uuid":
			if s, ok := val.(string); ok {
				out[k] = formatter.Truncate(s, uuidPreviewLen)
				continue
			}
			out[k] = val
		case k == "createdAt" || k == "updatedAt":
			if s, ok := val.(string); ok {
				out[k] = formatter.ShortDate(s)
				continue
			}
			out[k] = val
		default:
			out[k] = val
		}
	}
	out["associatedVpcs"] = countSlice(obj["associatedNetworks"])
	return out
}

// dnsServers returns the DNS server addresses as a string slice.
func dnsServers(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// countSlice returns the number of elements when v is an array, otherwise 0.
func countSlice(v interface{}) int {
	if arr, ok := v.([]interface{}); ok {
		return len(arr)
	}
	return 0
}

// outputDhcpList prints a DHCP option list. For table output it expands DNS servers
// into extra rows and uses a fixed column order; other formats show the full response.
func outputDhcpList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformListForTable(result), tableColumns)
	}
	return outputResult(cmd, cfg, result)
}

// outputDhcpDetail prints a single DHCP option. For table output it reuses the list's
// row-expansion (so DNS servers each get their own row); other formats show the full response.
func outputDhcpDetail(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		if obj := dhcpObject(result); obj != nil {
			return vserverclient.OutputWithColumns(cmd, cfg, expandRows([]interface{}{obj}), tableColumns)
		}
	}
	return outputResult(cmd, cfg, result)
}

// dhcpObject extracts the DHCP option object from a get response, handling both a
// {"data": {...}} envelope and a plain object.
func dhcpObject(result interface{}) map[string]interface{} {
	if v, ok := result.(map[string]interface{}); ok {
		if d, ok := v["data"].(map[string]interface{}); ok {
			return d
		}
		return v
	}
	return nil
}

// vpcTableColumns is the column order shown when listing the VPCs associated with a
// DHCP option set. Fields not listed are hidden from the table but remain in JSON.
var vpcTableColumns = []string{"id", "displayName", "cidr", "status", "createdAt"}

// transformVpcsForTable shortens VPC ids and formats timestamps for table output.
func transformVpcsForTable(result interface{}) interface{} {
	switch v := result.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = val
		}
		for _, key := range []string{"listData", "data"} {
			if arr, ok := v[key].([]interface{}); ok {
				out[key] = transformVpcRows(arr)
				return out
			}
		}
		return out
	case []interface{}:
		return transformVpcRows(v)
	default:
		return result
	}
}

func transformVpcRows(items []interface{}) []interface{} {
	rows := make([]interface{}, 0, len(items))
	for _, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			rows = append(rows, it)
			continue
		}
		out := make(map[string]interface{}, len(obj))
		for k, val := range obj {
			switch {
			case k == "id":
				if s, ok := val.(string); ok {
					out[k] = formatter.Truncate(s, uuidPreviewLen)
					continue
				}
				out[k] = val
			case k == "createdAt" || k == "updatedAt":
				if s, ok := val.(string); ok {
					out[k] = formatter.ShortDate(s)
					continue
				}
				out[k] = val
			default:
				out[k] = val
			}
		}
		rows = append(rows, out)
	}
	return rows
}

// outputVpcList prints the VPCs associated with a DHCP option set. For table output it
// shortens ids and formats timestamps; other formats show the full response.
func outputVpcList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformVpcsForTable(result), vpcTableColumns)
	}
	return outputResult(cmd, cfg, result)
}
