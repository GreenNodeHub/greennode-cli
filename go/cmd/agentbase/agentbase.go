// Package agentbase implements the `grn agentbase` subcommand group for the
// GreenNode AgentBase platform.
//
// agentbase shares the rest of the CLI's auth and config: it reads the same
// ~/.greennode profile (creds, auth_mode, iam_env, agent_identity) and uses the
// same shared token providers (auth.MachineTokenProvider for machine mode,
// auth.LoginTokenProvider for user mode) via cli.NewTokenProvider — the exact
// selector vks/vserver use. The dev/prod environment is the profile's iam_env
// (default prod); the current agent identity is persisted per-profile (the
// agent_identity key). agentbase no longer carries its own .greennode.json.
//
// Compiled into the default grn binary and the public release build
// (`-tags vks_only`); the `-tags agentbase` opt-in gate was dropped when
// agentbase went generally available.
package agentbase

import (
	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	"github.com/greennodehub/greennode-cli/internal/cli"
)

// Persistent-flag targets for the `grn agentbase` subtree. The --output flag
// shadows grn's inherited root --output for this subtree only (cobra lets a
// child flag shadow an inherited persistent flag); --profile is inherited from
// the root and selects the shared ~/.greennode profile agentbase reads.
var (
	interactiveMode bool
	outputFormat    string
)

// AgentbaseCmd is the `grn agentbase` subcommand. Its init() self-registers it
// with grn's service registry (cli.RegisterService), mirroring cmd/vks—so
// mounting requires no edit to root.go or main.go, only a build-tagged blank
// import in cmd/register_agentbase.go.
var AgentbaseCmd = &cobra.Command{
	Use:           "agentbase",
	Short:         "GreenNode AgentBase platform",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Manage the GreenNode AgentBase platform: agent identities, outbound
authentication providers, and MCP gateways. Runtime, memory, and deploy commands
arrive in later phases.

agentbase shares the ~/.greennode profile with the rest of the CLI. Configure
machine credentials with 'grn configure' (or log in as a user with 'grn login');
select dev/prod with 'grn configure set iam_env <dev|prod>' (machine) or
'grn login --iam-env <env>' (user); set the current agent with 'grn agentbase
identity workload use <name>'. Run 'grn agentbase context current' to see the
active environment and endpoints.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetFormat(output.ParseFormat(outputFormat))
		cliinput.SetInteractive(interactiveMode)
	},
}

func init() {
	AgentbaseCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for missing inputs instead of requiring flags")
	AgentbaseCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", `Output format: "table", "json", or "id"`)

	cli.RegisterService(AgentbaseCmd)
}
