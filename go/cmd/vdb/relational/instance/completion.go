package instance

import (
	"context"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// This file holds the completers; each flag is bound to one in the init() of the
// file that DEFINES the flag, not from a central function here.
//
// Why: within a package, Go runs init() functions file by file in filename order.
// A central registerCompletions() called from instance.go would run before
// list.go and list_histories.go have defined their flags, and
// RegisterFlagCompletionFunc fails on an unknown flag — silently, since the error
// is discarded at init time. That produced completion that looked wired but was
// not; completion_test.go now guards against it.
//
// vDB declares no enums anywhere in its spec (status, datastore type and flavor
// are free-form strings), so every completer here is API-backed. cli.FlagFromAPI
// bounds each call with a timeout and yields nothing on error, so a slow or
// unreachable backend never breaks the shell.

// InstanceResourceKey is the shared completion key for a Relational Database
// instance ID. Other vDB groups (backups, configurations) take an instance ID as
// a flag and can complete it with cli.ResourceCompletion(InstanceResourceKey)
// instead of importing this package.
//
// It suggests every ID the listing returns, including the PostgreSQL Cluster
// records ("pg-"), because the two commands consuming it — instance get and
// list-histories — were verified to serve those clusters as well. A future
// command that only works on true Relational instances must filter by the "db-"
// prefix itself, after checking how the endpoint actually behaves.
const InstanceResourceKey = "vdb:relational-instance"

func init() {
	cli.RegisterResourceCompleter(InstanceResourceKey, instanceIDCompletion())
}

// instanceIDCompletion is best-effort. The instance listing endpoint normally
// answers in ~1.3s (measured live in HCM-3, where it aggregates Relational
// instances and PostgreSQL Clusters), but spikes past 6s were observed; when it
// exceeds cli.FlagFromAPI's bound the shell simply gets no suggestions. The
// catalog endpoints are an order of magnitude faster (~0.2s).
func instanceIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fetchRelationalInstanceIDs)
}

func statusCompletion() cli.CompFunc {
	return cli.FlagFromAPI(fetchStatuses)
}

// relationalInstanceIDsFunc suggests only "db-" IDs. Every mutating command in this
// package rejects a "pg-" cluster, so offering those would only invite an error;
// the read-only commands share the completer for consistency, since a cluster is
// better inspected through `grn vdb postgresql cluster`.
func relationalInstanceIDsFunc() cli.CompFunc {
	return cli.FlagFromAPI(func(ctx context.Context, cmd *cobra.Command) ([]string, error) {
		ids, err := fetchRelationalInstanceIDs(ctx, cmd)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			if strings.HasPrefix(id, instanceIDPrefix) {
				out = append(out, id)
			}
		}
		return out, nil
	})
}

// The catalog values a create, resize or config change needs are registered by the
// catalog package, which owns those endpoints; these wrappers dispatch through the
// shared registry at completion time, so no import and no init ordering is
// involved.
func datastoreTypeCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-datastore-type")
}

func datastoreVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-datastore-version")
}

func volumeTypeCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-volume-type")
}

func zoneIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-zone")
}

func subnetCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-subnet")
}

func configIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-config-group")
}

// packageIDCompletion needs the engine to be known: the flavors endpoint takes a
// type and a version, which the completer reads from the same command line. On
// 'create' those are flags, so completion works once they are set; 'resize-instance'
// has no such flags, so it deliberately does not offer this — see the note there.
func packageIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-flavor")
}

// listAll fetches one large page of instances for completion purposes. pageSize
// is capped at 100 by the API (pageObject.maxSize), so asking for more is
// pointless; a project with more instances than that simply gets the first page
// suggested.
func listAll(cmd *cobra.Command) (interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: 1, PageSize: 100})
	return vdbclient.Get(apiClient, basePath, query)
}

func fetchRelationalInstanceIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	result, err := listAll(cmd)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(vdbclient.Unwrap(result), "id"), nil
}

// fetchStatuses suggests the statuses actually present in the project. The
// relational API has no status-listing endpoint, and the one MemoryStore has
// (/vdb-memory/v1/database/status) is outdated — the product owner ruled it out on
// 2026-08-13 — so both groups read their own listing.
func fetchStatuses(_ context.Context, cmd *cobra.Command) ([]string, error) {
	result, err := listAll(cmd)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(vdbclient.Unwrap(result), "status"), nil
}
