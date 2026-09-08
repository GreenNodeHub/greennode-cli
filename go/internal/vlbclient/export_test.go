package vlbclient

import "github.com/greennodehub/greennode-cli/internal/cli"

// extractForTest exposes the completion extraction the completers use, so the
// round-trip tests can pin the envelope key each of them passes.
func extractForTest(result interface{}, listKeys []string) []string {
	return cli.ExtractCompletions(result, listKeys, "uuid", "name")
}
