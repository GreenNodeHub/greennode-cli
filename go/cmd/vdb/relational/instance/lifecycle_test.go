package instance

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// sourceInstance is a captured live payload of a real instance, used to test the
// commands that default from or validate against current state.
func sourceInstance() map[string]interface{} {
	return map[string]interface{}{
		"id":               "db-66a37ca3-e688-4a8a-9e35-a89fa69733c6",
		"name":             "docs-agent",
		"datastoreType":    "PostgreSQL",
		"datastoreVersion": "15",
		"quotaPackageId":   "103",
		"volumeType":       "Gen2-NVMe2-IOPS3000",
		"volumeSize":       float64(60),
		"zoneId":           "HCM03-1A",
		"subnetId":         "sub-246df886-3ef8-4d70-87e7-6b87f0caea50",
		"publicAccess":     true,
		"backupAuto":       true,
		"backupDuration":   float64(2),
		"backupTime":       "00:00",
		"port":             float64(5432),
		// The flat configId is null even when a group is attached; the nested object
		// is where the API reports it.
		"configId":      nil,
		"configuration": map[string]interface{}{"id": "cfg-87d6499f", "name": "config-group"},
	}
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

// TestMutatingCommandsRejectClusterIDs is the safety net for this package. The
// relational reboot, delete and resize endpoints also serve PostgreSQL Clusters, so
// a "pg-" ID reaching them from here would disturb a resource managed by
// `grn vdb postgresql cluster`.
func TestMutatingCommandsRejectClusterIDs(t *testing.T) {
	if err := requireRelationalID("db-66a37ca3-e688"); err != nil {
		t.Errorf("valid instance ID rejected: %v", err)
	}
	for _, id := range []string{"pg-2e6f2253-9032", "66a37ca3", "", "db id with spaces"} {
		if err := requireRelationalID(id); err == nil {
			t.Errorf("requireRelationalID(%q) = nil, want an error", id)
		}
	}

	// Read-only commands accept both, because get and list-histories serve clusters.
	if err := validateInstanceID("pg-2e6f2253-9032"); err != nil {
		t.Errorf("read-only validation rejected a cluster ID: %v", err)
	}
}

func TestInstancePaths(t *testing.T) {
	const id = "db-1234"

	cases := map[string]string{
		detailPath(id):                      "/vdb-relational/v1/database-instances/id/db-1234",
		historiesPath(id):                   "/vdb-relational/v1/database-instances/db-1234/histories",
		instancePath(id, "/secrules"):       "/vdb-relational/v1/database-instances/db-1234/secrules",
		instancePath(id, "/replicas"):       "/vdb-relational/v1/database-instances/db-1234/replicas",
		instancePath(id, "/shutdown"):       "/vdb-relational/v1/database-instances/db-1234/shutdown",
		instancePath(id, "/resize-storage"): "/vdb-relational/v1/database-instances/db-1234/resize-storage",
		instancePath(id, "/update/setting"): "/vdb-relational/v1/database-instances/db-1234/update/setting",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}

	// Creation is an order, not a POST to the collection.
	if paymentPath != "/vdb-relational/v1/payment/database-instances" {
		t.Errorf("paymentPath = %q", paymentPath)
	}
}

// TestActionSpecs: the CLI verb and the API path diverge for stop, and each
// endpoint accepts only its own action value.
func TestActionSpecs(t *testing.T) {
	byUse := map[string]action{}
	for _, spec := range actions {
		byUse[spec.use] = spec
	}

	stop, ok := byUse["stop"]
	if !ok {
		t.Fatal("no stop command declared")
	}
	if stop.path != "/shutdown" {
		t.Errorf("stop path = %q, want /shutdown (the API's name for it)", stop.path)
	}
	if stop.name != "stop" {
		t.Errorf("stop action = %q, want stop", stop.name)
	}
	if stop.confirm == "" {
		t.Error("stop must confirm: applications lose their connections")
	}
	if byUse["reboot"].confirm == "" {
		t.Error("reboot must confirm")
	}
	if byUse["start"].confirm != "" {
		t.Error("start should not prompt; it breaks nothing")
	}

	// conventions_test.go requires --dry-run and --force on stop and reboot; give
	// them to start too so scripts can treat the three alike.
	for _, spec := range actions {
		cmd := newActionCmd(spec)
		for _, flag := range []string{"instance-id", "dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s must define --%s", spec.use, flag)
			}
		}
		if _, registered := cmd.GetFlagCompletionFunc("instance-id"); !registered {
			t.Errorf("%s --instance-id has no completion function registered", spec.use)
		}
	}
}

