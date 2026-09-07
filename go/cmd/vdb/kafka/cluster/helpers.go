package cluster

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BasePath is the Kafka cluster collection. Note the prefix has NO /v1 segment —
// relational, memorystore and postgresql all carry one, Kafka does not.
const BasePath = "/vdb-kafka/clusters"

// ClusterPath builds a per-cluster sub-path. Exported because the topic and user
// packages hang off the same cluster path.
func ClusterPath(clusterID, suffix string) string {
	return BasePath + "/" + clusterID + suffix
}

// ValidateClusterID checks the ID is URL-safe.
//
// There is deliberately NO prefix guard here, unlike relational ("db-") and
// postgresql ("pg-"): no Kafka path is shared with another product, so there is
// nothing for a prefix to protect against, and the ID format was not confirmed
// against the live API. Do not invent one from a sample ID.
func ValidateClusterID(clusterID string) error {
	return validator.ValidateID(clusterID, "cluster-id")
}

// listColumns is the table view of a KafkaCluster. The full object carries ~29
// fields including billing metadata and the nested securityGroupRules, so table
// output needs a subset; --output json keeps everything.
var listColumns = []string{
	"id", "name", "status", "kafkaVersion", "kafkaBrokerCount",
	"kafkaStorageType", "kafkaStorageSize", "vcpus", "ram", "createdAt",
}

// historyColumns matches HistoryDto, which is Kafka's own shape and NOT the
// HistoryResponse the other products return: it has startedAt/finishedAt instead of
// createdTime/updatedTime, and a clusterId. As elsewhere, description leads because
// `action` alone rarely says what changed.
var historyColumns = []string{
	"id", "action", "description", "status", "startedAt", "finishedAt", "errorMessage",
}

// secruleColumns matches KafkaSecurityGroupRule, the items nested in a cluster's
// securityGroupRules.
var secruleColumns = []string{"id", "remoteIp", "port", "status", "createdAt"}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// fetchCluster reads one cluster for the commands that need its current values —
// delete prints what it is about to remove, list-secrules digs into it, and
// resize-brokers refuses a no-op.
//
// PayloadObject rather than a plain type assertion: vDB answers "not found" with
// HTTP 200 and a null payload on some endpoints, and reading fields off the
// envelope would silently yield zeroes.
func fetchCluster(apiClient *vdbclient.Client, clusterID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(BasePath+"/"+clusterID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read Kafka cluster %s: %w", clusterID, err)
	}
	clusterObj, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("Kafka cluster %s not found (the API returned an empty payload)", clusterID)
	}
	return clusterObj, nil
}

// previewQuery is --dry-run for the Kafka operations whose arguments travel in the
// QUERY STRING rather than a body: vdbclient.PreviewBody prints a JSON body, and
// these have none, so it would show "null" and hide the very values being changed.
// Keys are sorted so the output is stable.
func previewQuery(verb, target, method, path string, params map[string]string) {
	fmt.Println("=== DRY RUN ===")
	if target != "" {
		fmt.Printf("Would %s %s with:\n", verb, target)
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	query := url.Values{}
	for _, k := range keys {
		fmt.Printf("  %s: %s\n", k, params[k])
		query.Set(k, params[k])
	}
	fmt.Printf("\n  %s %s?%s\n", method, path, query.Encode())
	cli.DryRunNotice(verb)
}

func stringField(payload map[string]interface{}, key string) string {
	value, _ := payload[key].(string)
	return value
}

func intField(payload map[string]interface{}, key string) int {
	value, _ := payload[key].(float64)
	return int(value)
}
