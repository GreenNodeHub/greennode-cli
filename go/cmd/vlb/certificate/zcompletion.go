package certificate

import "github.com/greennodehub/greennode-cli/internal/cli"

func init() {
	createCmd.RegisterFlagCompletionFunc("type", cli.FlagValues("CA", "TLS/SSL")) //nolint:errcheck
}
