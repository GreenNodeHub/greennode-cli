package backup

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// liveBackup is a captured record of a real backup, used to test the defaulting
// that `restore` does. Note netIds: the API recorded the NETWORK the instance sits
// in, not the subnet the restore request needs.
func liveBackup() map[string]interface{} {
	return map[string]interface{}{
		"id":               "bk-740f7917-8d6a-4a8d-a464-ce823c8b68b8",
		"name":             "phase4-test-backup",
		"dbInstanceId":     "db-ceb5fd7b-4483-462a-be42-fb66bc44c4de",
		"backupType":       "FULL",
		"status":           "COMPLETED",
		"datastoreType":    "MySQL",
		"datastoreVersion": "8.0",
		"packageId":        "180",
		"storageType":      "Gen2-NVMe2-IOPS3000-HCM03-1B",
		"storageSize":      float64(30),
		"configId":         "cfg-b8f60c5c-e96a-42a8-aca6-8f0f7e5a7530",
		"username":         "test_user",
		"netIds":           []interface{}{"net-ed67af02-c24d-48ba-9db8-ca9d77fcce0c"},
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

func TestRequireBackupIDRejectsOtherResources(t *testing.T) {
	if err := requireBackupID("bk-740f7917-8d6a"); err != nil {
		t.Errorf("valid backup ID rejected: %v", err)
	}
	// An instance ID reaching a backup path is the mistake this prevents.
	for _, id := range []string{"db-ceb5fd7b-4483", "pg-2e6f2253", "", "bk id with spaces"} {
		if err := requireBackupID(id); err == nil {
			t.Errorf("requireBackupID(%q) = nil, want an error", id)
		}
	}
}

func TestCreateBodyIncrementalNeedsParent(t *testing.T) {
	body, err := createBody(freshCmd(t, createFlagsForTest, map[string]string{
		"name": "b1", "backup-type": "FULL", "description": "nightly",
	}), "db-1")
	if err != nil {
		t.Fatalf("createBody: %v", err)
	}
	if body["dbInstanceId"] != "db-1" || body["name"] != "b1" || body["backupType"] != "FULL" {
		t.Errorf("body = %v", body)
	}
	if body["description"] != "nightly" {
		t.Errorf("description = %v", body["description"])
	}
	if _, present := body["parentId"]; present {
		t.Error("body carries parentId although the flag was not set")
	}

	// Lowercase input is accepted and normalised, since the API's values are uppercase.
	body, err = createBody(freshCmd(t, createFlagsForTest, map[string]string{
		"name": "b2", "backup-type": "incremental", "parent-id": "bk-parent", "description": "d",
	}), "db-1")
	if err != nil {
		t.Fatalf("createBody incremental: %v", err)
	}
	if body["backupType"] != "INCREMENTAL" || body["parentId"] != "bk-parent" {
		t.Errorf("body = %v", body)
	}

	for name, flags := range map[string]map[string]string{
		"incremental without a parent": {"name": "b", "backup-type": "INCREMENTAL", "description": "d"},
		"parent on a full backup":      {"name": "b", "backup-type": "FULL", "parent-id": "bk-parent", "description": "d"},
		"unknown type":                 {"name": "b", "backup-type": "SNAPSHOT", "description": "d"},
		"parent that is not a backup":  {"name": "b", "backup-type": "INCREMENTAL", "parent-id": "db-1", "description": "d"},
		// An empty description is accepted by the API and then fails the backup
		// asynchronously — the CLI refuses it up front.
		"empty description":      {"name": "b", "backup-type": "FULL", "description": ""},
		"whitespace description": {"name": "b", "backup-type": "FULL", "description": "   "},
	} {
		if _, err := createBody(freshCmd(t, createFlagsForTest, flags), "db-1"); err == nil {
			t.Errorf("%s: createBody = nil error, want a rejection", name)
		}
	}
}

// createFlagsForTest mirrors create's flag set. create defines its flags inline in
// init(), so this keeps the test independent of the mounted command.
func createFlagsForTest(f *pflag.FlagSet) {
	f.String("name", "", "")
	f.String("backup-type", "FULL", "")
	f.String("parent-id", "", "")
	f.String("description", "", "")
}

// TestRestoreBodyDefaultsFromTheBackup: a restore needs a full instance spec, and
// every part of it except the name is already recorded in the backup.
func TestRestoreBodyDefaultsFromTheBackup(t *testing.T) {
	cmd := freshCmd(t, restoreFlags, map[string]string{
		"backup-id": "bk-740f7917-8d6a-4a8d-a464-ce823c8b68b8",
		"name":      "restored",
		// Placement is the user's choice, so both are required rather than inherited.
		"zone-id":    "HCM03-1B",
		"subnet-ids": "sub-7cc39ad2",
	})

	body, err := restoreBody(cmd.Flags(), "bk-740f7917-8d6a-4a8d-a464-ce823c8b68b8", liveBackup())
	if err != nil {
		t.Fatalf("restoreBody: %v", err)
	}

	// The envelope: action restore_backup, resource type dbaas-backup, and a detail
	// with only a config.
	if body["action"] != "restore_backup" || body["resourceType"] != "dbaas-backup" {
		t.Errorf("envelope = %v", body)
	}
	detail := body["databaseInstances"].([]interface{})[0].(map[string]interface{})
	if _, present := detail["instancesId"]; present {
		t.Error("restore must not send instancesId")
	}

	config := detail["config"].(map[string]interface{})
	for field, want := range map[string]interface{}{
		"backupId":         "bk-740f7917-8d6a-4a8d-a464-ce823c8b68b8",
		"name":             "restored",
		"datastoreType":    "MySQL",
		"datastoreVersion": "8.0",
		"packageId":        "180",
		// storageType -> volumeType and storageSize -> volumeSize: the field names
		// differ between a backup record and this request.
		"volumeType":   "Gen2-NVMe2-IOPS3000-HCM03-1B",
		"volumeSize":   30,
		"configId":     "cfg-b8f60c5c-e96a-42a8-aca6-8f0f7e5a7530",
		"locateZoneId": "HCM03-1B",
	} {
		if config[field] != want {
			t.Errorf("config[%q] = %v, want %v", field, config[field], want)
		}
	}
	if netIDs := config["netIds"].([]interface{}); len(netIDs) != 1 || netIDs[0] != "sub-7cc39ad2" {
		t.Errorf("netIds = %v", config["netIds"])
	}
}

// TestRestoreBodyNeedsPlacement: zone and subnet are the user's decision, and the
// backup cannot supply either — its netIds field records the NETWORK the original
// instance sat in, not a subnet. Sending that network would be rejected by the API.
func TestRestoreBodyNeedsPlacement(t *testing.T) {
	for name, flags := range map[string]map[string]string{
		"no zone":   {"backup-id": "bk-1", "name": "r", "subnet-ids": "sub-1"},
		"no subnet": {"backup-id": "bk-1", "name": "r", "zone-id": "HCM03-1B"},
	} {
		cmd := freshCmd(t, restoreFlags, flags)
		if _, err := restoreBody(cmd.Flags(), "bk-1", liveBackup()); err == nil {
			t.Errorf("%s: restoreBody = nil error, want a rejection", name)
		}
	}

	// And the network from the backup is never silently reused as a subnet.
	cmd := freshCmd(t, restoreFlags, map[string]string{
		"backup-id": "bk-1", "name": "r", "zone-id": "HCM03-1B", "subnet-ids": "sub-1",
	})
	body, err := restoreBody(cmd.Flags(), "bk-1", liveBackup())
	if err != nil {
		t.Fatalf("restoreBody: %v", err)
	}
	config := body["databaseInstances"].([]interface{})[0].(map[string]interface{})["config"].(map[string]interface{})
	netIDs := config["netIds"].([]interface{})
	if len(netIDs) != 1 || netIDs[0] != "sub-1" {
		t.Errorf("netIds = %v, want only the subnet that was passed", netIDs)
	}
}

// TestRestoreIsRequiredToPlaceTheInstance pins the flags as required, since the API
// gives a misleading error when the zone is missing (it blames the flavor and the
// volume type instead).
func TestRestoreIsRequiredToPlaceTheInstance(t *testing.T) {
	for _, flag := range []string{"backup-id", "name", "zone-id", "subnet-ids"} {
		annotations := restoreCmd.Flags().Lookup(flag).Annotations[cobra.BashCompOneRequiredFlag]
		if len(annotations) == 0 || annotations[0] != "true" {
			t.Errorf("--%s must be marked required on backup restore", flag)
		}
	}
}

func TestSubnetIDsOnly(t *testing.T) {
	got := subnetIDsOnly([]string{"net-1", "sub-1", "", "sub-2", "db-1"})
	if len(got) != 2 || got[0] != "sub-1" || got[1] != "sub-2" {
		t.Errorf("subnetIDsOnly = %v, want the two sub- entries", got)
	}
}

func TestRestoreBodyBackupSchedule(t *testing.T) {
	base := map[string]string{"backup-id": "bk-1", "name": "r", "zone-id": "HCM03-1B", "subnet-ids": "sub-1"}
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

	body, err := restoreBody(freshCmd(t, restoreFlags, with(map[string]string{
		"backup-auto": "true", "backup-duration": "7", "backup-time": "02:00",
	})).Flags(), "bk-1", liveBackup())
	if err != nil {
		t.Fatalf("restoreBody: %v", err)
	}
	config := body["databaseInstances"].([]interface{})[0].(map[string]interface{})["config"].(map[string]interface{})
	if config["backupDuration"] != 7 || config["backupTime"] != "02:00" {
		t.Errorf("schedule = %v / %v", config["backupDuration"], config["backupTime"])
	}

	for name, flags := range map[string]map[string]string{
		"retention without backup-auto": {"backup-duration": "7"},
		"backup on without a schedule":  {"backup-auto": "true"},
		"retention out of range":        {"backup-auto": "true", "backup-duration": "1", "backup-time": "02:00"},
	} {
		if _, err := restoreBody(freshCmd(t, restoreFlags, with(flags)).Flags(), "bk-1", liveBackup()); err == nil {
			t.Errorf("%s: accepted, want a rejection", name)
		}
	}
}

func TestCommandsAndGates(t *testing.T) {
	want := map[string]bool{
		"list": true, "get": true, "create": true,
		"delete": true, "restore": true, "get-free-storage": true,
	}
	for _, sub := range BackupCmd.Commands() {
		delete(want, sub.Name())
	}
	for name := range want {
		t.Errorf("backup %s is not registered", name)
	}

	// Creating a backup consumes billable storage; deleting one is irreversible;
	// restoring places an order. All three are gated.
	for _, cmd := range []*cobra.Command{createCmd, deleteCmd, restoreCmd} {
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("backup %s must define --%s", cmd.Name(), flag)
			}
		}
	}
}

func TestFlagCompletionsAreRegistered(t *testing.T) {
	cases := []struct {
		cmd   *cobra.Command
		flags []string
	}{
		{listCmd, []string{"instance-id"}},
		{getCmd, []string{"backup-id"}},
		{createCmd, []string{"instance-id", "backup-type", "parent-id"}},
		{deleteCmd, []string{"backup-id"}},
		{restoreCmd, []string{"backup-id", "volume-type", "zone-id", "subnet-ids", "config-id"}},
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
