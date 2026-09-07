package catalog

import (
	"strings"
	"testing"
)

func TestSimpleListsAreWired(t *testing.T) {
	byName := map[string]bool{}
	for _, sub := range CatalogCmd.Commands() {
		byName[sub.Name()] = true
	}

	for _, spec := range simpleLists {
		if !byName[spec.use] {
			t.Errorf("%q is declared but not registered under catalog", spec.use)
			continue
		}
		if spec.short == "" || spec.long == "" {
			t.Errorf("%q is missing help text", spec.use)
		}
		if len(spec.columns) == 0 {
			t.Errorf("%q has no columns", spec.use)
		}
		if !strings.HasPrefix(spec.path, pgBase) {
			t.Errorf("%q path = %q, want it under %q", spec.use, spec.path, pgBase)
		}

		cmd, _, err := CatalogCmd.Find([]string{spec.use})
		if err != nil {
			t.Errorf("finding %q: %v", spec.use, err)
			continue
		}
		if hasZone := cmd.Flags().Lookup("zone-id") != nil; hasZone != spec.byZone {
			t.Errorf("%q has --zone-id = %v, want %v", spec.use, hasZone, spec.byZone)
		}
	}
}

// TestConfigGroupsPathIsRelational documents the one borrowed endpoint in this
// package: the cluster product has no config-group listing, so a cluster's
// --config-id values come from the relational one.
func TestConfigGroupsPathIsRelational(t *testing.T) {
	if !strings.HasPrefix(configGroupsPath, "/vdb-relational/") {
		t.Errorf("configGroupsPath = %q, want the relational listing", configGroupsPath)
	}
}

// TestKeepClusterGroups: only 'cluster' deploy types can be attached to a
// cluster, and the endpoint has no filter for it.
func TestKeepClusterGroups(t *testing.T) {
	payload := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"id": "pg-cfg-1", "deployType": "cluster"},
			map[string]interface{}{"id": "cfg-2", "deployType": "single_node"},
			map[string]interface{}{"id": "cfg-3"}, // no deployType at all
		},
		"pageObject": map[string]interface{}{"totalElements": float64(3)},
	}

	result, ok := keepClusterGroups(payload).(map[string]interface{})
	if !ok {
		t.Fatalf("keepClusterGroups returned %T", result)
	}

	// Items live under "content" here, not "data" — 6 of vdb's 8 paginated
	// endpoints key them that way, and filtering the wrong key would silently
	// return everything.
	kept, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("content = %T, want a slice", result["content"])
	}
	if len(kept) != 1 {
		t.Fatalf("kept %d groups, want 1", len(kept))
	}
	if kept[0].(map[string]interface{})["id"] != "pg-cfg-1" {
		t.Errorf("kept %v, want the cluster config group", kept[0])
	}
	if result["pageObject"] == nil {
		t.Error("keepClusterGroups dropped pageObject")
	}
	if len(payload["content"].([]interface{})) != 3 {
		t.Error("keepClusterGroups mutated the original payload")
	}
}

func TestListConfigGroupsHasAllEscapeHatch(t *testing.T) {
	// Filtering client-side is a guess about intent; --all must be there to see
	// everything the endpoint returned.
	if listConfigGroupsCmd.Flags().Lookup("all") == nil {
		t.Error("list-config-groups must offer --all")
	}
}
