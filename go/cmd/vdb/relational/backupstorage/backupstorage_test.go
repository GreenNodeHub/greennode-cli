package backupstorage

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireStorageIDRejectsOtherResources(t *testing.T) {
	if err := requireStorageID("db-bk-storage-7a0c736d-ba65"); err != nil {
		t.Errorf("valid storage ID rejected: %v", err)
	}
	// An instance ID starts with "db-" too, which is exactly why the prefix checked
	// here is the longer "db-bk-storage-".
	for _, id := range []string{"db-ceb5fd7b-4483", "bk-740f7917", "", "db-bk-storage id"} {
		if err := requireStorageID(id); err == nil {
			t.Errorf("requireStorageID(%q) = nil, want an error", id)
		}
	}
}

// TestFlattenPackages: the API nests packages under engine groups, and table output
// picks the first array it finds — the outer one — so each package has to become a
// row of its own, keeping its engine group.
func TestFlattenPackages(t *testing.T) {
	payload := []interface{}{
		map[string]interface{}{
			"engineGroup": float64(1),
			"packages": []interface{}{
				map[string]interface{}{"packageId": float64(1), "packageName": "db.backup.quota.100GB", "packageQuota": "100"},
				map[string]interface{}{"packageId": float64(2), "packageName": "db.backup.quota.200GB", "packageQuota": "200"},
			},
		},
		map[string]interface{}{
			"engineGroup": float64(2),
			"packages": []interface{}{
				map[string]interface{}{"packageId": float64(9), "packageName": "redis.backup.quota.100GB"},
			},
		},
	}

	rows, ok := flattenPackages(payload).([]interface{})
	if !ok {
		t.Fatalf("flattenPackages returned %T", flattenPackages(payload))
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (one per package)", len(rows))
	}

	first := rows[0].(map[string]interface{})
	if first["packageId"] != float64(1) || first["engineGroup"] != float64(1) {
		t.Errorf("first row = %v, want the package with its engine group", first)
	}
	last := rows[2].(map[string]interface{})
	if last["engineGroup"] != float64(2) {
		t.Errorf("last row lost its engine group: %v", last)
	}

	// The original payload must survive for JSON output.
	if len(payload[0].(map[string]interface{})["packages"].([]interface{})) != 2 {
		t.Error("flattenPackages mutated the payload")
	}

	// Anything unexpected is handed back untouched rather than silently emptied.
	if got := flattenPackages("not a list"); got != "not a list" {
		t.Errorf("flattenPackages(non-list) = %v, want it unchanged", got)
	}
	if got := flattenPackages([]interface{}{map[string]interface{}{"engineGroup": float64(1)}}); got == nil {
		t.Error("a group with no packages should not produce nil")
	}
}

func TestCommandsAndGates(t *testing.T) {
	want := map[string]bool{
		"list": true, "list-packages": true, "create": true, "resize": true, "delete": true,
	}
	for _, sub := range BackupStorageCmd.Commands() {
		delete(want, sub.Name())
	}
	for name := range want {
		t.Errorf("backup-storage %s is not registered", name)
	}

	// create and resize place paid orders; delete can strand backups.
	for _, cmd := range []*cobra.Command{createCmd, resizeCmd, deleteCmd} {
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("backup-storage %s must define --%s", cmd.Name(), flag)
			}
		}
	}
}
