package agentbase

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/config"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage the active environment context",
	Long: `Manage the active environment context (dev or prod).

The environment controls which API endpoints are used for all commands.
Resolution order: GREENNODE_ENV env var → ./.greennode.json → default (prod)`,
}

var contextSwitchCmd = &cobra.Command{
	Use:   "switch <dev|prod>",
	Short: "Switch the active environment",
	Long: `Switch the active environment context to 'dev' or 'prod'.

This writes the 'env' field to ./.greennode.json.
To override without modifying the config file, set GREENNODE_ENV instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := config.Env(args[0])
		if err := config.SetEnv(env); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Switched to environment: %s\n", env)
		return nil
	},
}

var contextCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active environment and resolved endpoints",
	Long:  `Display the currently active environment and all resolved API base URLs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadWithEnv(envOverride)
		if err != nil {
			return err
		}

		source := "config file (./.greennode.json)"
		switch {
		case envOverride != "":
			source = "--env flag"
		case os.Getenv("GREENNODE_ENV") != "":
			source = "environment variable (GREENNODE_ENV)"
		}

		fmt.Fprintf(os.Stdout, "Environment : %s\n", cfg.Env)
		fmt.Fprintf(os.Stdout, "Source      : %s\n\n", source)

		output.Table(
			[]string{"Service", "Base URL"},
			[][]string{
				{"Identity", cfg.Endpoints.Identity},
				{"Runtime", cfg.Endpoints.Runtime},
				{"Memory", cfg.Endpoints.Memory},
				{"OAuth2 Token", cfg.Endpoints.OAuth2Token},
			},
		)
		return nil
	},
}

var contextHeadersCmd = &cobra.Command{
	Use:   "headers",
	Short: "Show platform request headers reference",
	Long:  `Display the standard X-GreenNode-AgentBase-* HTTP request headers used by the platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		output.Table(
			[]string{"Header", "Description"},
			[][]string{
				{"X-GreenNode-AgentBase-Session-Id", "Unique session identifier for conversation continuity"},
				{"X-GreenNode-AgentBase-Request-Id", "Unique request identifier for tracing"},
				{"X-GreenNode-AgentBase-Access-Token", "User access token for 3LO OAuth2 flows"},
				{"X-GreenNode-AgentBase-User-Id", "User identifier forwarded to the agent"},
				{"X-GreenNode-AgentBase-OAuth2-Callback-Url", "Callback URL for OAuth2 redirect flows"},
				{"Authorization", "Bearer token (client credentials OAuth2 token)"},
				{"X-GreenNode-AgentBase-Custom-*", "Arbitrary custom headers forwarded to the agent"},
			},
		)
	},
}

var contextDecoratorsCmd = &cobra.Command{
	Use:   "decorators",
	Short: "Show SDK decorator reference",
	Long:  `Display the GreenNode AgentBase SDK decorators and their purpose.`,
	Run: func(cmd *cobra.Command, args []string) {
		output.Table(
			[]string{"Decorator", "Module", "Description"},
			[][]string{
				{
					"@entrypoint",
					"GreenNodeAgentBaseApp",
					"Registers the main handler. Extracts AgentBase context from incoming request headers.",
				},
				{
					"@requires_api_key",
					"identity",
					"Fetches a static or delegated API key before the handler is invoked.",
				},
				{
					"@requires_access_token",
					"identity",
					"Fetches an M2M (client credentials) or 3LO OAuth2 token before the handler is invoked.",
				},
			},
		)
	},
}

func init() {
	AgentbaseCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextSwitchCmd)
	contextCmd.AddCommand(contextCurrentCmd)
	contextCmd.AddCommand(contextHeadersCmd)
	contextCmd.AddCommand(contextDecoratorsCmd)
}
