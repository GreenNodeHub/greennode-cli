package cluster

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// A PostgreSQL Cluster is served by two API prefixes. Its own operations
// (create, resize, settings, config-group, volume-used) live under pgBase; the
// ones the product never got — list, get-by-id, histories, secrules, reboot,
// delete — are served by the RELATIONAL endpoints under relBase, which accept
// "pg-" IDs on purpose. Neither set can be derived from the other, so both live
// here as plain constants and each command states which one it uses.
const (
	pgBase  = "/vdb-postgresql/v1/cluster"
	relBase = "/vdb-relational/v1/database-instances"
)

// clusterIDPrefix identifies a PostgreSQL Cluster. The relational endpoints this
// package calls also serve Relational Database instances ("db-"), so every
// command validates the prefix before sending: without that check
// `postgresql cluster delete --cluster-id db-...` would happily delete a
// Relational Database instance from the PostgreSQL command group.
const clusterIDPrefix = "pg-"

func pgPath(clusterID, suffix string) string {
	return pgBase + "/" + clusterID + suffix
}

func relPath(clusterID, suffix string) string {
	return relBase + "/" + clusterID + suffix
}

// detailPath is the relational get-by-id path, which puts the ID behind an extra
// "id" segment.
func detailPath(clusterID string) string {
	return relBase + "/id/" + clusterID
}

// requireClusterID validates the ID and rejects anything that is not a cluster.
func requireClusterID(clusterID string) error {
	return vdbclient.RequireIDWithPrefix(clusterID, "cluster-id", clusterIDPrefix,
		"PostgreSQL Cluster IDs start with 'pg-'; use 'grn vdb relational instance' for Relational Database instances")
}

// listColumns is the table view of a cluster. The listing is the relational one,
// so rows carry all ~68 relational fields; the cluster-specific ones are
// numberOfNodes and privateRwIp/publicRwIp, which replace a single instance's ip
// list.
var listColumns = []string{
	"id", "name", "status", "datastoreVersion", "numberOfNodes",
	"vcpus", "ram", "volumeSize", "privateRwIp", "created",
}

// historyColumns is the table view of a HistoryResponse entry. It leads with description
// because `action` alone does not say what happened: every settings change, resize and
// config-group attach or detach is recorded as "Update", and only the description
// distinguishes them ("… Change redis password", "… Change flavor from db.s-general-2x4
// … to db.s-general-4x8", "… Attach config X"). Create and create-replica entries embed
// the whole order cart and run past 200 characters, which is the cost of having the
// column; it is worth paying, since those two rows are the ones a reader can already
// identify from `action`.
//
// updatedTime is left to JSON output: it equalled createdTime on all 57 records seen
// across three instances in both products, so in a table it is 31 characters spent
// repeating the previous column. errorMessage stays — it is the only place a Failed row
// explains itself.
var historyColumns = []string{
	"id", "action", "description", "status", "createdTime", "errorMessage",
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}
