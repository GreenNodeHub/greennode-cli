package catalog

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

// flavorColumns omits volumeSize and monthlyCost, which the API returns as 0, and
// the pricing SKUs. ram is the number that matters for Redis: it is the instance's
// capacity.
var flavorColumns = []string{
	"id", "name", "vcpus", "ram", "bandwidth", "familyType", "platformType", "locateZoneId",
}

var listFlavorsCmd = &cobra.Command{
	Use:   "list-flavors",
	Short: "List flavors for an engine version",
	Long: "List the flavors (vCPU/RAM combinations) available for one engine version.\n\n" +
		"Both --datastore-type and --datastore-version are required by the API; run " +
		"'catalog list-datastores' for the valid pairs. Note the flag is " +
		"--datastore-version, not --version, which is reserved globally.\n\n" +
		"The 'id' column is what --package-id expects on 'instance create' and " +
		"'instance resize-instance'. RAM is the instance's capacity — a MemoryStore " +
		"instance has no volume.",
	Args: cobra.NoArgs,
	RunE: runListFlavors,
}

func init() {
	f := listFlavorsCmd.Flags()
	f.String("datastore-type", "", "Engine, e.g. Redis (required)")
	f.String("datastore-version", "", "Engine version, e.g. 7.2 (required)")
	f.String("zone-id", "", "Restrict results to one availability zone")

	listFlavorsCmd.MarkFlagRequired("datastore-type")    //nolint:errcheck
	listFlavorsCmd.MarkFlagRequired("datastore-version") //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-type", datastoreTypeCompletion())       //nolint:errcheck
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-version", datastoreVersionCompletion()) //nolint:errcheck
	listFlavorsCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())                     //nolint:errcheck
}

// flavorsQuery maps the flags onto the endpoint's query params. The API names them
// "type" and "version"; the flags are prefixed because --version is taken by the
// root command.
func flavorsQuery(cmd *cobra.Command) url.Values {
	query := zoneQuery(cmd)
	datastoreType, _ := cmd.Flags().GetString("datastore-type")
	datastoreVersion, _ := cmd.Flags().GetString("datastore-version")
	query.Set("type", datastoreType)
	query.Set("version", datastoreVersion)
	return query
}

func runListFlavors(cmd *cobra.Command, args []string) error {
	if err := get(cmd, flavorsPath, flavorsQuery(cmd), flavorColumns); err != nil {
		return fmt.Errorf("failed to list flavors: %w", err)
	}
	return nil
}