// validDBPassword satisfies the product rules the CLI now enforces: 8-32 characters,
// letters/digits/$^_<> only.
const validDBPassword = "GrnCliTest2026$x"

func TestCreateBodyMapsFlagsToAPIFields(t *testing.T) {
	cmd := freshCmd(t, createFlags, map[string]string{
		"name": "my-mysql", "datastore-type": "MySQL", "datastore-version": "8.0",
		"package-id": "211", "volume-type": "Gen2-NVMe2-IOPS3000", "volume-size": "40",
		"zone-id": "HCM03-1A", "subnet-ids": "sub-1, sub-2", "username": "dbadmin",
		"password": validDBPassword, "database-name": "appdb",
		"character-set": "utf8mb4", "collate": "utf8mb4_general_ci",
	})

	body, err := createBody(cmd)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}

	for field, want := range map[string]interface{}{
		"name":             "my-mysql",
		"datastoreType":    "MySQL",
		"datastoreVersion": "8.0",
		// packageId is the flavor id as a STRING, and volumeType is the type NAME —
		// the PostgreSQL Cluster group uses opaque pgp-/pgst- IDs instead.
		"packageId":    "211",
		"volumeType":   "Gen2-NVMe2-IOPS3000",
		"volumeSize":   40,
		"locateZoneId": "HCM03-1A",
		"backupAuto":   false,
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}

	netIDs, ok := body["netIds"].([]interface{})
	if !ok || len(netIDs) != 2 {
		t.Errorf("netIds = %v, want two entries", body["netIds"])
	}
	user, ok := body["user"].(map[string]interface{})
	if !ok || user["name"] != "dbadmin" || user["password"] != validDBPassword {
		t.Errorf("user = %v", body["user"])
	}
	database := body["databases"].([]interface{})[0].(map[string]interface{})
	if database["name"] != "appdb" || database["characterSet"] != "utf8mb4" || database["collate"] != "utf8mb4_general_ci" {
		t.Errorf("databases[0] = %v", database)
	}

	// configId absent rather than empty: an empty configId means "detach" on the
	// update endpoint, so empty strings are not inert in this API.
	if _, present := body["configId"]; present {
		t.Error("body carries configId although the flag was not set")
	}
	// Retention fields must not appear when automatic backup is off.
	for _, field := range []string{"backupDuration", "backupTime"} {
		if _, present := body[field]; present {
			t.Errorf("body carries %q with backupAuto false", field)
		}
	}
}

