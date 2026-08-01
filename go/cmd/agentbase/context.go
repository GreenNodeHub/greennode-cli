package agentbase

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show the active environment context",
	Long: `Show the active environment context (dev or prod) for the agentbase subtree.

The environment selects the agentbase API endpoints AND the IAM v2 token URL; it
is stored as iam_env in the shared ~/.greennode profile (default prod), the same
selector vks/vserver use. Switch it with 'grn configure set iam_env <dev|prod>'
(machine) or 'grn login --iam-env <env>' (user).`,
}

var contextCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active environment and resolved endpoints",
	Long:  `Display the currently active environment (resolved from the profile's iam_env) and all agentbase API base URLs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ab := mustLoadAgentbaseCtx(cmd)
		source := "prod default (iam_env unset)"
		if ab.shared.IamEnv != "" {
			source = fmt.Sprintf("iam_env=%s (profile %q)", ab.shared.IamEnv, ab.shared.Profile)
		}

		fmt.Fprintf(os.Stdout, "Profile     : %s\n", ab.shared.Profile)
		fmt.Fprintf(os.Stdout, "Environment : %s\n", ab.env)
		fmt.Fprintf(os.Stdout, "Source      : %s\n\n", source)

		output.Table(
			[]string{"Service", "Base URL"},
			[][]string{
				{"Identity", ab.endpoints.Identity},
				{"Runtime", ab.endpoints.Runtime},
				{"Memory", ab.endpoints.Memory},
				{"OAuth2 Token", ab.endpoints.OAuth2Token},
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
	contextCmd.AddCommand(contextCurrentCmd)
	contextCmd.AddCommand(contextHeadersCmd)
	contextCmd.AddCommand(contextDecoratorsCmd)
}
