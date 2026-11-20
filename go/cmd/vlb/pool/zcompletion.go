package pool

import "github.com/greennodehub/greennode-cli/internal/cli"

// Pool protocols differ from listener protocols (PROXY, but no HTTPS).
func init() {
	createCmd.RegisterFlagCompletionFunc("protocol", cli.FlagValues("HTTP", "TCP", "UDP", "PROXY")) //nolint:errcheck
}
