// Package catalog holds the MemoryStore read-only lookups.
//
// They live under `/vdb-memory/v1/database/*`, NOT `/database-instances/*` as in the
// relational API, and the set is not the same: MemoryStore has **no zones or subnets
// endpoint** — those come from the Relational Database catalog, which is where the
// product owner confirmed they live.
//
// `/database/status` is deliberately unused: the product owner reported it OUTDATED
// on 2026-08-13, so neither a command nor completion reads it. Status values are
// taken from the instance listing instead, as the relational group does.
package catalog

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// CatalogCmd is the parent command for the MemoryStore lookups.
var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Look up MemoryStore engines, flavors, networks and config groups",
	Long: "Read-only lists of what a MemoryStore instance can be built from: engines, " +
		"versions, instance families, flavors, CPU platform codes, networks, volume types " +
		"and config groups.\n\n" +
		"Zones and subnets are NOT here: MemoryStore has no endpoint for them, so use " +
		"'grn vdb relational catalog list-zones' and 'list-subnets' — they are " +
		"project-wide.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

const (
	basePath = "/vdb-memory/v1/database"

	datastorePath   = basePath + "/datastore"
	enginePath      = basePath + "/engine"
	familiesPath    = basePath + "/families"
	codesPath       = basePath + "/codes"
	flavorsPath     = basePath + "/flavors"
	networksPath    = basePath + "/networks"
	subnetsPath     = basePath + "/networks/subnets"
	volumeTypesPath = basePath + "/volume-types"
	configPath      = basePath + "/configuration"
)

// simpleList describes a lookup needing no input beyond an optional zone.
type simpleList struct {
	use     string
	short   string
	long    string
	path    string
	byZone  bool
	columns []string
}

var simpleLists = []simpleList{
	{
		use:     "list-engines",
		short:   "List available engines",
		long:    "List the engines MemoryStore offers. In practice this is Redis.",
		path:    enginePath,
		columns: []string{"name", "description"},
	},
	{
		use:   "list-datastores",
		short: "List available engine versions",
		long: "List the engine/version pairs that can be deployed. The 'version' value is " +
			"what --datastore-version expects on 'instance create'.",
		path:    datastorePath,
		columns: []string{"type", "version", "name", "versionName", "licenseName"},
	},
	{
		use:     "list-families",
		short:   "List instance families",
		long:    "List the instance families (CPU/memory profiles) that flavors belong to.",
		path:    familiesPath,
		columns: []string{"group", "key", "value", "name", "description"},
	},
	{
		use:     "list-flavor-codes",
		short:   "List CPU platform codes",
		long:    "List the CPU platform codes that flavors are grouped by.",
		path:    codesPath,
		columns: []string{"key", "value", "familyType", "description"},
	},
	{
		use:     "list-networks",
		short:   "List networks available to instances",
		long:    "List the networks (VPCs) a MemoryStore instance can be attached to.",
		path:    networksPath,
		columns: []string{"id", "displayName", "cidr", "status", "createdAt"},
	},
	{
		use:   "list-subnets",
		short: "List networks with their subnets",
		long: "List the networks available to MemoryStore together with their subnets.\n\n" +
			"Each row is a network; its subnets are a nested array, so use --output json " +
			"(or --query) to see them. This mirrors the relational endpoint of the same " +
			"name, which is also where 'instance create --subnet-ids' values come from.",
		path:    subnetsPath,
		byZone:  true,
		columns: []string{"uuid", "displayName", "status", "networkId", "zoneId"},
	},
	{
		use:   "list-volume-types",
		short: "List volume types",
		long: "List the volume types the API reports for MemoryStore.\n\n" +
			"Informational: a MemoryStore instance takes no volume type or size — its " +
			"capacity is the flavor's RAM — so nothing on 'instance create' consumes these.",
		path:    volumeTypesPath,
		byZone:  true,
		columns: []string{"type", "description", "minVolumeSize", "maxVolumeSize", "iops", "zoneId"},
	},
	{
		use:   "list-config-groups",
		short: "List config groups that can be attached to an instance",
		long: "List the config groups available to MemoryStore instances.\n\n" +
			"A group only fits an instance with the same engine and version. Its ID is what " +
			"--config-id expects on 'instance create' and 'instance update-config-group'.",
		path:    configPath,
		columns: []string{"id", "name", "datastoreName", "datastoreVersionName", "created"},
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
				return fmt.Errorf("failed to %s: %w", strings.ReplaceAll(spec.use, "-", " "), err)
			}
			return nil
		},
	}
	if spec.byZone {
		cmd.Flags().String("zone-id", "", "Restrict results to one availability zone (e.g. HCM03-1A)")
		cmd.RegisterFlagCompletionFunc("zone-id", zoneIDCompletion()) //nolint:errcheck
	}
	return cmd
}

func init() {
	for _, spec := range simpleLists {
		CatalogCmd.AddCommand(newSimpleList(spec))
	}
	CatalogCmd.AddCommand(listFlavorsCmd)
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// get issues the GET and prints it with a column set.
func get(cmd *cobra.Command, path string, query url.Values, columns []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := vdbclient.Get(apiClient, path, query)
	if err != nil {
		return err
	}

	return vdbclient.OutputWithColumns(cmd, result, columns)
}

func zoneQuery(cmd *cobra.Command) url.Values {
	query := url.Values{}
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		query.Set("zoneId", zoneID)
	}
	return query
}
