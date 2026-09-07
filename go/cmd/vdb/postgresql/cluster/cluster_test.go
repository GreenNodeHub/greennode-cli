package cluster

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestRequireClusterIDRejectsInstanceIDs is the safety net for this package: the
// relational endpoints it calls for get, histories, secrules, reboot and delete
// serve Relational Database instances too, so a "db-" ID must never get through.
func TestRequireClusterIDRejectsInstanceIDs(t *testing.T) {
	if err := requireClusterID("pg-2e6f2253-9032-466f-975f-d6d6b6ec8330"); err != nil {
		t.Errorf("valid cluster ID rejected: %v", err)
	}

	for _, id := range []string{
		"db-66a37ca3-e688-4a8a-9e35-a89fa69733c6", // a relational instance
		"2e6f2253-9032-466f-975f-d6d6b6ec8330",    // bare UUID
		"",
		"pg id with spaces",
	} {
		if err := requireClusterID(id); err == nil {
			t.Errorf("requireClusterID(%q) = nil, want an error", id)
		}
	}
}

func TestPathsUseTheRightPrefix(t *testing.T) {
	const id = "pg-1234"

	cases := []struct {
		name string
		got  string
		want string
	}{
		// Relational: the operations the cluster product does not have.
		{"get", detailPath(id), "/vdb-relational/v1/database-instances/id/pg-1234"},
		{"histories", relPath(id, "/histories"), "/vdb-relational/v1/database-instances/pg-1234/histories"},
		{"secrules", relPath(id, "/secrules"), "/vdb-relational/v1/database-instances/pg-1234/secrules"},
		{"reboot", relPath(id, "/reboot"), "/vdb-relational/v1/database-instances/pg-1234/reboot"},
		{"delete", relPath(id, "/delete"), "/vdb-relational/v1/database-instances/pg-1234/delete"},
		// PostgreSQL: its own operations.
		{"resize", pgPath(id, "/resize"), "/vdb-postgresql/v1/cluster/pg-1234/resize"},
		{"settings", pgPath(id, "/settings"), "/vdb-postgresql/v1/cluster/pg-1234/settings"},
		{"config-group", pgPath(id, "/config-group"), "/vdb-postgresql/v1/cluster/pg-1234/config-group"},
		{"volume-used", pgPath(id, "/volume-used"), "/vdb-postgresql/v1/cluster/pg-1234/volume-used"},
	}

	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s path = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestKeepClusters: the listing is shared with Relational Database instances, and
// its server-side filters do not apply to cluster rows, so both the product
// filtering and --name/--status happen here.
func TestKeepClusters(t *testing.T) {
	payload := map[string]interface{}{
		"projectId": "pro-1",
		"data": []interface{}{
			map[string]interface{}{"id": "pg-a", "name": "prod-cluster", "status": "ACTIVE"},
			map[string]interface{}{"id": "db-b", "name": "prod-instance", "status": "ACTIVE"},
			map[string]interface{}{"id": "pg-c", "name": "staging", "status": "BUILDING"},
		},
		"pageObject": map[string]interface{}{"totalElements": float64(3)},
	}

	ids := func(result interface{}) []string {
		obj := result.(map[string]interface{})
		var out []string
		for _, item := range obj["data"].([]interface{}) {
			out = append(out, item.(map[string]interface{})["id"].(string))
		}
		return out
	}

	cases := []struct {
		name     string
		filter   string
		statuses []string
		want     string
	}{
		{"no filter drops db- rows", "", nil, "pg-a,pg-c"},
		{"name is a substring match", "prod", nil, "pg-a"},
		{"name is case-insensitive", "PROD", nil, "pg-a"},
		{"status filters", "", []string{"BUILDING"}, "pg-c"},
		{"several statuses", "", []string{"ACTIVE", "BUILDING"}, "pg-a,pg-c"},
		{"name and status combine", "prod", []string{"BUILDING"}, ""},
	}

	for _, c := range cases {
		got := strings.Join(ids(keepClusters(payload, c.filter, c.statuses)), ",")
		if got != c.want {
			t.Errorf("%s: kept %q, want %q", c.name, got, c.want)
		}
	}

	// The rest of the payload must survive: pageObject is how a user pages.
	result := keepClusters(payload, "", nil).(map[string]interface{})
	if result["pageObject"] == nil || result["projectId"] != "pro-1" {
		t.Error("keepClusters dropped keys other than the item array")
	}
	if len(payload["data"].([]interface{})) != 3 {
		t.Error("keepClusters mutated the original payload")
	}
}

func TestCreateBodyMapsFlagsToAPIFields(t *testing.T) {
	cmd := freshCreateCmd(t, map[string]string{
		"name":              "my-cluster",
		"datastore-version": "17",
		"package-id":        "pgp-1",
		"volume-type-id":    "pgst-1",
		"volume-size":       "40",
		"number-of-nodes":   "3",
		"zone-id":           "HCM03-1A",
		"subnet-ids":        "sub-1, sub-2",
		"username":          "pgadmin",
		"password":          "secret",
		"database-name":     "appdb",
	})

	body, err := createBody(cmd)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}

	// The flag names are CLI-shaped; the API's are not.
	for field, want := range map[string]interface{}{
		"name":             "my-cluster",
		"datastoreVersion": "17",
		"packageId":        "pgp-1",
		"volumeTypeId":     "pgst-1",
		"volumeSize":       40,
		"numberOfNodes":    3,
		"locateZoneId":     "HCM03-1A",
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}

	// netIds is a list even for one subnet; user and databases are nested.
	netIDs, ok := body["netIds"].([]interface{})
	if !ok || len(netIDs) != 2 || netIDs[0] != "sub-1" || netIDs[1] != "sub-2" {
		t.Errorf("netIds = %v, want [sub-1 sub-2]", body["netIds"])
	}
	user, ok := body["user"].(map[string]interface{})
	if !ok || user["name"] != "pgadmin" || user["password"] != "secret" {
		t.Errorf("user = %v, want the master credentials", body["user"])
	}
	databases, ok := body["databases"].([]interface{})
	if !ok || len(databases) != 1 {
		t.Fatalf("databases = %v, want one entry", body["databases"])
	}
	if databases[0].(map[string]interface{})["name"] != "appdb" {
		t.Errorf("databases[0] = %v, want name appdb", databases[0])
	}

	// Optional IDs must be absent rather than empty: an empty configId means
	// "detach" on the update endpoint, so empty strings are not inert here.
	for _, field := range []string{"configId", "backupLocationId", "backupPolicyId", "backupPointId"} {
		if _, present := body[field]; present {
			t.Errorf("body carries %q even though the flag was not set", field)
		}
	}
}

