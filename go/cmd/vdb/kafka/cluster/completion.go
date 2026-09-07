package cluster

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers live here; each flag is bound to one in the init() of the file that
// DEFINES the flag. init() runs in filename order within a package, so a central
// binding would fail silently for flags defined in a later file — see the note in
// cmd/vdb/CLAUDE.md.

// ClusterResourceKey is the completion key for a Kafka cluster ID. Distinct from
// "vdb:postgresql-cluster" on purpose: the two products share the noun and nothing
// else, and suggesting one's IDs for the other would offer values the API rejects.
const ClusterResourceKey = "vdb:kafka-cluster"

// SecruleResourceKey lists the security-rule IDs of the cluster named on the command
// line. Kafka has no listing endpoint for rules — they arrive nested in the cluster
// object — so the completer reads the cluster, which also makes it context-dependent:
// it yields nothing until --cluster-id is set.
const SecruleResourceKey = "vdb:kafka-secrule"

// Keys owned by other Kafka packages, consumed here.
const (
	kafkaVersionResource      = "vdb:kafka-version"
	flavorResource            = "vdb:kafka-flavor"
	volumeTypeResource        = "vdb:kafka-volume-type"
	configGroupVersionResourc = "vdb:kafka-config-group-version"
)

func init() {
	cli.RegisterResourceCompleter(ClusterResourceKey, clusterIDCompletion())
	cli.RegisterResourceCompleter(SecruleResourceKey, cli.FlagFromAPI(secruleIDs))
}

func clusterIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := listClusters(cmd)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(rows, "id"), nil
	})
}

// statusCompletion suggests the statuses actually present in the project. Kafka has
// no status enumeration endpoint at all, and the one MemoryStore offers is outdated,
// so deriving from the listing is the only source that cannot go stale.
func statusCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		rows, err := listClusters(cmd)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(rows, "status"), nil
	})
}

// listClusters fetches the cluster listing for completion. There is no pagination on
// this endpoint, so this is the whole list.
func listClusters(cmd *cobra.Command) (interface{}, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	result, err := apiClient.Get(BasePath, nil)
	if err != nil {
		return nil, err
	}
	return vdbclient.Unwrap(result), nil
}

// secruleIDs reads the rules nested in the cluster named by --cluster-id.
func secruleIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	if clusterID == "" {
		return nil, nil
	}
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}
	clusterObj, err := fetchCluster(apiClient, clusterID)
	if err != nil {
		return nil, err
	}
	return cli.ExtractIDs(clusterObj["securityGroupRules"], "id"), nil
}

func secruleIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(SecruleResourceKey)
}

func kafkaVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion(kafkaVersionResource)
}

func flavorIDCompletion() cli.CompFunc {
	return cli.ResourceCompletion(flavorResource)
}

func volumeTypeCompletion() cli.CompFunc {
	return cli.ResourceCompletion(volumeTypeResource)
}

func configGroupVersionCompletion() cli.CompFunc {
	return cli.ResourceCompletion(configGroupVersionResourc)
}
