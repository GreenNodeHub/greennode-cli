package vdbclient

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
)

// sensitiveKeys never reach the terminal in a preview. vDB carries master
// passwords in create, replica-create and update-setting bodies, and --dry-run
// output is exactly the kind of thing that gets pasted into a ticket.
var sensitiveKeys = map[string]bool{"password": true, "redisPassword": true}

// PreviewBody prints the request body a command would send, with sensitive values
// masked, followed by the standard dry-run footer.
//
// cli.PrintDryRun covers flat bodies; every mutating vdb body nests (user,
// databases, netIds, databaseInstances[].config), so these are shown as indented
// JSON — which doubles as something the user can replay against the API.
func PreviewBody(verb, target string, body interface{}) {
	previewBodyTo(os.Stdout, verb, target, body)
	cli.DryRunNotice(verb)
}

// previewBodyTo writes just the preview (no footer) so tests can capture it.
func previewBodyTo(w io.Writer, verb, target string, body interface{}) {
	fmt.Fprintln(w, "=== DRY RUN ===")
	if target != "" {
		fmt.Fprintf(w, "Would %s %s with:\n", verb, target)
	}

	encoded, err := json.MarshalIndent(maskSecrets(body), "", "  ")
	if err != nil {
		// Never block a preview on formatting; fall back to Go's own rendering.
		fmt.Fprintf(w, "  %v\n", body)
	} else {
		fmt.Fprintln(w, string(encoded))
	}
}

// maskSecrets returns a deep copy of v with every sensitive value replaced. It
// copies rather than editing in place so the real request body is untouched.
func maskSecrets(v interface{}) interface{} {
	switch value := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(value))
		for k, item := range value {
			if sensitiveKeys[k] {
				out[k] = "***"
				continue
			}
			out[k] = maskSecrets(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(value))
		for i, item := range value {
			out[i] = maskSecrets(item)
		}
		return out
	default:
		return v
	}
}
