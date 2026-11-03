package catalog

import (
	"context"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// This file holds the completers. Flags are bound to them in the init() of the
// file that DEFINES the flag (or right after the flag is added, for the
// generated listings) — never from a central function here, because init()
// functions run in filename order and a binding for a flag defined in a later
// file fails silently. See the note in instance/completion.go.

// Completion keys for values that come from these catalog endpoints. Later
// phases (instance create, resize, replica create) take the same values as flags
// and should complete them with cli.ResourceCompletion(<key>) rather than
// importing this package.
const (
	DatastoreTypeResourceKey    = "vdb:relational-datastore-type"
	DatastoreVersionResourceKey = "vdb:relational-datastore-version"
	ZoneResourceKey             = "vdb:relational-zone"
	VolumeTypeResourceKey       = "vdb:relational-volume-type"

	// SubnetResourceKey lists the subnets a vDB instance or cluster can be placed
	// in. It is vdb's own listing, not vserver's: this endpoint returns the subnets
	// vDB will actually accept, and unlike "vserver:subnet" it needs no --vpc-id on
	// the command.
	SubnetResourceKey = "vdb:relational-subnet"

	// NetworkResourceKey lists the networks (VPCs) vDB will accept. Same reasoning
	// as SubnetResourceKey: "vserver:network" is the platform's own listing, this
	// one is what vDB agrees to place a resource in. Consumed by Kafka's
	// 'cluster create --network-id' — Kafka has no networks endpoint of its own,
	// and networks are project-wide.
	NetworkResourceKey = "vdb:relational-network"

	// ConfigGroupResourceKey lists config groups usable by an instance.
	ConfigGroupResourceKey = "vdb:relational-config-group"

	// FlavorResourceKey lists flavor ids for --package-id. Unlike the other keys
	// this one is CONTEXT-DEPENDENT: the endpoint requires an engine and version,
	// which the completer reads from the command's own --datastore-type and
	// --datastore-version flags, so it yields nothing until those are set.
	FlavorResourceKey = "vdb:relational-flavor"
)

func init() {
	cli.RegisterResourceCompleter(DatastoreTypeResourceKey, datastoreTypeCompletion())
	cli.RegisterResourceCompleter(DatastoreVersionResourceKey, datastoreVersionCompletion())
	cli.RegisterResourceCompleter(ZoneResourceKey, zoneIDCompletion())
	cli.RegisterResourceCompleter(VolumeTypeResourceKey, volumeTypeCompletion())
	cli.RegisterResourceCompleter(SubnetResourceKey, subnetCompletion())
	cli.RegisterResourceCompleter(NetworkResourceKey, cli.FlagFromAPI(fieldFrom(networksPath, "id")))
	cli.RegisterResourceCompleter(ConfigGroupResourceKey, cli.FlagFromAPI(instanceConfigGroupIDs))
	cli.RegisterResourceCompleter(FlavorResourceKey, cli.FlagFromAPI(flavorIDs))
}

// subnetCompletion suggests subnet UUIDs. The listing returns NETWORKS, each with
// its subnets nested, so the UUIDs have to be gathered a level down — the generic
// cli.ExtractIDs would return the network UUIDs instead.
func subnetCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		apiClient, err := vdbclient.BuildClient(cmd)
		if err != nil {
			return nil, err
		}
		result, err := apiClient.Get(subnetsPath, nil)
		if err != nil {
			return nil, err
		}
		networks, ok := vdbclient.Unwrap(result).([]interface{})
		if !ok {
			return nil, nil
		}

		var out []string
		for _, item := range networks {
			network, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			subnets, _ := network["subnets"].([]interface{})
			out = append(out, cli.ExtractIDs(subnets, "uuid")...)
		}
		return out, nil
	})
}

func datastoreTypeCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fieldFrom(datastorePath, "type"))
}

// datastoreVersionCompletion suggests every version the API offers. It does not
// narrow by --datastore-type: the datastore listing pairs type with version, but
// filtering would silently offer nothing when --datastore-type is misspelled,
// which reads as "no versions exist" rather than "wrong type".
func datastoreVersionCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fieldFrom(datastorePath, "version"))
}

func zoneIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fieldFrom(zonesPath, "uuid"))
}

func volumeTypeCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fieldFrom(volumeTypesPath, "type"))
}

// flavorIDs suggests the flavor ids valid for the engine the user has already
// named on the same command line, the way vserver's subnet completer reads
// --vpc-id. Without both --datastore-type and --datastore-version the endpoint has
// nothing to filter on, so the completer stays silent rather than guessing.
//
// The ids come back as JSON NUMBERS while the create request wants them as strings,
// so they are formatted here — cli.ExtractIDs, which only collects strings, would
// return nothing.
func flavorIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	datastoreType, _ := cmd.Flags().GetString("datastore-type")
	datastoreVersion, _ := cmd.Flags().GetString("datastore-version")
	if datastoreType == "" || datastoreVersion == "" {
		return nil, nil
	}

	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("type", datastoreType)
	query.Set("version", datastoreVersion)
	if zoneID, _ := cmd.Flags().GetString("zone-id"); zoneID != "" {
		query.Set("zoneId", zoneID)
	}

	result, err := vdbclient.Get(apiClient, flavorsPath, query)
	if err != nil {
		return nil, err
	}

	return vdbclient.ExtractIDValues(vdbclient.Unwrap(result), "id"), nil
}

// instanceConfigGroupIDs suggests only config groups a Relational Database
// instance can actually use — the endpoint also returns PostgreSQL Cluster groups,
// which it would reject.
func instanceConfigGroupIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	result, err := apiClient.Get(configGroupsPath, nil)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(dropClusterGroups(vdbclient.Unwrap(result)), "id"), nil
}

// fieldFrom builds a completion fetcher that GETs a catalog path and collects one
// field from every item. Unwrap runs first: the values live inside the vDB
// {code, message, data} envelope, and cli.ExtractIDs would otherwise find no
// slice to walk.
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
