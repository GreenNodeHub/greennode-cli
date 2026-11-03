package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// The start / stop / reboot endpoints differ only in their path segment, their
// action value and their wording, so they are declared here rather than in three
// near-identical files. delete has its own file: it takes options and prints the
// instance it is about to destroy.
//
// The API's shutdown endpoint is exposed as `stop`, the canonical CLI verb —
// `conventions_test.go` requires --dry-run and --force for it.
type action struct {
	use     string
	path    string // path segment
	name    string // "action" value in the body; each endpoint accepts only its own
	short   string
	long    string
	confirm string // confirmation prompt; empty means do not prompt
}

var actions = []action{
	{
		use:   "start",
		path:  "/start",
		name:  "start",
		short: "Start a stopped Relational Database instance",
		long: "Start an instance that was stopped with 'instance stop'.\n\n" +
			"Billing for a stopped instance continues for its storage, so starting one " +
			"does not change what you pay. No confirmation is asked.",
	},
	{
		use:   "stop",
		path:  "/shutdown",
		name:  "stop",
		short: "Stop a Relational Database instance",
		long: "Stop the database. The instance keeps its storage and its IP, and can be " +
			"restarted with 'instance start'.\n\n" +
			"Applications lose their connections immediately, so the command confirms first. " +
			"Note the API calls this endpoint 'shutdown'; the CLI verb is 'stop'.",
		confirm: "Stop database instance %s? Applications will lose their connections.",
	},
	{
		use:     "reboot",
		path:    "/reboot",
		name:    "reboot",
		short:   "Reboot a Relational Database instance",
		long:    "Restart the database. Open connections are dropped while it restarts.",
		confirm: "Reboot database instance %s? Open connections will be dropped.",
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
	f.String("instance-id", "", "Database instance ID (required)")
	f.Bool("dry-run", false, fmt.Sprintf("Print the request that would be sent without %sing", spec.use))
	f.Bool("force", false, "Skip the confirmation prompt")

	cmd.MarkFlagRequired("instance-id")                                        //nolint:errcheck
	cmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck

	return cmd
}

func runAction(cmd *cobra.Command, spec action) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	body := vdbclient.ActionBody(vdbclient.ResourceTypeInstance, instanceID, spec.name, nil)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody(spec.use, fmt.Sprintf("database instance %s", instanceID), body)
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
		return fmt.Errorf("failed to %s database instance %s: %w", spec.use, instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}
