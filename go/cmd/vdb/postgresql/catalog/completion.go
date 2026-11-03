package catalog

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completion keys for the cluster-specific values these endpoints own. The
// cluster and backup packages consume them through cli.ResourceCompletion, so
// nothing has to import this package — and registration stays next to the
// endpoint that actually knows the shape.
//
// These are deliberately separate from the relational keys: a cluster flavor
// ('pgp-...') and a relational flavor are not interchangeable, so sharing a key
// would suggest values the API rejects. Zones are the exception — they are
// project-wide, so 'cluster create --zone-id' reuses vdb:relational-zone.
const (
	DatastoreVersionResourceKey = "vdb:postgresql-datastore-version"
	FlavorResourceKey           = "vdb:postgresql-flavor"
	VolumeTypeResourceKey       = "vdb:postgresql-volume-type"
	ConfigGroupResourceKey      = "vdb:postgresql-config-group"
	BackupLocationResourceKey   = "vdb:postgresql-backup-location"
	BackupPolicyResourceKey     = "vdb:postgresql-backup-policy"

	// zoneResourceKey is the RELATIONAL key, reused on purpose: zones are project-wide
	// and pg has no zones endpoint of its own.
	zoneResourceKey = "vdb:relational-zone"
)

// zoneIDCompletion serves the --zone-id that the zone-aware catalog lookups define.
func zoneIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(zoneResourceKey)
}

func init() {
	cli.RegisterResourceCompleter(DatastoreVersionResourceKey, cli.FlagFromAPI(fieldFrom(datastorePath, "version")))
	cli.RegisterResourceCompleter(FlavorResourceKey, cli.FlagFromAPI(fieldFrom(flavorsPath, "id")))
	cli.RegisterResourceCompleter(VolumeTypeResourceKey, cli.FlagFromAPI(fieldFrom(volumeTypesPath, "id")))
	cli.RegisterResourceCompleter(BackupLocationResourceKey, cli.FlagFromAPI(fieldFrom(locationsPath, "id")))
	cli.RegisterResourceCompleter(BackupPolicyResourceKey, cli.FlagFromAPI(fieldFrom(policiesPath, "id")))
	cli.RegisterResourceCompleter(ConfigGroupResourceKey, cli.FlagFromAPI(clusterConfigGroupIDs))
}

// fieldFrom builds a completion fetcher that GETs a catalog path and collects one
// field from every item. Unwrap runs first: the values sit inside the vDB
// {code, message, data} envelope.
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

// clusterConfigGroupIDs suggests only config groups a cluster can actually use.
// The listing is the relational one, which has no deployType filter, so the
// filtering happens here — same rule as list-config-groups.
func clusterConfigGroupIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: 1, PageSize: 100})
	result, err := vdbclient.Get(apiClient, configGroupsPath, query)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(keepClusterGroups(vdbclient.Unwrap(result)), "id"), nil
}
