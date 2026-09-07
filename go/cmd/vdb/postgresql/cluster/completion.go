package cluster

import (
	"context"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// This file holds the completers; each flag is bound to one in the init() of the
// file that DEFINES the flag. init() functions run in filename order within a
// package, so a central binding would silently fail for flags defined in a later
// file — see the note in cmd/vdb/relational/instance/completion.go.

// ClusterResourceKey is the shared completion key for a PostgreSQL Cluster ID.
const ClusterResourceKey = "vdb:postgresql-cluster"

func init() {
	cli.RegisterResourceCompleter(ClusterResourceKey, clusterIDCompletion())
}

// The catalog values a create or resize needs are registered by the catalog
// package, which owns those endpoints; these wrappers dispatch through the shared
// registry at completion time, so no import and no init ordering is involved.
// Zones are the exception: they are project-wide, so the relational key is reused
// rather than duplicated.
func datastoreVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-datastore-version")
}

func packageIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-flavor")
}

func volumeTypeIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-volume-type")
}

func configIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-config-group")
}

func backupLocationCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-backup-location")
}

func backupPolicyCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:postgresql-backup-policy")
}

func zoneIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-zone")
}

// subnetCompletion also reuses the relational listing: subnets are project
// infrastructure, and that endpoint is the one that reports which of them vDB
// accepts.
func subnetCompletion() cli.CompFunc {
	return cli.ResourceCompletion("vdb:relational-subnet")
}

// clusterIDCompletion suggests only "pg-" IDs. The listing behind it is the mixed
// relational one, and every command in this package rejects a "db-" ID, so
// offering those would only invite an error.
//
// Best-effort: that listing answers in ~1.3s but has been seen to spike past 6s,
// which is beyond cli.FlagFromAPI's bound; the shell then gets no suggestions.
func clusterIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := fetchClusterRows(cmd)
		if err != nil {
			return nil, err
		}
		return field(rows, "id"), nil
	})
}

// statusCompletion suggests the statuses actually present on clusters. Neither
// vdb group has a relational status-listing endpoint (the only one in the spec
// belongs to MemoryStore), and the values are free-form strings.
func statusCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := fetchClusterRows(cmd)
		if err != nil {
			return nil, err
		}
		return field(rows, "status"), nil
	})
}

// fetchClusterRows returns one page of cluster rows for completion. pageSize is
// capped at 100 by the API, so a project with more instances than that gets
// suggestions from the first page only.
func fetchClusterRows(cmd *cobra.Command) ([]interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: 1, PageSize: 100})
	result, err := vdbclient.Get(apiClient, relBase, query)
	if err != nil {
		return nil, err
	}

	payload, ok := vdbclient.Unwrap(result).(map[string]interface{})
	if !ok {
		return nil, nil
	}
	items, _ := payload["data"].([]interface{})

	rows := make([]interface{}, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); strings.HasPrefix(id, clusterIDPrefix) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// field collects one string field from rows, de-duplicated in order.
func field(rows []interface{}, name string) []string {
	var out []string
	seen := map[string]bool{}
	for _, item := range rows {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if v, _ := row[name].(string); v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
