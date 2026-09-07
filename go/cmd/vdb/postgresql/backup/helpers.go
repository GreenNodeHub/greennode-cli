package backup

import (
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
)

const (
	clusterIDPrefix = "pg-"

	// clusterResourceKey is the completion key registered by the cluster package.
	// Consumers use the literal, as vks does for the vserver keys, so no package
	// dependency is needed just to share a constant.
	clusterResourceKey = "vdb:postgresql-cluster"
)

func requireClusterID(clusterID string) error {
	return vdbclient.RequireIDWithPrefix(clusterID, "cluster-id", clusterIDPrefix,
		"PostgreSQL Cluster IDs start with 'pg-'; backups of Relational Database instances live under 'grn vdb relational'")
}
