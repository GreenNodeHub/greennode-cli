package vdbclient

import (
	"bytes"
	"strings"
	"testing"
)

func TestActionBody(t *testing.T) {
	// The endpoints want the instance ID in the body as well as the path, inside a
	// list, under "instancesId".
	body := ActionBody(ResourceTypeInstance, "db-1", "reboot", nil)
	if body["action"] != "reboot" || body["resType"] != ResourceTypeInstance {
		t.Errorf("action body = %v", body)
	}
	instance := body["databaseInstances"].([]interface{})[0].(map[string]interface{})
	if instance["instancesId"] != "db-1" {
		t.Errorf("instancesId = %v", instance["instancesId"])
	}
	if _, present := instance["config"]; present {
		t.Error("a nil config must not appear in the body")
	}

	// Each endpoint accepts only the action matching its path — the shared request
	// class documents all of them, which is a spec artifact, not permission.
	for _, action := range []string{"start", "stop", "reboot", "detach_replica", "delete"} {
		if got := ActionBody(ResourceTypeInstance, "db-1", action, nil)["action"]; got != action {
			t.Errorf("action = %v, want %q", got, action)
		}
	}

	withConfig := ActionBody(ResourceTypeInstance, "db-1", "delete", map[string]interface{}{"createFinalBackup": true})
	instance = withConfig["databaseInstances"].([]interface{})[0].(map[string]interface{})
	config, ok := instance["config"].(map[string]interface{})
	if !ok || config["createFinalBackup"] != true {
		t.Errorf("delete config = %v", instance["config"])
	}
}

// TestResizeBodyUsesResourceType pins the trap: the action endpoints spell the
// field "resType", the resize endpoints spell it "resourceType".
func TestResizeBodyUsesResourceType(t *testing.T) {
	body := ResizeBody(ResourceTypeInstance, "db-1", "resize", map[string]interface{}{"packageId": "211"})

	if body["resourceType"] != ResourceTypeInstance {
		t.Errorf("resourceType = %v, want %q", body["resourceType"], ResourceTypeInstance)
	}
	if _, present := body["resType"]; present {
		t.Error("resize must not send resType; that field belongs to the action endpoints")
	}
	if body["action"] != "resize" {
		t.Errorf("action = %v, want resize", body["action"])
	}
	instance := body["databaseInstances"].([]interface{})[0].(map[string]interface{})
	if instance["config"].(map[string]interface{})["packageId"] != "211" {
		t.Errorf("config = %v", instance["config"])
	}
}

func TestPreviewMasksSecrets(t *testing.T) {
	// --dry-run output gets pasted into tickets; master passwords must not be in it.
	var out bytes.Buffer
	previewBodyTo(&out, "create", "instance \"db\"", map[string]interface{}{
		"name":     "db",
		"user":     map[string]interface{}{"name": "admin", "password": "s3cret"},
		"replicas": []interface{}{map[string]interface{}{"redisPassword": "s3cret"}},
	})

	text := out.String()
	if strings.Contains(text, "s3cret") {
		t.Errorf("preview leaked a secret:\n%s", text)
	}
	for _, want := range []string{"=== DRY RUN ===", "Would create instance", `"admin"`, `"***"`} {
		if !strings.Contains(text, want) {
			t.Errorf("preview missing %q:\n%s", want, text)
		}
	}
}

func TestMaskSecretsDoesNotMutateTheRealBody(t *testing.T) {
	user := map[string]interface{}{"password": "keep-me"}
	body := map[string]interface{}{"user": user}

	maskSecrets(body)

	if user["password"] != "keep-me" {
		t.Error("maskSecrets edited the body that is about to be sent")
	}
}

// TestResourceTypesAreNotAllDbaas: each kind of resource has its own value, and the
// wrong one is a validation error that does not name the field.
func TestResourceTypesAreNotAllDbaas(t *testing.T) {
	if ResourceTypeInstance != "dbaas" || ResourceTypeBackup != "dbaas-backup" ||
		ResourceTypeBackupStorage != "dbaas-backup-storage" {
		t.Errorf("resource types drifted: %q %q %q",
			ResourceTypeInstance, ResourceTypeBackup, ResourceTypeBackupStorage)
	}

	if got := ActionBody(ResourceTypeBackupStorage, "db-bk-storage-1", "delete", nil)["resType"]; got != ResourceTypeBackupStorage {
		t.Errorf("backup-storage delete resType = %v", got)
	}
	if got := ResizeBody(ResourceTypeBackupStorage, "db-bk-storage-1", "resize", nil)["resourceType"]; got != ResourceTypeBackupStorage {
		t.Errorf("backup-storage resize resourceType = %v", got)
	}
}

// TestResizeConfigBodyCarriesNoResourceID: restore is the one endpoint whose detail
// has no instancesId — the backup is named inside the config instead.
func TestResizeConfigBodyCarriesNoResourceID(t *testing.T) {
	body := ResizeConfigBody(ResourceTypeBackup, "restore_backup", map[string]interface{}{
		"backupId": "bk-1", "name": "restored",
	})

	if body["action"] != "restore_backup" || body["resourceType"] != ResourceTypeBackup {
		t.Errorf("body = %v", body)
	}
	detail := body["databaseInstances"].([]interface{})[0].(map[string]interface{})
	if _, present := detail["instancesId"]; present {
		t.Error("restore must not send instancesId; the detail carries only config")
	}
	if detail["config"].(map[string]interface{})["backupId"] != "bk-1" {
		t.Errorf("config = %v", detail["config"])
	}
}