func TestCreateBodyRejectsBadInput(t *testing.T) {
	base := map[string]string{
		"name": "c", "datastore-version": "17", "package-id": "pgp-1",
		"volume-type-id": "pgst-1", "volume-size": "40", "zone-id": "HCM03-1A",
		"subnet-ids": "sub-1", "username": "u", "password": "p", "database-name": "d",
	}

	cases := []struct {
		name     string
		override map[string]string
	}{
		{"one node is below the cluster minimum", map[string]string{"number-of-nodes": "1"}},
		{"eleven nodes is above the maximum", map[string]string{"number-of-nodes": "11"}},
		{"zero volume size", map[string]string{"volume-size": "0"}},
		{"no password anywhere", map[string]string{"password": ""}},
		{"no subnet", map[string]string{"subnet-ids": ""}},
	}

	for _, c := range cases {
		flags := map[string]string{}
		for k, v := range base {
			flags[k] = v
		}
		for k, v := range c.override {
			flags[k] = v
		}

		// The password test must not pick up a real environment value.
		t.Setenv(passwordEnv, "")

		if _, err := createBody(freshCreateCmd(t, flags)); err == nil {
			t.Errorf("%s: createBody = nil error, want a rejection", c.name)
		}
	}
}

func TestCreateBodyReadsPasswordFromEnv(t *testing.T) {
	t.Setenv(passwordEnv, "from-env")

	cmd := freshCreateCmd(t, map[string]string{
		"name": "c", "datastore-version": "17", "package-id": "pgp-1",
		"volume-type-id": "pgst-1", "volume-size": "40", "zone-id": "HCM03-1A",
		"subnet-ids": "sub-1", "username": "u", "database-name": "d",
	})

	body, err := createBody(cmd)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}
	if got := body["user"].(map[string]interface{})["password"]; got != "from-env" {
		t.Errorf("password = %v, want the value from $%s", got, passwordEnv)
	}
}

