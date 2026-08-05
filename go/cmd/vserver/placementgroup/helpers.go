package placementgroup

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

// Preview widths (in runes) for long table fields.
const (
	uuidPreviewLen = 20
	descPreviewLen = 40
)

// tableColumns is the column order shown in table output. Fields not listed here
// (e.g. policyId, serverGroupId) are hidden from the table but remain in JSON.
var tableColumns = []string{"uuid", "name", "policyName", "description", "servers", "createdAt"}

// serverNames turns a "servers" array of {name, uuid} objects into a comma-separated
// list of names, so the table column shows names instead of nested maps.
func serverNames(v interface{}) interface{} {
	items, ok := v.([]interface{})
	if !ok {
		return v
	}
	names := make([]string, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok && n != "" {
				names = append(names, n)
				continue
			}
		}
		names = append(names, fmt.Sprint(it))
	}
	return strings.Join(names, ", ")
}

// transformForTable adapts a placement group / policy response for table output:
// it shortens the uuid and description, formats timestamps compactly, and renders
// servers as a comma-separated list of names. Applied only for table output — JSON
// keeps the full response.
func transformForTable(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			switch {
			case k == "uuid":
				out[k] = truncStr(val, uuidPreviewLen)
			case k == "description":
				out[k] = truncStr(val, descPreviewLen)
			case k == "createdAt" || k == "updatedAt":
				out[k] = shortDate(val)
			case k == "servers":
				out[k] = serverNames(val)
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

// truncStr shortens a string value to max runes; non-strings are returned as-is.
func truncStr(val interface{}, max int) interface{} {
	if s, ok := val.(string); ok {
		return formatter.Truncate(s, max)
	}
	return val
}

// shortDate reformats a timestamp string compactly; non-strings are returned as-is.
func shortDate(val interface{}) interface{} {
	if s, ok := val.(string); ok {
		return formatter.ShortDate(s)
	}
	return val
}

// outputGroupList prints a placement group list. For table output it applies
// table-friendly transforms and a fixed column order; other formats (including
// JSON) show the full response.
func outputGroupList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformForTable(result), tableColumns)
	}
	return outputResult(cmd, cfg, result)
}

// policyTableColumns is the column order shown when listing policies as a table.
var policyTableColumns = []string{"uuid", "name", "status"}

// selectPolicyLanguage collapses the bilingual description fields into a single
// "description" field for the requested language ("vi" → descriptionVi, anything
// else → the default English description) and drops "descriptionVi".
func selectPolicyLanguage(v interface{}, lang string) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[k] = selectPolicyLanguage(val, lang)
		}
		if _, hasVi := out["descriptionVi"]; hasVi {
			if lang == "vi" {
				out["description"] = out["descriptionVi"]
			}
			delete(out, "descriptionVi")
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = selectPolicyLanguage(item, lang)
		}
		return out
	default:
		return v
	}
}

// outputPolicies prints the policy list. The description is rendered in the chosen
// language; the table view shows only uuid/name/status with a shortened uuid.
func outputPolicies(cmd *cobra.Command, cfg *config.Config, result interface{}, lang string) error {
	result = selectPolicyLanguage(result, lang)
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformForTable(result), policyTableColumns)
	}
	return outputResult(cmd, cfg, result)
}

// policyItem is a single policy reduced to the fields needed for selection.
type policyItem struct {
	uuid string
	name string
}

// extractPolicyItems pulls policy {uuid, name} pairs from a policies response,
// handling the common envelope keys and a plain array.
func extractPolicyItems(result interface{}) []policyItem {
	var raw []interface{}
	switch v := result.(type) {
	case []interface{}:
		raw = v
	case map[string]interface{}:
		for _, key := range []string{"data", "listData", "policies"} {
			if d, ok := v[key].([]interface{}); ok {
				raw = d
				break
			}
		}
	}

	items := make([]policyItem, 0, len(raw))
	for _, r := range raw {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		uuid, _ := m["uuid"].(string)
		if uuid == "" {
			uuid, _ = m["id"].(string)
		}
		if uuid == "" {
			continue
		}
		name, _ := m["name"].(string)
		items = append(items, policyItem{uuid: uuid, name: name})
	}
	return items
}
