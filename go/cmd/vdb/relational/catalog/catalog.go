package catalog

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CatalogCmd is the parent command for the Relational Database lookup lists.
var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Look up Relational Database engines, flavors, zones and networks",
	Long: "Read-only lists of what a Relational Database instance can be built from: " +
		"engines, datastore versions, instance families, flavors, availability zones, " +
		"networks and volume types.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

// simpleList describes a catalog listing that needs no input beyond an optional
// zone filter. Eight of them differ only in path, wording and column set, so they
// are declared here instead of in eight near-identical files; list-flavors, the
// one taking required arguments, has its own file.
type simpleList struct {
	use     string
	short   string
	long    string
	path    string
	byZone  bool // offer --zone-id, sent as ?zoneId=
	columns []string
}

var simpleLists = []simpleList{
	{
		use:   "list-engines",
		short: "List available database engines",
		long: "List the database engines offered by Relational Database (MySQL, MariaDB, " +
			"PostgreSQL) with their licences.\n\n" +
			"Use 'catalog list-datastores' for the engine/version pairs that can actually " +
			"be deployed.",
		path: enginePath,
		// image comes back empty on every engine and engineLicenses is a nested
		// array; both are left out of the table but kept in JSON output.
		columns: []string{"name", "description"},
	},
	{
		use:   "list-datastores",
		short: "List available datastore types and versions",
		long: "List the engine/version pairs that can be deployed.\n\n" +
			"The 'type' and 'version' values are what --datastore-type and " +
			"--datastore-version expect on 'catalog list-flavors'.",
		path:    datastorePath,
		columns: []string{"type", "version", "name", "versionName", "licenseName"},
	},
	{
		use:   "list-families",
		short: "List instance families",
		long: "List the instance families (CPU/memory profiles) that flavors belong to.\n\n" +
			"The response mixes two kinds of row, told apart by the group field: " +
			"'family' rows are the real families (key + value set, condition.codes " +
			"listing their CPU platform codes), while 'family_custom' rows are custom " +
			"zones carrying only a name.",
		path:    familiesPath,
		columns: []string{"group", "key", "value", "name", "description"},
	},
	{
		use:     "list-flavor-codes",
		short:   "List CPU platform codes",
		long:    "List the CPU platform codes that flavors are grouped by.",
		path:    flavorCodesPath,
		columns: []string{"key", "value", "familyType", "description"},
	},
	{
		use:     "list-zones",
		short:   "List availability zones",
		long:    "List the availability zones a Relational Database instance can be placed in.",
		path:    zonesPath,
		columns: []string{"uuid", "name", "status", "zoneType", "isDefault", "description"},
	},
	{
		use:     "list-networks",
		short:   "List networks available to database instances",
		long:    "List the networks (VPCs) a Relational Database instance can be attached to.",
		path:    networksPath,
		columns: []string{"id", "displayName", "cidr", "status", "createdAt"},
	},
	{
		use:   "list-subnets",
		short: "List networks with their subnets",
		long: "List the networks available to database instances together with their " +
			"subnets, optionally restricted to one availability zone.\n\n" +
			"Each row is a network; its subnets are a nested array, so use " +
			"--output json (or --query) to see them.",
		path:    subnetsPath,
		byZone:  true,
		columns: []string{"uuid", "displayName", "status", "networkId", "zoneId"},
	},
	{
		use:   "list-volume-types",
		short: "List volume types",
		long: "List the volume types available for database instance storage, with their " +
			"size limits and provisioned IOPS.\n\n" +
			"Unlike the other catalog listings this one wraps its array in an object, so a " +
			"--query has to go through that key: --query 'data[].type'.",
		path:   volumeTypesPath,
		byZone: true,
		// displayName is null on every row; the human-readable label is description.
		columns: []string{"type", "description", "minVolumeSize", "maxVolumeSize", "iops", "zoneId"},
	},
}

func newSimpleList(spec simpleList) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := get(cmd, spec.path, zoneQuery(cmd), spec.columns); err != nil {
				// "list-volume-types" -> "failed to list volume types: ..."
				return fmt.Errorf("failed to %s: %w", strings.ReplaceAll(spec.use, "-", " "), err)
			}
			return nil
		},
	}
	if spec.byZone {
		cmd.Flags().String("zone-id", "", "Restrict results to one availability zone (run 'catalog list-zones' for IDs)")
		cmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion()) //nolint:errcheck
	}
	return cmd
}

func init() {
	for _, spec := range simpleLists {
		CatalogCmd.AddCommand(newSimpleList(spec))
	}
	CatalogCmd.AddCommand(listFlavorsCmd)
	CatalogCmd.AddCommand(listConfigGroupsCmd)
}