func TestResizeBodySendsOnlyTheChosenDimension(t *testing.T) {
	cases := []struct {
		resizeType string
		flags      map[string]string
		wantField  string
		wantValue  interface{}
		absent     []string
	}{
		{"VOLUME-SIZE", map[string]string{"volume-size": "80"}, "volumeSize", 80,
			[]string{"volumeTypeId", "numberOfNodes"}},
		{"VOLUME-TYPE", map[string]string{"volume-type-id": "pgst-2"}, "volumeTypeId", "pgst-2",
			[]string{"volumeSize", "numberOfNodes"}},
		{"NUMBER-OF-NODES", map[string]string{"number-of-nodes": "5"}, "numberOfNodes", 5,
			[]string{"volumeSize", "volumeTypeId"}},
	}

	for _, c := range cases {
		flags := map[string]string{"cluster-id": "pg-1", "type": c.resizeType}
		for k, v := range c.flags {
			flags[k] = v
		}

		body, summary, err := resizeBody(freshResizeCmd(t, flags))
		if err != nil {
			t.Errorf("%s: resizeBody: %v", c.resizeType, err)
			continue
		}
		if body["type"] != c.resizeType {
			t.Errorf("%s: type = %v", c.resizeType, body["type"])
		}
		if body[c.wantField] != c.wantValue {
			t.Errorf("%s: body[%q] = %v, want %v", c.resizeType, c.wantField, body[c.wantField], c.wantValue)
		}
		for _, field := range c.absent {
			if _, present := body[field]; present {
				t.Errorf("%s: body carries unrelated field %q", c.resizeType, field)
			}
		}
		if summary == "" {
			t.Errorf("%s: no summary for the confirmation prompt", c.resizeType)
		}
	}
}

func TestResizeBodyRequiresTheMatchingFlag(t *testing.T) {
	// Each mode reads exactly one flag; without it the API would accept an order
	// that changes nothing.
	for resizeType, missing := range resizeFlagFor {
		_, _, err := resizeBody(freshResizeCmd(t, map[string]string{"cluster-id": "pg-1", "type": resizeType}))
		if err == nil {
			t.Errorf("--type %s without --%s = nil error, want a rejection", resizeType, missing)
		}
	}

	if _, _, err := resizeBody(freshResizeCmd(t, map[string]string{"cluster-id": "pg-1", "type": "GROW"})); err == nil {
		t.Error("unknown --type accepted")
	}
}

func TestSettingsBodyOnlySendsWhatWasSet(t *testing.T) {
	t.Setenv(passwordEnv, "")

	// publicAccess false is a real, connectivity-breaking setting, so it must be
	// sent only when the flag was actually given.
	body, _, err := settingsBody(freshSettingsCmd(t, map[string]string{"password": "new-secret"}))
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["password"] != "new-secret" {
		t.Errorf("password not sent: %v", body)
	}
	if _, present := body["publicAccess"]; present {
		t.Error("publicAccess sent although --public-access was not given")
	}

	body, _, err = settingsBody(freshSettingsCmd(t, map[string]string{"public-access": "false"}))
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["publicAccess"] != false {
		t.Errorf("publicAccess = %v, want false", body["publicAccess"])
	}
	if _, present := body["password"]; present {
		t.Error("password sent although none was given")
	}

	if _, _, err := settingsBody(freshSettingsCmd(t, nil)); err == nil {
		t.Error("empty update accepted; it would be a pointless API call")
	}
}

