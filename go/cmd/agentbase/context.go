package agentbase

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	agentbaseconfig "github.com/vngcloud/greennode-cli/internal/agentbase/config"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
	coreconfig "github.com/vngcloud/greennode-cli/internal/config"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage the active environment context",
	Long: `Manage the active environment context (dev or prod) for the agentbase subtree.

The environment selects the agentbase API endpoints AND the IAM v2 token URL; it
is stored as iam_env in the shared ~/.greennode profile (default prod), the same
selector vks/vserver use. 'context switch <dev|prod>' repoints a machine
profile's iam_env; a user profile is bound to its login token — re-login with
'grn login --iam-env <env>' to switch it.`,
}

var contextSwitchCmd = &cobra.Command{
	Use:   "switch <dev|prod>",
	Short: "Switch the active environment",
	Long: `Switch the active environment context to 'dev' or 'prod'.

This writes the 'iam_env' key to the shared ~/.greennode profile, so all three
services (vks/vserver/agentbase) resolve the v2 token URL + endpoints from one
place. Only machine profiles can switch here: a user profile's iam_env is bound
to its login token (the dev/prod client_id selected at login), so switching it
would invalidate the refresh token — re-login with 'grn login --iam-env <env>'
instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := agentbaseconfig.EnvFromString(args[0])
		if err != nil {
			return err
		}
		profile := resolveProfile(cmd)
		shared, err := coreconfig.LoadConfig(profile)
		if err != nil {
			return err
		}
		// A user profile's iam_env is bound to the login token (the dev/prod
		// client selected at login). Repointing it here would invalidate the
		// refresh token; refuse and point the user at re-login. Machine
		// profiles switch freely.
		if shared.AuthMode == "user" {
			return fmt.Errorf("iam_env is bound to the login token on a user profile; re-login with 'grn login --iam-env %s' to switch", env)
		}
		if err := coreconfig.NewConfigFileWriter().WriteIamEnv(profile, string(env)); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Switched to environment: %s (iam_env written to profile %q)\n", env, profile)
		return nil
	},
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
	contextCmd.AddCommand(contextSwitchCmd)
	contextCmd.AddCommand(contextCurrentCmd)
	contextCmd.AddCommand(contextHeadersCmd)
	contextCmd.AddCommand(contextDecoratorsCmd)
}
