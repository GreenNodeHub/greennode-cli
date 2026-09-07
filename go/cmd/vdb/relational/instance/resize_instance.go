package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var resizeInstanceCmd = &cobra.Command{
	Use:   "resize-instance",
	Short: "Change the flavor (vCPU/RAM) of a Relational Database instance",
	Long: "Move an instance to a different flavor.\n\n" +
		"THIS COSTS MONEY: resizing goes through the order/payment flow and completes " +
		"asynchronously; the database restarts on the new flavor. Use --dry-run first.\n\n" +
		"--package-id is the numeric flavor id from 'catalog list-flavors'. Storage is " +
		"resized separately with 'instance resize-storage'.",
	Args: cobra.NoArgs,
	RunE: runResizeInstance,
}

func init() {
	f := resizeInstanceCmd.Flags()
	f.String("instance-id", "", "Database instance ID (required)")
	f.String("package-id", "", "New flavor id, numeric (required; the 'id' column of 'catalog list-flavors')")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")

	resizeInstanceCmd.MarkFlagRequired("instance-id") //nolint:errcheck
	resizeInstanceCmd.MarkFlagRequired("package-id")  //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	resizeInstanceCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
	// --package-id has no completer here on purpose. The flavors endpoint needs an
	// engine and version, which this command does not take as flags; deriving them
	// would mean fetching the instance inside the completion timeout, and that
	// listing is the slowest endpoint in the API. Run 'catalog list-flavors' instead.
}

func runResizeInstance(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	packageID, _ := cmd.Flags().GetString("package-id")
	poc, _ := cmd.Flags().GetBool("poc")

	// Note ResizeBody, not ActionBody: the resize endpoints spell the resource-type
	// field "resourceType" where the action endpoints use "resType".
	body := vdbclient.ResizeBody(vdbclient.ResourceTypeInstance, instanceID, "resize", map[string]interface{}{
		"packageId": packageID,
		"poc":       poc,
	})

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("resize", fmt.Sprintf("database instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Move database instance %s to flavor %s? This places a paid order and restarts the database.",
		instanceID, packageID)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(instancePath(instanceID, "/resize-instance"), body)
	if err != nil {
		return fmt.Errorf("failed to resize database instance %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}
