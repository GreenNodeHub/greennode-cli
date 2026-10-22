package vserverclient

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

// BuildClient creates a GreennodeClient from cobra command flags.
func BuildClient(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, error) {
	return BuildOperationClient(cmd, false)
}

// BuildOperationClient adds only the operation's required headers.
func BuildOperationClient(cmd *cobra.Command, requirePortalUserID bool) (*client.GreennodeClient, *config.Config, error) {
	return cli.BuildClient(cmd, cli.ClientOptions{
		ResolveEndpoint: func(cfg *config.Config) (string, error) {
			return cfg.GetEndpoint("vserver")
		},
		ApplyRegion: true,
		AfterBuild: func(_ *cobra.Command, cfg *config.Config, apiClient *client.GreennodeClient) error {
			if !requirePortalUserID {
				return nil
			}
			if _, err := config.ValidatePositiveInt32(cfg.PortalUserID); err != nil {
				return fmt.Errorf("this vServer operation requires a positive 32-bit portal_user_id; configure it or set GRN_PORTAL_USER_ID")
			}
			apiClient.SetHeader("portal-user-id", cfg.PortalUserID)
			return nil
		},
	})
}

// ProjectID extracts and validates the project ID from config.
func ProjectID(cfg *config.Config) (string, error) {
	if cfg.ProjectID == "" {
		return "", fmt.Errorf("project_id is not configured. Run 'grn configure' or set GRN_DEFAULT_PROJECT_ID")
	}
	if err := validator.ValidateID(cfg.ProjectID, "project-id"); err != nil {
		return "", err
	}
	return cfg.ProjectID, nil
}

// Output formats and writes the API result to stdout.
func Output(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	output, _ := cmd.Flags().GetString("output")
	query, _ := cmd.Flags().GetString("query")

	if output == "" && cfg != nil {
		output = cfg.Output
	}
	if output == "" {
		output = "json"
	}

	colorMode, _ := cmd.Flags().GetString("color")
	return formatter.FormatColor(data, output, query, os.Stdout, formatter.ColorEnabled(colorMode, os.Stdout))
}

// OutputWithColumns applies ordered table columns.
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
