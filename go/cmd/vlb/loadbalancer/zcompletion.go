package loadbalancer

import "github.com/greennodehub/greennode-cli/internal/cli"

// --type means the OSI layer here; the certificate package uses the same flag
// name for something else, so the values are wired per command.
func init() {
	createCmd.RegisterFlagCompletionFunc("type", cli.FlagValues("Layer 4", "Layer 7")) //nolint:errcheck
}
