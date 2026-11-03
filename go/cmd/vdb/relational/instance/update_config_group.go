package instance

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var updateConfigGroupCmd = &cobra.Command{
	Use:   "update-config-group",
	Short: "Attach or detach the config group of a Relational Database instance",
	Long: "Attach a config group to an instance, or detach the current one with " +
		"--detach.\n\n" +
		"The config group must match the instance's engine and version — see " +
		"'catalog list-config-groups'. Applying configuration parameters can restart the " +
		"database, so the command confirms first.\n\n" +
		"The change is ASYNCHRONOUS and its response body carries nothing useful, so this " +
		"command reports acceptance and points you at 'instance get'.",
	Args: cobra.NoArgs,
	RunE: runUpdateConfigGroup,
}

func init() {
	f := updateConfigGroupCmd.Flags()
	f.String("instance-id", "", "Database instance ID (required)")
	f.String("config-id", "", "Config group ID to attach (required unless --detach)")
	f.Bool("detach", false, "Detach the current config group instead of attaching one")
	f.Bool("dry-run", false, "Print the request that would be sent without applying it")
	f.Bool("force", false, "Skip the confirmation prompt")

	updateConfigGroupCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	// Mutually exclusive rather than "empty means detach": the API detaches on an
	// empty string, which is far too easy to send by accident from a script whose
	// variable is unset.
	updateConfigGroupCmd.MarkFlagsMutuallyExclusive("config-id", "detach")

	// Bound here, next to the flags: see the init-order note in completion.go.
	updateConfigGroupCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
	updateConfigGroupCmd.RegisterFlagCompletionFunc("config-id", configIDCompletion())          //nolint:errcheck
}

func runUpdateConfigGroup(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	configID, _ := cmd.Flags().GetString("config-id")
	detach, _ := cmd.Flags().GetBool("detach")

	if !detach && configID == "" {
		return fmt.Errorf("pass --config-id to attach a config group, or --detach to remove the current one")
	}

	// dbInstanceId is required in the body as well as the path.
	//
	// Detaching sends configId **null**, not the empty string the spec describes:
	// `""` is rejected with `The configId  doesn't exist` (HTTP 400), while null
	// detaches and leaves the instance RESTART_REQUIRED. Verified against the live
	// API on 2026-08-13 — do not "fix" this back to the spec.
	body := map[string]interface{}{"dbInstanceId": instanceID}
	if detach {
		body["configId"] = nil
	} else {
		body["configId"] = configID
	}

	summary := fmt.Sprintf("attach config group %s", configID)
	if detach {
		summary = "detach the config group"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		previewAsyncUpdate("update", fmt.Sprintf("the config group of database instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf("%s on database instance %s? This may restart the database.",
		capitalize(summary), instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	warnIfNotActive(apiClient, instanceID)

	if _, err := apiClient.Put(instancePath(instanceID, "/update/config-group"), body); err != nil {
		return fmt.Errorf("failed to update the config group of database instance %s: %w", instanceID, err)
	}

	reportAsyncAccepted(cmd, instanceID, summary)
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
