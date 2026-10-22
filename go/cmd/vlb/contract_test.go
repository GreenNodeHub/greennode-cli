package vlb

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type catalogRecord struct {
	OperationID string              `json:"operation_id"`
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
	Schema   struct {
		Type string `json:"type"`
	} `json:"schema"`
}

type catalogRequestBody struct {
	Required bool `json:"required"`
	Content  map[string]struct {
		Schema struct {
			Ref  string `json:"$ref"`
			Type string `json:"type"`
		} `json:"schema"`
	} `json:"content"`
}

type schemaCatalog struct {
	Components struct {
		Schemas map[string]struct {
			Type     string   `json:"type"`
			Required []string `json:"required"`
		} `json:"schemas"`
	} `json:"components"`
}

func TestOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	records := readVLBCatalog(t)
	schemas := readVLBSchemas(t)
	operations := allOperations()
	if len(operations) != 37 {
		t.Fatalf("operation descriptor count = %d, want 37", len(operations))
	}
	if len(records) != 37 {
		t.Fatalf("official catalog operation count = %d, want 37", len(records))
	}

	seen := make(map[string]bool, len(operations))
	reads := 0
	for _, op := range operations {
		signature := op.Method + " " + op.Path
		if seen[signature] {
			t.Fatalf("duplicate operation descriptor: %s", signature)
		}
		seen[signature] = true
		record, ok := records[signature]
		if !ok {
			t.Errorf("descriptor is absent from official catalog: %s", signature)
			continue
		}
		assertOperationContract(t, op, record, schemas)
		if op.Method == "GET" {
			reads++
		}
	}
	for signature := range records {
		if !seen[signature] {
			t.Errorf("missing descriptor for %s", signature)
		}
	}
	if reads != 18 {
		t.Errorf("GET descriptor count = %d, want 18", reads)
	}
}

func TestAllWritesExposeRequiredSafetyFlags(t *testing.T) {
	writes := 0
	for _, op := range allOperations() {
		cmd := newOperationCommand(op)
		if isMutation(op) {
			writes++
			if cmd.Flags().Lookup("dry-run") == nil {
				t.Errorf("%s %s has no --dry-run", op.Method, op.Path)
			}
		}
		if op.Destructive && cmd.Flags().Lookup("force") == nil {
			t.Errorf("%s %s has no --force", op.Method, op.Path)
		}
	}
	if writes != 19 {
		t.Errorf("write descriptor count = %d, want 19", writes)
	}
}

func readVLBCatalog(t *testing.T) map[string]catalogRecord {
	t.Helper()
	var fixture struct {
		Operations []catalogRecord `json:"operations"`
	}
	readContractFixture(t, &fixture)
	result := make(map[string]catalogRecord)
	for _, record := range fixture.Operations {
		signature := record.Method + " " + record.Path
		if _, exists := result[signature]; exists {
			t.Fatalf("duplicate fixture operation: %s", signature)
		}
		result[signature] = record
	}
	return result
}

func readVLBSchemas(t *testing.T) schemaCatalog {
	t.Helper()
	var schemas schemaCatalog
	readContractFixture(t, &schemas)
	return schemas
}

func readContractFixture(t *testing.T, target any) {
	t.Helper()
	data, err := os.ReadFile("testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		SchemaVersion int                                                     `json:"schema_version"`
		Source        struct{ URL, Title, Version, Retrieved, SHA256 string } `json:"source"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.SchemaVersion != 1 || metadata.Source.URL != "https://docs.api.greennode.ai/service-docs/vlb-api.html" || metadata.Source.Title != "VLB Service API" || len(metadata.Source.SHA256) != 64 {
		t.Fatal("invalid public fixture provenance")
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func assertOperationContract(t *testing.T, op operation, record catalogRecord, schemas schemaCatalog) {
	t.Helper()
	if op.OperationID != record.OperationID {
		t.Errorf("%s operation ID = %q, want %q", op.Path, op.OperationID, record.OperationID)
	}

	wantParameters := make(map[string]catalogParameter)
	for _, parameter := range record.Parameters {
		wantParameters[parameter.In+":"+parameter.Name] = parameter
	}
	gotParameters := make(map[string]bool)
	for _, parameter := range op.Paths {
		gotParameters["path:"+parameter.Placeholder] = true
	}
	for _, parameter := range op.Queries {
		key := "query:" + parameter.WireName
		gotParameters[key] = parameter.Required
		if want := wantParameters[key].Schema.Type; want == "object" && parameter.Kind != queryObject {
			t.Errorf("%s %s query %s kind = %d, want JSON object", op.Method, op.Path, parameter.WireName, parameter.Kind)
		}
	}
	if len(gotParameters) != len(wantParameters) {
		t.Errorf("%s %s parameters = %#v, want %#v", op.Method, op.Path, gotParameters, wantParameters)
	} else {
		for key, parameter := range wantParameters {
			if required, ok := gotParameters[key]; !ok || required != parameter.Required {
				t.Errorf("%s %s parameter %s required = %v/%v, want present/%v", op.Method, op.Path, key, ok, required, parameter.Required)
			}
		}
	}

	if record.RequestBody == nil {
		if op.Body != nil {
			t.Errorf("%s %s has body schema %q, want no body", op.Method, op.Path, op.Body.Name)
		}
	} else {
		if op.Body == nil {
			t.Errorf("%s %s has no body contract", op.Method, op.Path)
		} else {
			ref := record.RequestBody.Content["application/json"].Schema.Ref
			wantSchema := strings.TrimPrefix(ref, "#/components/schemas/")
			if op.Body.Name != wantSchema {
				t.Errorf("%s %s body schema = %q, want %q", op.Method, op.Path, op.Body.Name, wantSchema)
			}
			if !record.RequestBody.Required {
				t.Errorf("%s %s request body unexpectedly optional in catalog", op.Method, op.Path)
			}
			want := append([]string(nil), schemas.Components.Schemas[wantSchema].Required...)
			got := append([]string(nil), op.Body.RequiredFields...)
			sort.Strings(want)
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("%s %s required body fields = %v, want %v", op.Method, op.Path, got, want)
			}
		}
	}

	successes := make(map[int]bool)
	for status, response := range record.Responses {
		code, err := strconv.Atoi(status)
		if err == nil && code >= 200 && code < 300 {
			successes[code] = len(response.Content) > 0
		}
	}
	if len(successes) != 1 {
		t.Fatalf("%s %s success responses = %#v, want exactly one", op.Method, op.Path, successes)
	}
	wantBody, ok := successes[op.Status]
	if !ok || wantBody != op.ResponseBody {
		t.Errorf("%s %s response = HTTP %d body=%v, want %#v", op.Method, op.Path, op.Status, op.ResponseBody, successes)
	}
}
