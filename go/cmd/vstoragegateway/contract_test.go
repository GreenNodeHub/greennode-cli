package vstoragegateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type catalogRecord struct {
	ServiceID  string `json:"service_id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Parameters []struct {
		In       string `json:"in"`
		Name     string `json:"name"`
		Required bool   `json:"required"`
	} `json:"parameters"`
}

func TestOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	catalog := readCatalog(t)
	operations := allOperations()
	if len(operations) != 3 || len(catalog) != 3 {
		t.Fatalf("operation counts = %d/%d, want 3/3", len(operations), len(catalog))
	}
	for _, op := range operations {
		record, ok := catalog[op.Method+" "+op.Path]
		if !ok {
			t.Errorf("missing descriptor for %s %s", op.Method, op.Path)
			continue
		}
		want := map[string]bool{}
		for _, parameter := range record.Parameters {
			if parameter.In == "query" {
				want[parameter.Name] = parameter.Required
			}
		}
		got := map[string]bool{}
		for _, parameter := range op.Queries {
			got[parameter.WireName] = parameter.Required
		}
		if len(got) != len(want) {
			t.Errorf("%s %s query count = %d, want %d", op.Method, op.Path, len(got), len(want))
		}
		for name, required := range want {
			if gotRequired, exists := got[name]; !exists || gotRequired != required {
				t.Errorf("%s %s query %s = %v/%v, want present/%v", op.Method, op.Path, name, exists, gotRequired, required)
			}
		}
	}
}

func readCatalog(t *testing.T) map[string]catalogRecord {
	t.Helper()
	id := "vmonitor_vstorage_gateway"
	data, err := os.ReadFile(filepath.Join("testdata", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct {
			ID     string `json:"id"`
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
		} `json:"source"`
		Operations []catalogRecord `json:"operations"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 || fixture.Source.ID != id || len(fixture.Source.SHA256) != 64 || fixture.Source.URL == "" {
		t.Fatal("invalid public fixture provenance")
	}
	result := make(map[string]catalogRecord)
	for _, record := range fixture.Operations {
		signature := record.Method + " " + record.Path
		if _, duplicate := result[signature]; duplicate {
			t.Fatalf("duplicate fixture operation: %s", signature)
		}
		result[signature] = record
	}
	return result
}
