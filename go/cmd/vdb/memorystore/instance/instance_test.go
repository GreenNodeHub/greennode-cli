package instance

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// validRedisPassword satisfies the product rules the CLI now enforces: 16-128
// characters, letters/digits/$^_<> only.
const validRedisPassword = "RedisCliTest2026abcd"

// liveInstance is a captured MemoryStore instance, used to test the defaulting that
// create-replica does and the port that update-secrule picks.
func liveInstance() map[string]interface{} {
	return map[string]interface{}{
		"id":                   "db-dc665cbe-63f9-4f43-8fd9-5586c4ff038b",
		"name":                 "nhontt-test",
		"status":               "ACTIVE",
		"datastoreType":        "Redis",
		"datastoreVersion":     "7.2",
		"quotaPackageId":       "284",
		"zoneId":               "HCM03-1B",
		"subnetId":             "sub-7cc39ad2-a00f-4edd-8e8b-5f5aaa3eeffe",
		"port":                 float64(6379),
		"publicAccess":         false,
		"redisPasswordEnabled": false,
		"backupAuto":           true,
		"backupDuration":       float64(2),
		"backupTime":           "00:00",
		"configId":             nil,
		"configuration":        nil,
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

// TestPathsDifferFromRelational pins the spellings that a copy from the relational
// package would get wrong.
func TestPathsDifferFromRelational(t *testing.T) {
	const id = "db-1234"

	cases := map[string]string{
		// No /id/ segment — that is the relational form.
		detailPath(id): "/vdb-memory/v1/database-instances/db-1234",
		// Hyphenated, where relational nests these under /update/.
		instancePath(id, "/update-setting"):      "/vdb-memory/v1/database-instances/db-1234/update-setting",
		instancePath(id, "/update-config-group"): "/vdb-memory/v1/database-instances/db-1234/update-config-group",
		instancePath(id, "/shutdown"):            "/vdb-memory/v1/database-instances/db-1234/shutdown",
		instancePath(id, "/secrules"):            "/vdb-memory/v1/database-instances/db-1234/secrules",
		instancePath(id, "/replicas"):            "/vdb-memory/v1/database-instances/db-1234/replicas",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}

	if paymentPath != "/vdb-memory/v1/payment/database-instances" {
		t.Errorf("paymentPath = %q", paymentPath)
	}
}

// TestNoResizeStorageCommand: MemoryStore has no volume, so a storage resize would be
// a command with no endpoint behind it.
func TestNoResizeStorageCommand(t *testing.T) {
	for _, sub := range InstanceCmd.Commands() {
		if sub.Name() == "resize-storage" {
			t.Error("memorystore has no volume and no resize-storage endpoint; this command should not exist")
		}
	}
	if _, _, err := InstanceCmd.Find([]string{"resize-instance"}); err != nil {
		t.Errorf("resize-instance should exist: %v", err)
	}
}

func TestCreateBodyHasNoVolumeAndSetsThePassword(t *testing.T) {
	t.Setenv(passwordEnv, "")

	cmd := freshCmd(t, createFlags, map[string]string{
		"name": "my-redis", "datastore-version": "7.2", "package-id": "284",
		"zone-id": "HCM03-1B", "subnet-ids": "sub-1", "redis-password": validRedisPassword,
	})

	body, err := createBody(cmd)
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}

	for field, want := range map[string]interface{}{
		"name":             "my-redis",
		"datastoreType":    "Redis",
		"datastoreVersion": "7.2",
		"packageId":        "284",
		"locateZoneId":     "HCM03-1B",
		// A password given without the enable flag means "require it" — otherwise the
		// body would set a password and disable it in the same breath.
		"redisPasswordEnabled": true,
		"redisPassword":        validRedisPassword,
		"backupAuto":           false,
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}

	// Redis has no volume and no database user.
	for _, field := range []string{"volumeType", "volumeSize", "user", "databases"} {
		if _, present := body[field]; present {
			t.Errorf("body carries %q, which MemoryStore has no concept of", field)
		}
	}
}

// TestCreateRequiresPasswordForPublicAccess: the API documents that public access
// needs the master password enabled, and rejecting it here names the flags.
func TestCreateRequiresPasswordForPublicAccess(t *testing.T) {
	t.Setenv(passwordEnv, "")

	base := map[string]string{
		"name": "r", "datastore-version": "7.2", "package-id": "284",
		"zone-id": "Z", "subnet-ids": "sub-1",
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

	if _, err := createBody(freshCmd(t, createFlags, with(map[string]string{"public-access": "true"}))); err == nil {
		t.Error("--public-access without a password was accepted")
	}
	if _, err := createBody(freshCmd(t, createFlags, with(map[string]string{"redis-password-enabled": "true"}))); err == nil {
		t.Error("--redis-password-enabled without a password was accepted")
	}

	// With a password, public access is fine and the requirement is implied.
	body, err := createBody(freshCmd(t, createFlags, with(map[string]string{
		"public-access": "true", "redis-password": validRedisPassword,
	})))
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}
	if body["redisPasswordEnabled"] != true || body["publicAccess"] != true {
		t.Errorf("body = %v", body)
	}

	// No password at all is valid as long as nothing requires one.
	body, err = createBody(freshCmd(t, createFlags, base))
	if err != nil {
		t.Fatalf("createBody without a password: %v", err)
	}
	if body["redisPasswordEnabled"] != false {
		t.Errorf("redisPasswordEnabled = %v, want false", body["redisPasswordEnabled"])
	}
	if _, present := body["redisPassword"]; present {
		t.Error("an empty password must not be sent")
	}
}

// TestSettingsBodySetsEditRedisPassword: the API needs that marker alongside any
// change to the password or to whether it is required. It is not a setting, so the CLI
// sets it rather than exposing a flag for it.
func TestSettingsBodySetsEditRedisPassword(t *testing.T) {
	t.Setenv(passwordEnv, "")

	body, _, err := settingsBody(freshCmd(t, settingsFlags, map[string]string{"redis-password": validRedisPassword}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody: %v", err)
	}
	if body["editRedisPassword"] != true {
		t.Error("editRedisPassword must accompany a password change")
	}
	// A password implies the requirement; sending false with a password would
	// contradict itself.
	if body["redisPasswordEnabled"] != true || body["redisPassword"] != validRedisPassword {
		t.Errorf("body = %v", body)
	}
	if _, present := body["publicAccess"]; present {
		t.Error("publicAccess sent although its flag was not given")
	}

	// Disabling explicitly is allowed and needs no password.
	body, _, err = settingsBody(freshCmd(t, settingsFlags, map[string]string{"redis-password-enabled": "false"}).Flags(), "db-1", backupsOff)
	if err != nil {
		t.Fatalf("settingsBody disable: %v", err)
	}
	if body["editRedisPassword"] != true || body["redisPasswordEnabled"] != false {
		t.Errorf("body = %v", body)
	}

	for name, flags := range map[string]map[string]string{
		"enable without a password":      {"redis-password-enabled": "true"},
		"public access with it disabled": {"public-access": "true", "redis-password-enabled": "false"},
		"schedule while backups are off": {"backup-duration": "7"},
		"nothing at all":                 {},
	} {
		if _, _, err := settingsBody(freshCmd(t, settingsFlags, flags).Flags(), "db-1", backupsOff); err == nil {
			t.Errorf("%s: accepted, want a rejection", name)
		}
	}
}

// backupsOff / backupsOn stand in for the instance record settingsBody reads the current
// backup schedule from.
var (
	backupsOff = map[string]interface{}{"backupAuto": false}
	backupsOn  = map[string]interface{}{"backupAuto": true, "backupTime": "01:00", "backupDuration": float64(7)}
)

// TestSettingsBodyAlwaysCarriesTheBackupSchedule guards a live finding: the API answers
// HTTP 500 to any update-setting body without backupAuto, so a password-only or
// public-access-only change must still repeat the instance's own schedule.
func TestSettingsBodyAlwaysCarriesTheBackupSchedule(t *testing.T) {
	t.Setenv(passwordEnv, "")

	body, summary, err := settingsBody(
		freshCmd(t, settingsFlags, map[string]string{"redis-password": validRedisPassword}).Flags(), "db-1", backupsOn)
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
		freshCmd(t, settingsFlags, map[string]string{"redis-password": validRedisPassword}).Flags(), "db-1", backupsOff)
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

func TestReplicaBodyDefaultsToTheSource(t *testing.T) {
	t.Setenv(passwordEnv, "")

	cmd := freshCmd(t, replicaFlagsOn, map[string]string{
		"instance-id": "db-dc665cbe-63f9-4f43-8fd9-5586c4ff038b", "name": "redis-replica",
	})

	body, err := replicaBody(cmd.Flags(), "db-dc665cbe-63f9-4f43-8fd9-5586c4ff038b", liveInstance())
	if err != nil {
		t.Fatalf("replicaBody: %v", err)
	}

	for field, want := range map[string]interface{}{
		"name":             "redis-replica",
		"replicaSourceId":  "db-dc665cbe-63f9-4f43-8fd9-5586c4ff038b",
		"datastoreType":    "Redis",
		"datastoreVersion": "7.2",
		// quotaPackageId on an instance, packageId in the request.
		"packageId":            "284",
		"locateZoneId":         "HCM03-1B",
		"publicAccess":         false,
		"redisPasswordEnabled": false,
		"backupAuto":           true,
	} {
		if body[field] != want {
			t.Errorf("body[%q] = %v, want %v", field, body[field], want)
		}
	}
	if netIDs := body["netIds"].([]interface{}); len(netIDs) != 1 || netIDs[0] != "sub-7cc39ad2-a00f-4edd-8e8b-5f5aaa3eeffe" {
		t.Errorf("netIds = %v, want the source's subnet", body["netIds"])
	}
	if body["backupDuration"] != 2 || body["backupTime"] != "00:00" {
		t.Errorf("backup schedule = %v / %v", body["backupDuration"], body["backupTime"])
	}
	// No volume fields anywhere.
	for _, field := range []string{"volumeType", "volumeSize"} {
		if _, present := body[field]; present {
			t.Errorf("body carries %q", field)
		}
	}
}

// TestReplicaNeedsItsOwnPassword: a source that requires a password cannot hand its
// own over, so the replica needs one supplied.
func TestReplicaNeedsItsOwnPassword(t *testing.T) {
	t.Setenv(passwordEnv, "")

	source := liveInstance()
	source["redisPasswordEnabled"] = true

	cmd := freshCmd(t, replicaFlagsOn, map[string]string{"instance-id": "db-1", "name": "r"})
	if _, err := replicaBody(cmd.Flags(), "db-1", source); err == nil {
		t.Error("replicaBody accepted a password-protected source without a password")
	}

	cmd = freshCmd(t, replicaFlagsOn, map[string]string{
		"instance-id": "db-1", "name": "r", "redis-password": validRedisPassword,
	})
	body, err := replicaBody(cmd.Flags(), "db-1", source)
	if err != nil {
		t.Fatalf("replicaBody: %v", err)
	}
	if body["redisPasswordEnabled"] != true || body["redisPassword"] != validRedisPassword {
		t.Errorf("body = %v", body)
	}
}

// TestDefaultPortOf: Redis listens on 6379, and there is one engine here — unlike
// relational, which needs a per-engine map.
func TestDefaultPortOf(t *testing.T) {
	if got := defaultPortOf(liveInstance()); got != 6379 {
		t.Errorf("port from the instance = %d, want 6379", got)
	}
	if got := defaultPortOf(map[string]interface{}{}); got != redisPort {
		t.Errorf("fallback port = %d, want %d", got, redisPort)
	}
}

func TestRequireInstanceIDRejectsOtherResources(t *testing.T) {
	if err := requireInstanceID("db-dc665cbe-63f9"); err != nil {
		t.Errorf("valid instance ID rejected: %v", err)
	}
	// It cannot tell a relational instance apart — both are "db-…" — but everything
	// else is caught.
	for _, id := range []string{"pg-2e6f2253", "bk-740f7917", "cfg-b8f60c5c", "", "db id with spaces"} {
		if err := requireInstanceID(id); err == nil {
			t.Errorf("requireInstanceID(%q) = nil, want an error", id)
		}
	}
}

func TestMutatingCommandsAreGated(t *testing.T) {
	cmds := []*cobra.Command{
		createCmd, createReplicaCmd, detachReplicaCmd, resizeInstanceCmd, deleteCmd,
		updateSettingsCmd, updateConfigGroupCmd, updateSecruleCmd,
	}
	for _, spec := range actions {
		cmds = append(cmds, newActionCmd(spec))
	}

	for _, cmd := range cmds {
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
		{listReplicasCmd, []string{"instance-id"}},
		{listSecrulesCmd, []string{"instance-id"}},
		{createCmd, []string{"datastore-type", "datastore-version", "zone-id", "subnet-ids", "config-id"}},
		{createReplicaCmd, []string{"instance-id", "zone-id", "subnet-ids", "config-id"}},
		{resizeInstanceCmd, []string{"instance-id"}},
		{deleteCmd, []string{"instance-id"}},
		{detachReplicaCmd, []string{"instance-id"}},
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
