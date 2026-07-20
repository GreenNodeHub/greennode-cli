// Package agentbase implements the `grn agentbase` subcommand group for the
// GreenNode AgentBase platform.
//
// It is intentionally self-contained: it carries its own OAuth2 (v2
// client-credentials) auth, its own ./.greennode.json config, and its own HTTP
// client. It does NOT share state with grn's core (the v1 TokenManager, the INI
// config in ~/.greennode, or the retry/refresh HTTP client). The two stacks are
// fully independent.
//
// Compiled in ONLY with `-tags agentbase`. The default grn binary and the
// public release build (`-tags vks_only`) both exclude it while agentbase is
// still under development.
package agentbase

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/cliinput"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
	"github.com/vngcloud/greennode-cli/internal/cli"
)

// Persistent-flag targets for the `grn agentbase` subtree. The --output flag
// shadows grn's inherited root --output for this subtree only (cobra lets a
// child flag shadow an inherited persistent flag); the other grn root flags are
// inherited but inert — agentbase never reads them.
var (
	interactiveMode bool
	envOverride     string
	outputFormat    string
)

const greennodeASCIIArt = `
   _____ _____  ______ ______ _   _ _   _  ____  _____  ______
  / ____|  __ \|  ____|  ____| \ | | \ | |/ __ \|  __ \|  ____|
 | |  __| |__) | |__  | |__  |  \| |  \| | |  | | |  | | |__
 | | |_ |  _  /|  __| |  __| | . ` + "`" + ` | . ` + "`" + ` | |  | | |  | |  __|
 | |__| | | \ \| |____| |____| |\  | |\  | |__| | |__| | |____
  \_____|_|  \_\______|______|_| \_|_| \_|\____/|_____/|______| AGENTBASE
`

func printBanner() {
	color.New(color.FgGreen, color.Bold).Fprint(os.Stderr, greennodeASCIIArt)
}

// skipBannerCommands suppresses the ASCII banner for non-product commands.
var skipBannerCommands = map[string]bool{
	"help":       true,
	"completion": true,
}

// AgentbaseCmd is the `grn agentbase` subcommand. Its init() self-registers it
// with grn's service registry (cli.RegisterService), mirroring cmd/vks—so
// mounting requires no edit to root.go or main.go, only a build-tagged blank
// import in cmd/register_agentbase.go.
var AgentbaseCmd = &cobra.Command{
	Use:           "agentbase",
	Short:         "GreenNode AgentBase platform",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Manage the GreenNode AgentBase platform: agent identities and outbound
authentication providers (Phase 1). Runtime, memory, and deploy commands arrive
in later phases.

Configuration is read from ./.greennode.json in the current working directory
(separate from grn's ~/.greennode profile config). Run 'grn agentbase context
current' to see the active environment and endpoints.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetFormat(output.ParseFormat(outputFormat))
		if !skipBannerCommands[cmd.Name()] && output.GetFormat() == output.FormatTable {
			printBanner()
		}
		cliinput.SetInteractive(interactiveMode)
	},
}

func init() {
	AgentbaseCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Prompt for missing inputs instead of requiring flags")
	AgentbaseCmd.PersistentFlags().StringVar(&envOverride, "env", "", `Target environment: "dev" or "prod" (overrides GREENNODE_ENV and ./.greennode.json)`)
	AgentbaseCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", `Output format: "table", "json", or "id"`)

	cli.RegisterService(AgentbaseCmd)
}
