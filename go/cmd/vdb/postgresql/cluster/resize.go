package cluster

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resizeTypes are the API's three resize modes. Each one reads exactly one of the
// new-value flags and ignores the others, so the command requires the matching
// flag instead of letting a silent no-op order go through.
var resizeTypes = []string{"VOLUME-SIZE", "VOLUME-TYPE", "NUMBER-OF-NODES"}

// resizeFlagFor maps a resize type to the flag that carries its new value.
var resizeFlagFor = map[string]string{
	"VOLUME-SIZE":     "volume-size",
	"VOLUME-TYPE":     "volume-type-id",
	"NUMBER-OF-NODES": "number-of-nodes",
}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a PostgreSQL Cluster",
	Long: "Change a cluster's storage size, storage type or node count.\n\n" +
		"THIS COSTS MONEY: like create, resize goes through the order/payment flow and " +
		"completes asynchronously. Use --dry-run first.\n\n" +
		"One resize does one thing: --type picks which, and only that dimension's flag " +
		"is read (VOLUME-SIZE reads --volume-size, VOLUME-TYPE reads --volume-type-id, " +
		"NUMBER-OF-NODES reads --number-of-nodes). Storage can normally only grow.",
	Args: cobra.NoArgs,
	RunE: runResize,
}

// resizeFlagsOn defines the flags on f; see the note on createFlags.
func resizeFlagsOn(f *pflag.FlagSet) {
	f.String("cluster-id", "", "PostgreSQL Cluster ID (required)")
	f.String("type", "", fmt.Sprintf("What to resize: %s (required)", strings.Join(resizeTypes, ", ")))
	f.Int("volume-size", 0, "New volume size in GB (with --type VOLUME-SIZE)")
	f.String("volume-type-id", "", "New volume type ID, 'pgst-...' (with --type VOLUME-TYPE)")
	f.Int("number-of-nodes", 0, fmt.Sprintf("New node count, %d-%d (with --type NUMBER-OF-NODES)", minNodes, maxNodes))
	f.Bool("poc", false, "Pay with PoC credit (Auto Payment only)")
	f.Bool("dry-run", false, "Print the request that would be sent without placing an order")
	f.Bool("force", false, "Skip the confirmation prompt")
}

func init() {
	resizeFlagsOn(resizeCmd.Flags())

	resizeCmd.MarkFlagRequired("cluster-id") //nolint:errcheck
	resizeCmd.MarkFlagRequired("type")       //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	resizeCmd.RegisterFlagCompletionFunc("cluster-id", clusterIDCompletion())        //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("type", cli.FlagValues(resizeTypes...))     //nolint:errcheck
	resizeCmd.RegisterFlagCompletionFunc("volume-type-id", volumeTypeIDCompletion()) //nolint:errcheck
}

func runResize(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if err := requireClusterID(clusterID); err != nil {
		return err
	}

	body, summary, err := resizeBody(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if dryRun {
		vdbclient.PreviewBody("resize", fmt.Sprintf("PostgreSQL Cluster %s", clusterID), body)
		return nil
	}
	if !cli.Confirm(force, fmt.Sprintf(
		"Resize PostgreSQL Cluster %s (%s)? This places a paid order.", clusterID, summary)) {
		fmt.Println("Aborted.")
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(pgPath(clusterID, "/resize"), body)
	if err != nil {
		return fmt.Errorf("failed to resize PostgreSQL Cluster %s: %w", clusterID, err)
	}

	return vdbclient.Output(cmd, result)
}

// resizeBody assembles ResizePostgreClusterRequest and returns a one-line summary
// of the change for the confirmation prompt.
func resizeBody(cmd *cobra.Command) (map[string]interface{}, string, error) {
	flags := cmd.Flags()

	resizeType, _ := flags.GetString("type")
	resizeType = strings.ToUpper(resizeType)

	flagName, ok := resizeFlagFor[resizeType]
	if !ok {
		return nil, "", fmt.Errorf("invalid --type '%s': must be one of %s",
			resizeType, strings.Join(resizeTypes, ", "))
	}

	poc, _ := flags.GetBool("poc")
	body := map[string]interface{}{"type": resizeType, "isPoc": poc}

	var summary string
	switch resizeType {
	case "VOLUME-SIZE":
		size, _ := flags.GetInt("volume-size")
		if size <= 0 {
			return nil, "", missingResizeFlag(resizeType, flagName)
		}
		body["volumeSize"] = size
		summary = fmt.Sprintf("volume size -> %d GB", size)
	case "VOLUME-TYPE":
		volumeTypeID, _ := flags.GetString("volume-type-id")
		if volumeTypeID == "" {
			return nil, "", missingResizeFlag(resizeType, flagName)
		}
		body["volumeTypeId"] = volumeTypeID
		summary = fmt.Sprintf("volume type -> %s", volumeTypeID)
	case "NUMBER-OF-NODES":
		nodes, _ := flags.GetInt("number-of-nodes")
		// Distinguish "flag not given" from "given out of range", so the error
		// names the flag the user actually forgot.
		if nodes == 0 {
			return nil, "", missingResizeFlag(resizeType, flagName)
		}
		if nodes < minNodes || nodes > maxNodes {
			return nil, "", fmt.Errorf("invalid --number-of-nodes %d: a PostgreSQL Cluster takes %d to %d nodes",
				nodes, minNodes, maxNodes)
		}
		body["numberOfNodes"] = nodes
		summary = fmt.Sprintf("node count -> %d", nodes)
	}

	return body, summary, nil
}

func missingResizeFlag(resizeType, flagName string) error {
	return fmt.Errorf("--type %s requires --%s", resizeType, flagName)
}
