package catalog

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

// flavorColumns omits the pricing fields (packageSku, priceKey, monthlyCost — the
// last comes back 0 on every flavor) and volumeSize, which is 0 here because
// storage is chosen separately. locateZoneId earns its place: the same flavor name
// is repeated once per zone with a different id. JSON output keeps all 19 fields.
var flavorColumns = []string{
	"id", "name", "vcpus", "ram", "volumeType", "bandwidth",
	"familyType", "platformType", "locateZoneId",
}

var listFlavorsCmd = &cobra.Command{
	Use:   "list-flavors",
	Short: "List flavors for a datastore type and version",
	Long: "List the flavors (vCPU/RAM combinations) available for one engine version.\n\n" +
		"Both --datastore-type and --datastore-version are required by the API; run " +
		"'catalog list-datastores' to see the valid pairs. --datastore-type takes either " +
		"the lowercase form listed there ('postgresql') or the display form " +
		"('PostgreSQL') — the API matches both. Note the flag is --datastore-version, " +
		"not --version, which is reserved globally.\n\n" +
		"The same flavor name appears once per availability zone; narrow with --zone-id.",
	Args: cobra.NoArgs,
	RunE: runListFlavors,
}

func init() {
	f := listFlavorsCmd.Flags()
	f.String("datastore-type", "", "Datastore type, e.g. MySQL (required)")
	f.String("datastore-version", "", "Datastore version, e.g. 8.0 (required)")
	f.String("zone-id", "", "Restrict results to one availability zone (run 'catalog list-zones' for IDs)")

	listFlavorsCmd.MarkFlagRequired("datastore-type")    //nolint:errcheck
	listFlavorsCmd.MarkFlagRequired("datastore-version") //nolint:errcheck

	// Bound here, next to the flags: see the init-order note in completion.go.
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-type", datastoreTypeCompletion())       //nolint:errcheck
	listFlavorsCmd.RegisterFlagCompletionFunc("datastore-version", datastoreVersionCompletion()) //nolint:errcheck
	listFlavorsCmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion())                     //nolint:errcheck
}

// flavorsQuery maps the flags onto the endpoint's query params. The API names
// them "type" and "version"; the flags are prefixed because --version is taken by
// the root command. Split out from runListFlavors so the mapping is testable
// without a configured client.
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