func TestCreateBodyBackupRules(t *testing.T) {
	base := map[string]string{
		"name": "n", "datastore-type": "MySQL", "datastore-version": "8.0",
		"package-id": "211", "volume-type": "T", "volume-size": "40",
		"zone-id": "Z", "subnet-ids": "sub-1", "username": "u",
		"password": validDBPassword, "database-name": "d",
	}
	with := func(extra map[string]string) map[string]string {
		flags := map[string]string{}
		for k, v := range base {
			flags[k] = v
		}
		for k, v := range extra {
			flags[k] = v
		}
		return flags
	}

	// The API requires retention and time whenever automatic backup is on.
	body, err := createBody(freshCmd(t, createFlags, with(map[string]string{
		"backup-auto": "true", "backup-duration": "7", "backup-time": "02:00",
	})))
	if err != nil {
		t.Fatalf("createBody with backup: %v", err)
	}
	if body["backupDuration"] != 7 || body["backupTime"] != "02:00" {
		t.Errorf("backup fields = %v / %v", body["backupDuration"], body["backupTime"])
	}

	for name, flags := range map[string]map[string]string{
		"backup on, no retention":       {"backup-auto": "true"},
		"backup on, no time":            {"backup-auto": "true", "backup-duration": "7"},
		"retention below the minimum":   {"backup-auto": "true", "backup-duration": "1", "backup-time": "02:00"},
		"retention above the maximum":   {"backup-auto": "true", "backup-duration": "15", "backup-time": "02:00"},
		"retention without backup-auto": {"backup-duration": "7"},
		"time without backup-auto":      {"backup-time": "02:00"},
		"zero volume size":              {"volume-size": "0"},
	} {
		t.Setenv(passwordEnv, "")
		if _, err := createBody(freshCmd(t, createFlags, with(flags))); err == nil {
			t.Errorf("%s: createBody = nil error, want a rejection", name)
		}
	}
}

func TestCreateBodyReadsPasswordFromEnv(t *testing.T) {
	t.Setenv(passwordEnv, validDBPassword)

	body, err := createBody(freshCmd(t, createFlags, map[string]string{
		"name": "n", "datastore-type": "MySQL", "datastore-version": "8.0",
		"package-id": "211", "volume-type": "T", "volume-size": "40",
		"zone-id": "Z", "subnet-ids": "sub-1", "username": "u", "database-name": "d",
	}))
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}
	if got := body["user"].(map[string]interface{})["password"]; got != validDBPassword {
		t.Errorf("password = %v, want the value from $%s", got, passwordEnv)
	}
}

// TestReplicaBodyDefaultsToTheSource is the point of create-replica: a replica has
// a dozen required fields that all describe the source, so the command reads them
// instead of making the user retype them.
func TestReplicaBodyDefaultsToTheSource(t *testing.T) {
	source := sourceInstance()
	cmd := freshCmd(t, createReplicaFlags, map[string]string{
		"instance-id": "db-66a37ca3-e688-4a8a-9e35-a89fa69733c6",
		"name":        "docs-agent-replica",
	})

	body, err := replicaBody(cmd, "db-66a37ca3-e688-4a8a-9e35-a89fa69733c6", source)
	if err != nil {
		t.Fatalf("replicaBody: %v", err)
	}

	for field, want := range map[string]interface{}{
		"name":             "docs-agent-replica",
		"replicaSourceId":  "db-66a37ca3-e688-4a8a-9e35-a89fa69733c6",
		"datastoreType":    "PostgreSQL",
		"datastoreVersion": "15",
		// quotaPackageId in the instance payload, packageId in the request.
		"packageId":    "103",
		"volumeType":   "Gen2-NVMe2-IOPS3000",
		"volumeSize":   60,
		"locateZoneId": "HCM03-1A",
		"publicAccess": true,
		"backupAuto":   true,
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}

	// The source's subnet becomes a one-element list.
	netIDs := body["netIds"].([]interface{})
	if len(netIDs) != 1 || netIDs[0] != "sub-246df886-3ef8-4d70-87e7-6b87f0caea50" {
		t.Errorf("netIds = %v", body["netIds"])
	}
	// The attached config group is only in the nested object, not the flat field.
	if body["configId"] != "cfg-87d6499f" {
		t.Errorf("configId = %v, want the group from configuration.id", body["configId"])
	}
	// Backup is on for the source, so its schedule must come along.
	if body["backupDuration"] != 2 || body["backupTime"] != "00:00" {
		t.Errorf("backup schedule = %v / %v", body["backupDuration"], body["backupTime"])
	}
}

