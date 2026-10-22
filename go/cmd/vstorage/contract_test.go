package vstorage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/config"
)

type catalogRecord struct {
	ServiceID   string              `json:"service_id"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Parameters  []catalogParameter  `json:"parameters"`
	RequestBody *catalogRequestBody `json:"request_body"`
	Responses   map[string]struct {
		Content map[string]json.RawMessage `json:"content"`
	} `json:"responses"`
}

type catalogParameter struct {
	In       string `json:"in"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

type catalogRequestBody struct {
	Required *bool `json:"required"`
	Content  map[string]struct {
		Schema struct {
			Type string `json:"type"`
		} `json:"schema"`
	} `json:"content"`
}

func TestOperationDescriptorsMatchOfficialRegionalCatalogs(t *testing.T) {
	catalogs := readVStorageCatalogs(t)
	tests := []struct {
		serviceID string
		family    []operation
		wantCount int
	}{
		{serviceID: "vstorage_hcm03", family: appendOperations(commonOperations(), swiftOperations()), wantCount: 46},
		{serviceID: "vstorage_han02", family: appendOperations(commonOperations(), cephOperations()), wantCount: 66},
		{serviceID: "vstorage_hcm04", family: appendOperations(commonOperations(), cephOperations()), wantCount: 66},
	}

	for _, tt := range tests {
		t.Run(tt.serviceID, func(t *testing.T) {
			got := operationSet(t, tt.family)
			want := catalogs[tt.serviceID]
			if len(got) != tt.wantCount {
				t.Fatalf("descriptor count = %d, want %d", len(got), tt.wantCount)
			}
			if len(want) != tt.wantCount {
				t.Fatalf("catalog count = %d, want %d", len(want), tt.wantCount)
			}
			for signature := range want {
				if !got[signature] {
					t.Errorf("missing descriptor for %s", signature)
				}
			}
			for signature := range got {
				if _, ok := want[signature]; !ok {
					t.Errorf("descriptor is absent from official catalog: %s", signature)
				}
			}
			for _, op := range tt.family {
				assertOperationContract(t, op, want[op.Method+" "+op.Path])
			}
		})
	}
}

func TestAllOperationsExposeRequiredSafetyFlags(t *testing.T) {
	operations := allOperations()
	if len(operations) != 97 {
		t.Fatalf("unique command operation count = %d, want 97", len(operations))
	}
	for _, op := range operations {
		cmd := newOperationCommand(op)
		if op.Mutation && cmd.Flags().Lookup("dry-run") == nil {
			t.Errorf("%s %s has no --dry-run", op.Method, op.Path)
		}
		if op.Destructive && cmd.Flags().Lookup("force") == nil {
			t.Errorf("%s %s has no --force", op.Method, op.Path)
		}
	}
}

func TestRegionalEndpointsMatchPublicFixtures(t *testing.T) {
	for region, id := range map[string]string{"HCM-3": "vstorage_hcm03", "HAN": "vstorage_han02", "HCM-4": "vstorage_hcm04"} {
		data, err := os.ReadFile(filepath.Join("testdata", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var fixture struct {
			Servers []struct {
				URL string `json:"url"`
			} `json:"servers"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		if len(fixture.Servers) == 0 || config.REGIONS[region]["vstorage_endpoint"] != fixture.Servers[0].URL {
			t.Errorf("%s endpoint differs from public fixture", region)
		}
	}
}

func readVStorageCatalogs(t *testing.T) map[string]map[string]catalogRecord {
	t.Helper()
	result := make(map[string]map[string]catalogRecord)
	for _, id := range []string{"vstorage_hcm03", "vstorage_han02", "vstorage_hcm04"} {
		result[id] = readFixture(t, id)
	}
	return result
}

func readFixture(t *testing.T, id string) map[string]catalogRecord {
	t.Helper()

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

func assertOperationContract(t *testing.T, op operation, record catalogRecord) {
	t.Helper()
	wantParameters := make(map[string]bool)
	for _, parameter := range record.Parameters {
		if parameter.In == "header" {
			continue
		}
		wantParameters[parameter.In+":"+parameter.Name] = parameter.Required
	}
	gotParameters := make(map[string]bool)
	for _, parameter := range op.Paths {
		gotParameters["path:"+parameter.Placeholder] = true
	}
	for _, parameter := range op.Queries {
		gotParameters["query:"+parameter.WireName] = parameter.Required
	}
	if len(gotParameters) != len(wantParameters) {
		t.Errorf("%s %s parameters = %#v, want %#v", op.Method, op.Path, gotParameters, wantParameters)
	} else {
		for name, required := range wantParameters {
			if gotRequired, ok := gotParameters[name]; !ok || gotRequired != required {
				t.Errorf("%s %s parameter %s required = %v/%v, want present/%v", op.Method, op.Path, name, ok, gotRequired, required)
			}
		}
	}

	wantBody := noBody
	if record.RequestBody != nil {
		wantBody = optionalJSONBody
		if record.RequestBody.Required != nil && *record.RequestBody.Required {
			schemaType := record.RequestBody.Content["application/json"].Schema.Type
			switch schemaType {
			case "object":
				wantBody = requiredObjectBody
			case "array":
				wantBody = requiredArrayBody
			}
		}
	}
	if op.Body != wantBody {
		t.Errorf("%s %s body kind = %v, want %v", op.Method, op.Path, op.Body, wantBody)
	}

	wantEmptyStatuses := make(map[int]bool)
	for status, response := range record.Responses {
		statusCode, err := strconv.Atoi(status)
		if err != nil || statusCode < 200 || statusCode >= 300 || len(response.Content) > 0 {
			continue
		}
		wantEmptyStatuses[statusCode] = true
	}
	gotEmptyStatuses := make(map[int]bool)
	for statusCode := 200; statusCode < 300; statusCode++ {
		if allowsEmptySuccess(op.Method, statusCode) {
			gotEmptyStatuses[statusCode] = true
		}
	}
	if len(gotEmptyStatuses) != len(wantEmptyStatuses) {
		t.Errorf("%s %s empty success statuses = %#v, want %#v", op.Method, op.Path, gotEmptyStatuses, wantEmptyStatuses)
	} else {
		for statusCode := range wantEmptyStatuses {
			if !gotEmptyStatuses[statusCode] {
				t.Errorf("%s %s does not accept documented empty HTTP %d", op.Method, op.Path, statusCode)
			}
		}
	}
}

func operationSet(t *testing.T, operations []operation) map[string]bool {
	t.Helper()
	result := make(map[string]bool, len(operations))
	for _, op := range operations {
		signature := op.Method + " " + op.Path
		if result[signature] {
			t.Fatalf("duplicate operation descriptor: %s", signature)
		}
		result[signature] = true
	}
	return result
}

func appendOperations(first, second []operation) []operation {
	combined := make([]operation, 0, len(first)+len(second))
	combined = append(combined, first...)
	combined = append(combined, second...)
	return combined
}
