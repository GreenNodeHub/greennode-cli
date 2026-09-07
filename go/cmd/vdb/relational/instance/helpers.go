package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// basePath is the Relational Database instance collection. Note the sibling
// MemoryStore API uses the same resource name under a different prefix but with
// different sub-paths, so nothing here is reusable across groups by design.
const basePath = "/vdb-relational/v1/database-instances"

// paymentPath is where instances are CREATED. Creation is an order, not a POST to
// the collection — a detail with no equivalent in vks or vserver.
const paymentPath = "/vdb-relational/v1/payment/database-instances"

// instanceIDPrefix marks a Relational Database instance. The listing and several
// action endpoints also serve PostgreSQL Clusters ("pg-"), which belong to
// `grn vdb postgresql`; commands that MUTATE must reject those IDs so a cluster is
// never rebooted or deleted through the wrong command group.
const instanceIDPrefix = "db-"

// detailPath is the get-by-id path. Relational puts the ID behind an extra "id"
// segment (`/database-instances/id/{dbInstanceId}`) where MemoryStore uses
// `/database-instances/{dbInstanceId}` — one of the asymmetries that rules out a
// shared path builder.
func detailPath(instanceID string) string {
	return basePath + "/id/" + instanceID
}

// historiesPath is the instance action log. Here the path parameter is named
// instanceId, while detailPath's is dbInstanceId — same resource, two names.
func historiesPath(instanceID string) string {
	return basePath + "/" + instanceID + "/histories"
}

// instancePath builds any other per-instance path (actions, secrules, replicas).
func instancePath(instanceID, suffix string) string {
	return basePath + "/" + instanceID + suffix
}

// validateInstanceID accepts any vdb instance ID. Read-only commands use it,
// because get and list-histories serve PostgreSQL Clusters too (verified live).
func validateInstanceID(instanceID string) error {
	return validator.ValidateID(instanceID, "instance-id")
}

// requireRelationalID additionally rejects PostgreSQL Cluster IDs. Every command
// that changes state uses this: the relational reboot and delete endpoints happily
// act on a "pg-" cluster, and doing that from `grn vdb relational` would disturb a
// resource the user manages elsewhere.
func requireRelationalID(instanceID string) error {
	return vdbclient.RequireIDWithPrefix(instanceID, "instance-id", instanceIDPrefix,
		"Relational Database instance IDs start with 'db-'; use 'grn vdb postgresql cluster' for 'pg-' clusters")
}

// listColumns is the table view of a database instance. The API returns ~68
// fields per instance (including billing metadata such as cost, period and
// enableAutoRenew); JSON output keeps all of them.
var listColumns = []string{
	"id", "name", "status", "datastoreType", "datastoreVersion",
	"vcpus", "ram", "volumeSize", "ip", "created",
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

// replicaColumns is the table view of a replica. A replica listing returns a
// REDUCED instance — 19 fields, with no ip, port or securityGroup — so listColumns
// cannot be reused here (shape confirmed by the product team).
var replicaColumns = []string{
	"id", "name", "status", "datastoreType", "datastoreVersion",
	"vcpus", "ram", "volumeSize", "created",
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

func outputList(cmd *cobra.Command, data interface{}) error {
	return vdbclient.OutputWithColumns(cmd, data, listColumns)
}

// fetchInstance reads one instance, unwrapped, for commands that need its current
// values: resize-storage must send the fields it is NOT changing, create-replica
// defaults everything to the source instance, and update-secrule takes its default
// port from it.
func fetchInstance(apiClient *vdbclient.Client, instanceID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(detailPath(instanceID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read database instance %s: %w", instanceID, err)
	}
	instance, ok := vdbclient.PayloadObject(result)
	if !ok {
		// vDB reports "not found" as HTTP 200 with a null payload on some endpoints, so
		// this is a miss rather than a malformed response.
		return nil, fmt.Errorf("database instance %s not found (the API returned an empty payload)", instanceID)
	}
	return instance, nil
}

// stringField reads a string field from an instance payload, tolerating null.
func stringField(instance map[string]interface{}, key string) string {
	value, _ := instance[key].(string)
	return value
}

// intField reads a numeric field. Every JSON number decodes to float64.
func intField(instance map[string]interface{}, key string) int {
	value, _ := instance[key].(float64)
	return int(value)
}

func boolField(instance map[string]interface{}, key string) bool {
	value, _ := instance[key].(bool)
	return value
}
