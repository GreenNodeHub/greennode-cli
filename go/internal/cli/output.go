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