func TestReplicaBodyOverrides(t *testing.T) {
	cmd := freshCmd(t, createReplicaFlags, map[string]string{
		"instance-id": "db-1", "name": "r",
		"package-id": "271", "volume-size": "120", "zone-id": "HCM03-1B",
		"subnet-ids": "sub-x", "public-access": "false",
	})

	body, err := replicaBody(cmd, "db-1", sourceInstance())
	if err != nil {
		t.Fatalf("replicaBody: %v", err)
	}

	for field, want := range map[string]interface{}{
		"packageId":    "271",
		"volumeSize":   120,
		"locateZoneId": "HCM03-1B",
		// Changed() rather than emptiness, so an explicit false is an override and
		// not an unset flag.
		"publicAccess": false,
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}
	if netIDs := body["netIds"].([]interface{}); netIDs[0] != "sub-x" {
		t.Errorf("netIds = %v, want the override", body["netIds"])
	}
	// Engine and version are never overridable: a replica must match its source.
	if body["datastoreType"] != "PostgreSQL" || body["datastoreVersion"] != "15" {
		t.Errorf("engine = %v %v, want the source's", body["datastoreType"], body["datastoreVersion"])
	}
}

func TestReplicaBodyNeedsWhatTheSourceCannotProvide(t *testing.T) {
	bare := map[string]interface{}{"datastoreType": "MySQL", "datastoreVersion": "8.0"}

	cmd := freshCmd(t, createReplicaFlags, map[string]string{"instance-id": "db-1", "name": "r"})
	if _, err := replicaBody(cmd, "db-1", bare); err == nil {
		t.Error("replicaBody accepted a source with no subnet, size or flavor")
	}
}

// TestStorageBodyFillsTheUnchangedField: the API wants both size and type on every
// request, using the current value for whichever is not changing, so omitting one
// is not the same as leaving it alone.
func TestStorageBodyFillsTheUnchangedField(t *testing.T) {
	instance := sourceInstance()

	cmd := freshCmd(t, resizeStorageFlags, map[string]string{"instance-id": "db-1", "volume-size": "80"})
	body, summary, err := storageBody(cmd.Flags(), instance, "db-1")
	if err != nil {
		t.Fatalf("storageBody: %v", err)
	}
	config := body["databaseInstances"].([]interface{})[0].(map[string]interface{})["config"].(map[string]interface{})
	if config["volumeSize"] != 80 {
		t.Errorf("volumeSize = %v, want 80", config["volumeSize"])
	}
	if config["volumeType"] != "Gen2-NVMe2-IOPS3000" {
		t.Errorf("volumeType = %v, want the instance's current type", config["volumeType"])
	}
	if !strings.Contains(summary, "60 -> 80") {
		t.Errorf("summary = %q, want it to name the change", summary)
	}

	// Type-only change keeps the current size.
	cmd = freshCmd(t, resizeStorageFlags, map[string]string{"instance-id": "db-1", "volume-type": "Gen2-NVMe2-IOPS5000"})
	body, _, err = storageBody(cmd.Flags(), instance, "db-1")
	if err != nil {
		t.Fatalf("storageBody type-only: %v", err)
	}
	config = body["databaseInstances"].([]interface{})[0].(map[string]interface{})["config"].(map[string]interface{})
	if config["volumeSize"] != 60 || config["volumeType"] != "Gen2-NVMe2-IOPS5000" {
		t.Errorf("config = %v", config)
	}
}

func TestStorageBodyRefusesPointlessOrHarmfulResizes(t *testing.T) {
	instance := sourceInstance()

	for name, flags := range map[string]map[string]string{
		"shrinking":        {"instance-id": "db-1", "volume-size": "20"},
		"no change at all": {"instance-id": "db-1", "volume-size": "60"},
		"same type as now": {"instance-id": "db-1", "volume-type": "Gen2-NVMe2-IOPS3000"},
	} {
		cmd := freshCmd(t, resizeStorageFlags, flags)
		if _, _, err := storageBody(cmd.Flags(), instance, "db-1"); err == nil {
			t.Errorf("%s: storageBody = nil error, want a rejection", name)
		}
	}
}

