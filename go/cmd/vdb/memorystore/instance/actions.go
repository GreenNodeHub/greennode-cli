package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// start / stop / reboot differ only in path segment, action value and wording. The
// API's endpoint for stop is /shutdown; the CLI verb is `stop`, which is also what
// conventions_test.go requires --dry-run and --force for.
type action struct {
	use     string
	path    string
	name    string
	short   string
	long    string
	confirm string // empty means do not prompt
}

var actions = []action{
	{
		use:   "start",
		path:  "/start",
		name:  "start",
		short: "Start a stopped MemoryStore instance",
		long: "Start an instance that was stopped with 'instance stop'. No confirmation is " +
			"asked.",
	},
	{
		use:   "stop",
		path:  "/shutdown",
		name:  "stop",
		short: "Stop a MemoryStore instance",
		long: "Stop the instance. It keeps its IP and can be restarted with " +
			"'instance start'.\n\n" +
			"Clients lose their connections immediately, and because Redis is an in-memory " +
			"store, anything not persisted by its own snapshot or AOF settings is gone. The " +
			"command confirms first. Note the API calls this endpoint 'shutdown'.",
		confirm: "Stop MemoryStore instance %s? Clients lose their connections and unpersisted data is lost.",
	},
	{
		use:     "reboot",
		path:    "/reboot",
		name:    "reboot",
		short:   "Reboot a MemoryStore instance",
		long:    "Restart the instance. Connections are dropped while it restarts.",
		confirm: "Reboot MemoryStore instance %s? Open connections will be dropped.",
	},
}

func newActionCmd(spec action) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, spec)
		},
	}

	f := cmd.Flags()
	f.String("instance-id", "", "MemoryStore instance ID (required)")
	f.Bool("dry-run", false, fmt.Sprintf("Print the request that would be sent without %sing", spec.use))
	f.Bool("force", false, "Skip the confirmation prompt")

	cmd.MarkFlagRequired("instance-id")                                   //nolint:errcheck
	cmd.RegisterFlagCompletionFunc("instance-id", instanceIDCompletion()) //nolint:errcheck

	return cmd
}

func runAction(cmd *cobra.Command, spec action) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireInstanceID(instanceID); err != nil {
		return err
	}

	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, instanceID, spec.name, nil)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody(spec.use, fmt.Sprintf("MemoryStore instance %s", instanceID), body)
		return nil
	}
	if spec.confirm != "" && !cli.Confirm(force, fmt.Sprintf(spec.confirm, instanceID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(instancePath(instanceID, spec.path), body)
	if err != nil {
		return fmt.Errorf("failed to %s MemoryStore instance %s: %w", spec.use, instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}
