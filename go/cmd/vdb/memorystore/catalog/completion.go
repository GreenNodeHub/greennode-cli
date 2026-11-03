package catalog

import (
	"context"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completion keys owned by these endpoints. They are distinct from the relational
// ones because the values are not interchangeable — a MemoryStore flavor id is not a
// relational flavor id, and a Redis config group cannot be attached to MySQL.
const (
	DatastoreTypeResourceKey    = "vdb:memorystore-datastore-type"
	DatastoreVersionResourceKey = "vdb:memorystore-datastore-version"
	ConfigGroupResourceKey      = "vdb:memorystore-config-group"

	// defaultDatastoreType is what the consuming commands default --datastore-type to.
	defaultDatastoreType = "Redis"

	// FlavorResourceKey lists flavor ids for --package-id. It is CONTEXT-DEPENDENT:
	// the endpoint requires an engine version, which the completer reads from the
	// command's own --datastore-version (and --zone-id when given), so it yields
	// nothing until that is set.
	FlavorResourceKey = "vdb:memorystore-flavor"
)

// zoneResourceKey is the RELATIONAL key, reused: MemoryStore has no zones endpoint.
const zoneResourceKey = "vdb:relational-zone"

func init() {
	cli.RegisterResourceCompleter(DatastoreTypeResourceKey, cli.FlagFromAPI(fieldFrom(datastorePath, "type")))
	cli.RegisterResourceCompleter(DatastoreVersionResourceKey, cli.FlagFromAPI(fieldFrom(datastorePath, "version")))
	cli.RegisterResourceCompleter(ConfigGroupResourceKey, cli.FlagFromAPI(fieldFrom(configPath, "id")))
	cli.RegisterResourceCompleter(FlavorResourceKey, cli.FlagFromAPI(flavorIDs))
}

func datastoreTypeCompletion() cli.CompFunc {
	return cli.ResourceCompletion(DatastoreTypeResourceKey)
}

func datastoreVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion(DatastoreVersionResourceKey)
}

func zoneIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(zoneResourceKey)
}

// flavorIDs suggests package ids for the engine version named on the command line. The
// ids are JSON numbers, so they are formatted rather than read as strings, and duplicates
// are dropped: one flavor name can appear twice in a zone (once with familyType and
// platformType populated, once with both null) under different ids.
func flavorIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	datastoreVersion, _ := cmd.Flags().GetString("datastore-version")
	if datastoreVersion == "" {
		return nil, nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("version", datastoreVersion)
	// The type flag is optional on the consuming commands; Redis is the only engine
	// MemoryStore serves, so its own default is what the endpoint expects.
	datastoreType, _ := cmd.Flags().GetString("datastore-type")
	if datastoreType == "" {
		datastoreType = defaultDatastoreType
	}
	query.Set("type", datastoreType)
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		query.Set("zoneId", zoneID)
	}

	result, err := vdbclient.Get(apiClient, flavorsPath, query)
	if err != nil {
		return nil, err
	}
	return vdbclient.ExtractIDValues(vdbclient.Unwrap(result), "id"), nil
}

// fieldFrom collects one field from every item of a catalog listing. Unwrap runs
// first: the values sit inside the {code, message, data} envelope.
func fieldFrom(path, field string) func(context.Context, *cobra.Command) ([]string, error) {
	return func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		apiClient, err := vdbclient.BuildClient(cmd)
		if err != nil {
			return nil, err
		}
		result, err := apiClient.Get(path, nil)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(vdbclient.Unwrap(result), field), nil
	}
}
