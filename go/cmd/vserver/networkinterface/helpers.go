package networkinterface

import (
	"fmt"
	"strings"

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
// are hidden from the table but remain in JSON.
var tableColumns = []string{"uuid", "name", "status", "vpcName", "serverName", "ip", "zone", "createdAt"}

// transformForTable adapts a network interface response for table output: it shortens
// the uuid, reduces the zone object to its name, and formats timestamps compactly.
// Applied only for table output — JSON keeps the full response.
func transformForTable(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
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
			case k == "zone":
				out[k] = zoneName(val)
			default:
				out[k] = transformForTable(val)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = transformForTable(item)
		}
		return out
	default:
		return v
	}
}

// zoneName reduces a zone object to its "name" field for table display; if the value
// is not a zone object it is returned unchanged.
func zoneName(val interface{}) interface{} {
	if z, ok := val.(map[string]interface{}); ok {
		if name, ok := z["name"].(string); ok {
			return name
		}
	}
	return val
}

// outputInterfaceList prints a network interface list. For table output it applies
// table-friendly transforms and a fixed column order; other formats (including JSON)
// show the full response.
func outputInterfaceList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformForTable(result), tableColumns)
	}
	return outputResult(cmd, cfg, result)
}

// parseTags converts repeated key=value flags into the API tag list used on create.
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

// parseTagRequests converts key=value flags into tag request entries for update-tags,
// stamping each with the given isEdited marker.
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
