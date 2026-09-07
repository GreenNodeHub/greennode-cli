package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var resizeStorageCmd = &cobra.Command{
	Use:   "resize-storage",
	Short: "Change the storage size or type of a Relational Database instance",
	Long: "Grow an instance's volume, or move it to a different volume type.\n\n" +
		"THIS COSTS MONEY: resizing goes through the order/payment flow and completes " +
		"asynchronously. Storage can normally only grow. Use --dry-run first.\n\n" +
		"The API expects BOTH the size and the type on every request, using the current " +
		"value for whichever is not changing — so this command reads the instance first " +
		"and fills in the field you did not pass. --dry-run shows the resolved request.",
	Args: cobra.NoArgs,
	RunE: runResizeStorage,
}

// resizeStorageFlags defines the flags on f; a function so tests can build a
// throwaway command with the same set instead of mutating the mounted one.
func resizeStorageFlags(f *pflag.FlagSet) {
	f.String("instance-id", "", "Database instance ID (required)")
	f.Int("volume-size", 0, "New volume size in GB (default: unchanged)")
	f.String("volume-type", "", "New volume type NAME (default: unchanged; see 'catalog list-volume-types')")
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	resizeStorageFlags(resizeStorageCmd.Flags())

	resizeStorageCmd.MarkFlagRequired("instance-id") //nolint:errcheck

	// Bound here, next to the flag: see the init-order note in completion.go.
	resizeStorageCmd.RegisterFlagCompletionFunc("instance-id", relationalInstanceIDsFunc()) //nolint:errcheck
	resizeStorageCmd.RegisterFlagCompletionFunc("volume-type", volumeTypeCompletion())      //nolint:errcheck
}

func runResizeStorage(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")
	if err := requireRelationalID(instanceID); err != nil {
		return err
	}

	flags := cmd.Flags()
	if !flags.Changed("volume-size") && !flags.Changed("volume-type") {
		return fmt.Errorf("nothing to resize: pass --volume-size and/or --volume-type")
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	instance, err := fetchInstance(apiClient, instanceID)
	if err != nil {
		return err
	}

	body, summary, err := storageBody(flags, instance, instanceID)
	if err != nil {
		return err
	}

	dryRun, _ := flags.GetBool("dry-run")
	force, _ := flags.GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("resize", fmt.Sprintf("the storage of database instance %s", instanceID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Resize the storage of database instance %s (%s)? This places a paid order.", instanceID, summary)) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := apiClient.Post(instancePath(instanceID, "/resize-storage"), body)
	if err != nil {
		return fmt.Errorf("failed to resize the storage of database instance %s: %w", instanceID, err)
	}

	return vdbclient.Output(cmd, result)
}

// storageBody assembles ResizeVolumeRequest, taking the unchanged field from the
// instance, and returns a summary of the change for the prompt.
//
// Both fields are always sent: the API documents "if unchanged, use the current
// volume size/type of the DB instance", so omitting one is not the same as leaving
// it alone.
func storageBody(flags *pflag.FlagSet, instance map[string]interface{}, instanceID string) (map[string]interface{}, string, error) {
	currentSize := intField(instance, "volumeSize")
	currentType := stringField(instance, "volumeType")

	size := intOrDefault(flags, "volume-size", currentSize)
	volumeType := stringOrDefault(flags, "volume-type", currentType)
	poc, _ := flags.GetBool("poc")

	if size <= 0 {
		return nil, "", fmt.Errorf("invalid volume size %d for database instance %s: pass --volume-size", size, instanceID)
	}
	if volumeType == "" {
		return nil, "", fmt.Errorf("could not read the current volume type of database instance %s: pass --volume-type", instanceID)
	}
	if size < currentSize {
		return nil, "", fmt.Errorf("cannot shrink storage: instance %s is at %d GB and --volume-size is %d",
			instanceID, currentSize, size)
	}

	var summary string
	switch {
	case size != currentSize && volumeType != currentType:
		summary = fmt.Sprintf("%d -> %d GB, %s -> %s", currentSize, size, currentType, volumeType)
	case size != currentSize:
		summary = fmt.Sprintf("%d -> %d GB, volume type unchanged (%s)", currentSize, size, currentType)
	case volumeType != currentType:
		summary = fmt.Sprintf("%s -> %s, size unchanged (%d GB)", currentType, volumeType, currentSize)
	default:
		return nil, "", fmt.Errorf("nothing would change: instance %s is already %d GB on %s",
			instanceID, currentSize, currentType)
	}

	return vdbclient.ResizeBody(vdbclient.ResourceTypeInstance, instanceID, "resize", map[string]interface{}{
		"volumeSize": size,
		"volumeType": volumeType,
		"poc":        poc,
	}), summary, nil
}
