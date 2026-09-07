package instance

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// The two update endpoints — update/setting and update/config-group — behave
// unlike anything else in vdb, so their reporting lives here rather than being
// repeated:
//
//   - They are asynchronous. The envelope comes back with code 200 while the
//     payload's own status is 202: the request was accepted, not applied.
//   - **Their response body is meaningless.** It carries only status, dbInstanceId
//     and projectId, and the last two arrive transposed (the project ID in
//     dbInstanceId and vice versa). The product team confirmed the body has no
//     meaning and the HTTP status is the whole result, so these commands print
//     their own confirmation instead of echoing the payload — printing it would
//     show the user a wrong instance ID.

// activeStatus is the only state in which the two async updates are actually
// applied. Status values are mixed-case in this API ("ACTIVE", "SHUTDOWN", but
// "stopping", "starting", "deleting"), so compare case-insensitively.
const activeStatus = "ACTIVE"

// previewAsyncUpdate is PreviewBody plus the async warning, so --dry-run says the
// same thing the real run will.
func previewAsyncUpdate(verb, target string, body map[string]interface{}) {
	vdbclient.PreviewBody(verb, target, body)
	fmt.Println("\nThe API applies this in the background; poll 'instance get' to confirm the result.")
}

// warnIfNotActive is the guard for a trap found on the live API: while an instance
// is RESTART_REQUIRED, these endpoints still answer 202 "success" and then **do
// nothing** — an attach can be issued, acknowledged, and silently dropped
// (2026-08-13). Since the failure leaves no trace anywhere, the command says so up
// front rather than letting the user believe a change landed.
//
// It warns instead of refusing: the instance may legitimately be in another state,
// and this is only the one case that was observed to swallow updates.
func warnIfNotActive(apiClient *vdbclient.Client, instanceID string) {
	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		// Not worth blocking the update over: the request itself will report any
		// real problem.
		return
	}

	status := stringField(instance, "status")
	if strings.EqualFold(status, activeStatus) {
		return
	}

	fmt.Printf("Warning: database instance %s is %s, not %s.\n", instanceID, status, activeStatus)
	fmt.Println("The API may accept this change and silently not apply it — this is known to")
	fmt.Println("happen while an instance is RESTART_REQUIRED. Consider 'instance reboot' first,")
	fmt.Println("then re-check with 'instance get'.")
	fmt.Println()
}

// reportAsyncAccepted is what these commands print instead of the response body.
func reportAsyncAccepted(cmd *cobra.Command, instanceID, summary string) {
	fmt.Printf("Accepted: %s on database instance %s.\n", summary, instanceID)
	fmt.Println("The API applies this asynchronously — run the following to confirm:")
	fmt.Printf("\n  grn vdb relational instance get --instance-id %s\n", instanceID)
}
