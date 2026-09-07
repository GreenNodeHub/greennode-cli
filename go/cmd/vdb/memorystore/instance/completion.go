package instance

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers live here; each flag is bound to one in the init() of the file that
// DEFINES the flag — init() runs in filename order, so a central binding would
// silently fail for flags defined later. See cmd/vdb/relational/instance/completion.go.

// InstanceResourceKey is the completion key for a MemoryStore instance ID. It is
// deliberately distinct from "vdb:relational-instance": both products issue "db-…"
// IDs but keep separate listings, so suggesting one product's IDs for the other's
// command would offer values the API rejects.
const InstanceResourceKey = "vdb:memorystore-instance"

// Keys owned by the memorystore catalog package.
const (
	datastoreTypeResource    = "vdb:memorystore-datastore-type"
	datastoreVersionResource = "vdb:memorystore-datastore-version"
	configGroupResource      = "vdb:memorystore-config-group"
	flavorResource           = "vdb:memorystore-flavor"
)

// zoneResource and subnetResource are the RELATIONAL keys, reused on purpose:
// MemoryStore has no zones or subnets endpoint of its own — zones and subnets are
// project infrastructure, and the product owner confirmed the relational catalog is
// where they come from.
const (
	zoneResource   = "vdb:relational-zone"
	subnetResource = "vdb:relational-subnet"
)

func init() {
	cli.RegisterResourceCompleter(InstanceResourceKey, instanceIDCompletion())
}

func instanceIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := listInstances(cmd)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(rows, "id"), nil
	})
}

// listInstances fetches one page of instances for completion. pageSize is capped at
// 100 by the API, so a project with more gets suggestions from the first page.
func listInstances(cmd *cobra.Command) (interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: 1, PageSize: 100})
	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return nil, err
	}
	return vdbclient.Unwrap(result), nil
}

// statusCompletion suggests the statuses actually present in the project, read from
// the instance listing.
//
// MemoryStore does have a `/database/status` endpoint — the only status enumeration
// in the whole vDB API — but the product owner reported it OUTDATED on 2026-08-13, so
// it is not used at all. Deriving from the listing costs nothing extra here and cannot
// go stale.
func statusCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := listInstances(cmd)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(rows, "status"), nil
	})
}

func datastoreTypeCompletion() cli.CompFunc {
	return cli.ResourceCompletion(datastoreTypeResource)
}

func datastoreVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion(datastoreVersionResource)
}

func configIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(configGroupResource)
}

// packageIDCompletion needs the engine version to be known: the flavors endpoint takes a
// version, which the completer reads from the same command line. 'create' has that flag,
// so completion works once it is set. 'resize-instance' and 'create-replica' do not — the
// version comes from the existing instance — so they deliberately do not offer it; see
// the note on each.
func packageIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(flavorResource)
}

func zoneIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(zoneResource)
}

func subnetCompletion() cli.CompFunc {
	return cli.ResourceCompletion(subnetResource)
}
