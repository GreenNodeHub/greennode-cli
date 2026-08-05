package userimage

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
// are hidden from the table but remain in JSON.
var tableColumns = []string{"uuid", "name", "minDisk", "imageSize", "status", "createdAt"}

// transformForTable adapts a user image response for table output: it shortens the
// uuid and formats timestamps compactly. Applied only for table output — JSON keeps
// the full response.
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

// outputImageList prints a user image list. For table output it applies table-friendly
// transforms and a fixed column order; other formats (including JSON) show the full response.
func outputImageList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	if resolveOutput(cmd, cfg) == "table" {
		return vserverclient.OutputWithColumns(cmd, cfg, transformForTable(result), tableColumns)
	}
	return outputResult(cmd, cfg, result)
}
