package agentbase

import (
	"fmt"

	"github.com/spf13/cobra"
)

// requireSubcommand is the RunE for group commands — commands that only own
// subcommands ('grn agentbase access', 'grn agentbase runtime', ...).
//
// Without a Run/RunE, cobra treats a non-root group as not runnable: a bare
// invocation prints help and exits 0, and — worse — an unknown subcommand word
// also prints help and exits 0 ('grn agentbase runtime bogus' looked like
// success to scripts and to the e2e runner, which treats exit 0 as PASS). This
// makes groups behave like the root command: bare invocation still prints help
// (exit 0, discoverability), but an unrecognized word is a hard error with the
// same wording cobra uses at the root.
func requireSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	return fmt.Errorf("unknown subcommand %q for %q\nRun '%s --help' for usage.", args[0], cmd.CommandPath(), cmd.CommandPath())
}