// TestDeleteHasNoDeleteAllBackupFlag: a cluster honours only createFinalBackup —
// the shared schema's deleteAllBackup is a Relational Database instance option.
// (The action-body shape itself is covered in internal/vdbclient.)
func TestDeleteHasNoDeleteAllBackupFlag(t *testing.T) {
	if deleteCmd.Flags().Lookup("delete-all-backup") != nil {
		t.Error("cluster delete defines --delete-all-backup, which a cluster does not support")
	}
}

func TestDestructiveAndPaidCommandsAreGated(t *testing.T) {
	// The repo-wide conventions test only requires this of delete/stop/reboot; the
	// vdb notes extend it to create and resize, which place paid orders, and to the
	// updates that can break connectivity.
	for _, cmd := range []*cobra.Command{
		createCmd, resizeCmd, deleteCmd, rebootCmd,
		updateSettingsCmd, updateConfigGroupCmd, updateSecruleCmd,
	} {
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s must define --%s", cmd.Name(), flag)
			}
		}
	}
}

func TestClusterFlagCompletionsAreRegistered(t *testing.T) {
	cases := []struct {
		cmd   *cobra.Command
		flags []string
	}{
		{getCmd, []string{"cluster-id"}},
		{listCmd, []string{"status"}},
		{listHistoriesCmd, []string{"cluster-id"}},
		{getVolumeUsedCmd, []string{"cluster-id"}},
		{listSecrulesCmd, []string{"cluster-id"}},
		{createCmd, []string{"datastore-version", "package-id", "volume-type-id", "zone-id",
			"subnet-ids", "config-id", "backup-location-id", "backup-policy-id"}},
		{resizeCmd, []string{"cluster-id", "type", "volume-type-id"}},
		{rebootCmd, []string{"cluster-id"}},
		{deleteCmd, []string{"cluster-id"}},
		{updateSettingsCmd, []string{"cluster-id"}},
		{updateConfigGroupCmd, []string{"cluster-id", "config-id"}},
		{updateSecruleCmd, []string{"cluster-id"}},
	}

	for _, c := range cases {
		for _, flag := range c.flags {
			if c.cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s has no --%s flag", c.cmd.Name(), flag)
				continue
			}
			if _, ok := c.cmd.GetFlagCompletionFunc(flag); !ok {
				t.Errorf("%s --%s has no completion function registered", c.cmd.Name(), flag)
			}
		}
	}
}

// freshCreateCmd and friends build a throwaway command carrying the same flags as
// the real one. Tests use these rather than the package-level commands, which are
// mounted in the assembled command tree: setting flags on those leaks values
// between tests and into the running CLI.
func freshCreateCmd(t *testing.T, values map[string]string) *cobra.Command {
	t.Helper()
	return freshCmd(t, createFlags, values)
}

func freshResizeCmd(t *testing.T, values map[string]string) *cobra.Command {
	t.Helper()
	return freshCmd(t, resizeFlagsOn, values)
}

func freshSettingsCmd(t *testing.T, values map[string]string) *cobra.Command {
	t.Helper()
	return freshCmd(t, settingsFlagsOn, values)
}

func freshCmd(t *testing.T, define func(*pflag.FlagSet), values map[string]string) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	define(cmd.Flags())

	for name, value := range values {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("setting --%s=%q: %v", name, value, err)
		}
	}
	return cmd
}

// TestHistoryColumnsCarryDescription: `action` is "Update" for every settings change,
// resize and config-group attach or detach, so a history table without description cannot
// tell them apart — which is why it leads the column set.
func TestHistoryColumnsCarryDescription(t *testing.T) {
	var hasDescription, hasUpdated bool
	for _, column := range historyColumns {
		switch column {
		case "description":
			hasDescription = true
		case "updatedTime":
			hasUpdated = true
		}
	}
	if !hasDescription {
		t.Errorf("history table has no description column: %v", historyColumns)
	}
	// updatedTime duplicated createdTime on every record observed live. If that ever
	// changes this is the place to reconsider, rather than a silent re-add.
	if hasUpdated {
		t.Errorf("updatedTime is JSON-only, it repeats createdTime: %v", historyColumns)
	}
	if historyColumns[1] != "action" || historyColumns[2] != "description" {
		t.Errorf("description must follow action so a row reads as one phrase: %v", historyColumns)
	}
}
