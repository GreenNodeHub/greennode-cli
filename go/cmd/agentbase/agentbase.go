// Package agentbase implements AgentBase commands using shared profiles and auth.
package agentbase

import (
	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	"github.com/greennodehub/greennode-cli/internal/cli"
)

// AgentBase overrides output; profile remains inherited.
var (
	interactiveMode bool
	outputFormat    string
	showSecret      bool
)

// AgentbaseCmd self-registers with the service registry.
var AgentbaseCmd = &cobra.Command{
	Use:           "agentbase",
	Short:         "GreenNode AgentBase platform",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Manage AgentBase identities, authentication providers, gateways,
runtimes, memory, policies, registry, marketplace, and deployments.

agentbase shares the ~/.greennode profile with the rest of the CLI. Configure
machine credentials with 'grn configure' (or log in as a user with 'grn login');
select dev/prod with 'grn configure set iam_env <dev|prod>' (machine) or
'grn login --iam-env <env>' (user); set the current agent with 'grn agentbase
access agent-id use <name>'. Run 'grn agentbase context current' to see the
active environment and endpoints.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Use configured output unless explicitly overridden.
		output.SetFormat(output.ParseFormat(effectiveOutputFormat(cmd, outputFormat)))
		output.SetShowSecret(showSecret)
		cliinput.SetInteractive(interactiveMode && !cli.IsNonInteractive())
	},
}

func init() {
	AgentbaseCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for missing inputs instead of requiring flags")
	AgentbaseCmd.PersistentFlags().BoolVar(&showSecret, "show-secret", false, "Reveal credentials in command output")
	AgentbaseCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", `Output format: "table", "json", or "id"`)

	cli.RegisterService(AgentbaseCmd)
}
