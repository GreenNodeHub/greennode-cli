package cli

import (
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/spf13/cobra"
)

// Output formats and prints an API response using the command's --output/--query
// flags, falling back to the configured default output then "json".
func Output(cmd *cobra.Command, data interface{}) error {
	output, _ := cmd.Flags().GetString("output")
	query, _ := cmd.Flags().GetString("query")

	if output == "" {
		profile, _ := cmd.Flags().GetString("profile")
		cfg, _ := config.LoadConfig(profile)
		if cfg != nil {
			output = cfg.Output
		}
	}
	if output == "" {
		output = "json"
	}

	colorMode, _ := cmd.Flags().GetString("color")
	return formatter.FormatColor(data, output, query, os.Stdout, formatter.ColorEnabled(colorMode, os.Stdout))
}

// OutputWithConfig is Output for callers that already hold the resolved config
// (see NewClientWithConfig), avoiding a second config load per command.
func OutputWithConfig(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return OutputWithColumns(cmd, cfg, data, nil)
}

// OutputWithColumns is OutputWithConfig that, for "table" output only, renders
// just the named fields in the given order. Other formats are unaffected — JSON
// always carries the full payload.
func OutputWithColumns(cmd *cobra.Command, cfg *config.Config, data interface{}, columns []string) error {
	output, _ := cmd.Flags().GetString("output")
	query, _ := cmd.Flags().GetString("query")

	if output == "" && cfg != nil {
		output = cfg.Output
	}
	if output == "" {
		output = "json"
	}

	colorMode, _ := cmd.Flags().GetString("color")
	color := formatter.ColorEnabled(colorMode, os.Stdout)
	if output == "table" && len(columns) > 0 {
		return formatter.FormatTableWithColumnsColor(data, columns, query, os.Stdout, color)
	}
	return formatter.FormatColor(data, output, query, os.Stdout, color)
}