// backupsOff / backupsOn stand in for the instance record settingsBody reads the
// current backup schedule from.
var (
	backupsOff = map[string]interface{}{"backupAuto": false}
	backupsOn  = map[string]interface{}{"backupAuto": true, "backupTime": "01:00", "backupDuration": float64(7)}
)

func TestSettingsBodyOnlySendsWhatWasSet(t *testing.T) {
	t.Setenv(passwordEnv, "")

	// publicAccess is a boolean whose zero value is a real, destructive setting, so it
	// must be sent only when given.
	body, _, err := settingsBody(freshCmd(t, settingsFlags, map[string]string{"password": validDBPassword}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["password"] != validDBPassword || body["dbInstanceId"] != "db-1" {
		t.Errorf("body = %v", body)
	}
	if _, present := body["publicAccess"]; present {
		t.Error("publicAccess sent although its flag was not given")
	}

	// An explicit false must reach the API.
	body, _, err = settingsBody(freshCmd(t, settingsFlags, map[string]string{"public-access": "false"}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody public-access: %v", err)
	}
	if body["publicAccess"] != false {
		t.Errorf("publicAccess = %v, want false", body["publicAccess"])
	}

	if _, _, err := settingsBody(freshCmd(t, settingsFlags, nil).Flags(), "db-1", backupsOff); err == nil {
		t.Error("empty update accepted; it would be a pointless API call")
	}
}

// TestSettingsBodyAlwaysCarriesTheBackupSchedule guards a live finding: the API answers
// HTTP 500 to any update/setting body without backupAuto, so a password-only or
// public-access-only change must still repeat the instance's own schedule.
func TestSettingsBodyAlwaysCarriesTheBackupSchedule(t *testing.T) {
	t.Setenv(passwordEnv, "")

	body, summary, err := settingsBody(
		freshCmd(t, settingsFlags, map[string]string{"password": validDBPassword}).Flags(), "db-1", backupsOn)
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["backupAuto"] != true || body["backupTime"] != "01:00" || body["backupDuration"] != 7 {
		t.Errorf("the instance's schedule must be repeated verbatim, body = %v", body)
	}
	// Repeating it is not a change, so it must not be reported as one.
	if strings.Contains(summary, "backup") {
		t.Errorf("summary claims a backup change: %q", summary)
	}

	// backupAuto false is carried too — the field's presence is what the API needs.
	body, _, err = settingsBody(
		freshCmd(t, settingsFlags, map[string]string{"password": validDBPassword}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["backupAuto"] != false {
		t.Errorf("backupAuto missing or wrong, body = %v", body)
	}
	if _, present := body["backupTime"]; present {
		t.Error("backupTime sent although backups are off")
	}

	// The time alone can be changed while backups are on; retention comes from the
	// instance.
	body, summary, err = settingsBody(
		freshCmd(t, settingsFlags, map[string]string{"backup-time": "05:30"}).Flags(), "db-1", backupsOn)
	if err != nil {
		t.Fatalf("settingsBody time only: %v", err)
	}
	if body["backupTime"] != "05:30" || body["backupDuration"] != 7 || body["backupAuto"] != true {
		t.Errorf("body = %v", body)
	}
	if !strings.Contains(summary, "05:30") {
		t.Errorf("summary must name the new time: %q", summary)
	}
}

func TestSettingsBodyBackupRules(t *testing.T) {
	t.Setenv(passwordEnv, "")

	body, _, err := settingsBody(freshCmd(t, settingsFlags, map[string]string{
		"backup-auto": "true", "backup-duration": "7", "backup-time": "03:00",
	}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["backupDuration"] != 7 || body["backupTime"] != "03:00" {
		t.Errorf("body = %v", body)
	}

	// Turning backup off needs neither, and must not carry them.
	body, _, err = settingsBody(freshCmd(t, settingsFlags, map[string]string{"backup-auto": "false"}).Flags(), "db-1", backupsOn)
	if err != nil {
		t.Fatalf("settingsBody backup off: %v", err)
	}
	if body["backupAuto"] != false {
		t.Errorf("backupAuto = %v", body["backupAuto"])
	}
	if _, present := body["backupDuration"]; present {
		t.Error("retention sent while disabling backup")
	}

	for name, flags := range map[string]map[string]string{
		"schedule while backups are off": {"backup-duration": "7"},
		"backup on without retention":    {"backup-auto": "true", "backup-time": "03:00"},
		"backup on without time":         {"backup-auto": "true", "backup-duration": "7"},
	} {
		if _, _, err := settingsBody(freshCmd(t, settingsFlags, flags).Flags(), "db-1", backupsOff); err == nil {
			t.Errorf("%s: accepted, want a rejection", name)
		}
	}
}

// TestDefaultPortOf: MySQL and MariaDB listen on 3306 and PostgreSQL on 5432, so a
// single hard-coded default for --rule would be wrong for some instances.
func TestDefaultPortOf(t *testing.T) {
	if got := defaultPortOf(sourceInstance()); got != 5432 {
		t.Errorf("port from the instance = %d, want 5432", got)
	}

	// No port reported: fall back to the engine.
	for engine, want := range map[string]int{"MySQL": 3306, "MariaDB": 3306, "PostgreSQL": 5432} {
		got := defaultPortOf(map[string]interface{}{"datastoreType": engine})
		if got != want {
			t.Errorf("%s fallback = %d, want %d", engine, got, want)
		}
	}
	// Neither available: still a usable number rather than 0, which the API rejects.
	if got := defaultPortOf(map[string]interface{}{}); got <= 0 {
		t.Errorf("last-resort port = %d, want a valid port", got)
	}
}

func TestMutatingCommandsAreGated(t *testing.T) {
	// conventions_test.go requires --dry-run and --force of delete/stop/reboot; the
	// vdb notes extend it to the order-flow commands and to anything that can break
	// connectivity.
	for _, cmd := range []*cobra.Command{
		createCmd, createReplicaCmd, detachReplicaCmd,
		resizeInstanceCmd, resizeStorageCmd, deleteCmd,
		updateSettingsCmd, updateConfigGroupCmd, updateSecruleCmd,
	} {
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s must define --%s", cmd.Name(), flag)
			}
		}
	}
}

func TestFlagCompletionsAreRegistered(t *testing.T) {
	cases := []struct {
		cmd   *cobra.Command
		flags []string
	}{
		{listCmd, []string{"status"}},
		{getCmd, []string{"instance-id"}},
		{listHistoriesCmd, []string{"instance-id"}},
		{listSecrulesCmd, []string{"instance-id"}},
		{listReplicasCmd, []string{"instance-id"}},
		{createCmd, []string{"datastore-type", "datastore-version", "package-id", "volume-type", "zone-id", "subnet-ids", "config-id"}},
		{createReplicaCmd, []string{"instance-id", "volume-type", "zone-id", "subnet-ids", "config-id"}},
		{detachReplicaCmd, []string{"instance-id"}},
		{resizeInstanceCmd, []string{"instance-id"}},
		{resizeStorageCmd, []string{"instance-id", "volume-type"}},
		{deleteCmd, []string{"instance-id"}},
		{updateSettingsCmd, []string{"instance-id"}},
		{updateConfigGroupCmd, []string{"instance-id", "config-id"}},
		{updateSecruleCmd, []string{"instance-id"}},
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

	// resize-instance deliberately does NOT complete --package-id: the flavors
	// endpoint needs an engine and version, which this command has no flags for, and
	// deriving them would mean fetching the instance inside the completion timeout —
	// against the slowest endpoint in the API. Asserted so the omission reads as a
	// decision rather than an oversight.
	if _, ok := resizeInstanceCmd.GetFlagCompletionFunc("package-id"); ok {
		t.Error("resize-instance --package-id now completes; if that is intentional, " +
			"check it does not fetch the instance listing inside the completion bound")
	}
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
