package listener

import "github.com/greennodehub/greennode-cli/internal/cli"

// Listener protocols differ from pool protocols (no PROXY, but HTTPS), so the
// --protocol values are wired here rather than in the shared VLB map.
func init() {
	createCmd.RegisterFlagCompletionFunc("protocol", cli.FlagValues("HTTP", "HTTPS", "TCP", "UDP")) //nolint:errcheck
}
