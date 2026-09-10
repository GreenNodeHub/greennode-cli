package cli

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/spf13/cobra"
)

// Output applies flags, then configured defaults, then JSON.
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
	if output != "json" && output != "text" && output != "table" {
		return fmt.Errorf("invalid output format %q: must be json, text, or table", output)
	}

	colorMode, _ := cmd.Flags().GetString("color")
	return formatter.FormatColor(data, output, query, os.Stdout, formatter.ColorEnabled(colorMode, os.Stdout))
}
