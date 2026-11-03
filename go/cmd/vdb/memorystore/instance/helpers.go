package instance

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

const (
	basePath = "/vdb-memory/v1/database-instances"

	// paymentPath is where instances are CREATED — an order, not a POST to the
	// collection.
	paymentPath = "/vdb-memory/v1/payment/database-instances"

	// instanceIDPrefix is shared with Relational Database: both products issue
	// "db-…" IDs, so the prefix cannot tell them apart. It still rejects a cluster
	// ("pg-"), a backup ("bk-") or a config group ("cfg-") reaching a path.
	instanceIDPrefix = "db-"

	// redisPort is what a MemoryStore instance listens on, used as the default for a
	// security rule that names no port.
	redisPort = 6379
)

// detailPath is get-by-id. **No `/id/` segment** — that is the relational spelling
// and it 404s here.
func detailPath(instanceID string) string {
	return basePath + "/" + instanceID
}

// instancePath builds any per-instance sub-path.
func instancePath(instanceID, suffix string) string {
	return basePath + "/" + instanceID + suffix
}

func validateInstanceID(instanceID string) error {
	return validator.ValidateID(instanceID, "instance-id")
}

// requireInstanceID rejects IDs that belong to another resource. Note it cannot
// distinguish a MemoryStore instance from a Relational Database one — both are
// "db-…" — so a mixed-up ID is caught by the API, not here.
func requireInstanceID(instanceID string) error {
	return vdbclient.RequireIDWithPrefix(instanceID, "instance-id", instanceIDPrefix,
		"MemoryStore instance IDs start with 'db-'; run 'memorystore instance list' to find one")
}

// listColumns is the table view of a MemoryStore instance. There is no volume type
// column: Redis capacity comes from the flavor, so ram is the size that matters.
var listColumns = []string{
	"id", "name", "status", "datastoreType", "datastoreVersion",
	"vcpus", "ram", "ip", "created",
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

// replicaColumns matches the reduced replica payload — 19 fields, no ip or port.
var replicaColumns = []string{
	"id", "name", "status", "datastoreType", "datastoreVersion",
	"vcpus", "ram", "created",
}

func createClient(cmd *cobra.Command) (*vdbclient.Client, error) {
	return vdbclient.BuildClient(cmd)
}

// fetchInstance reads one instance for the commands that need its current values:
// create-replica defaults from it, and update-secrule takes its port from it.
func fetchInstance(apiClient *vdbclient.Client, instanceID string) (map[string]interface{}, error) {
	result, err := apiClient.Get(detailPath(instanceID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read MemoryStore instance %s: %w", instanceID, err)
	}
	instance, ok := vdbclient.PayloadObject(result)
	if !ok {
		return nil, fmt.Errorf("MemoryStore instance %s not found (the API returned an empty payload)", instanceID)
	}
	return instance, nil
}

func stringField(payload map[string]interface{}, key string) string {
	value, _ := payload[key].(string)
	return value
}

func intField(payload map[string]interface{}, key string) int {
	value, _ := payload[key].(float64)
	return int(value)
}

func boolField(payload map[string]interface{}, key string) bool {
	value, _ := payload[key].(bool)
	return value
}
