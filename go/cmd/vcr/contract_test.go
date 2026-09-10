package vcr

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
}

type catalogRequestBody struct {
	Required bool `json:"required"`
	Content  map[string]struct {
		Schema struct {
			Title    string   `json:"title"`
			Type     string   `json:"type"`
			Required []string `json:"required"`
		} `json:"schema"`
	} `json:"content"`
}

func TestOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	records := readVCRCatalog(t)
	operations := allOperations()
	if len(operations) != 23 {
		t.Fatalf("operation descriptor count = %d, want 23", len(operations))
	}
	if len(records) != 23 {
		t.Fatalf("official catalog operation count = %d, want 23", len(records))
	}

	seen := make(map[string]bool, len(operations))
	reads, mutations := 0, 0
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
		assertOperationContract(t, op, record)
		if op.Mutation {
			mutations++
		} else {
			reads++
		}
	}
	for signature := range records {
		if !seen[signature] {
			t.Errorf("missing descriptor for %s", signature)
		}
	}
	if reads != 9 || mutations != 14 {
		t.Errorf("read/mutation counts = %d/%d, want 9/14", reads, mutations)
	}
}

func TestMutationsExposeRequiredSafetyFlags(t *testing.T) {
	for _, op := range allOperations() {
		cmd := newOperationCommand(op)
		if op.Mutation && cmd.Flags().Lookup("dry-run") == nil {
			t.Errorf("%s %s has no --dry-run", op.Method, op.Path)
		}
		if op.Destructive && cmd.Flags().Lookup("force") == nil {
			t.Errorf("%s %s has no --force", op.Method, op.Path)
		}
		if op.SecretResponse && cmd.Flags().Lookup("show-secret") == nil {
			t.Errorf("%s %s has no --show-secret", op.Method, op.Path)
		}
	}
}

func readVCRCatalog(t *testing.T) map[string]catalogRecord {
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
	if metadata.SchemaVersion != 1 || metadata.Source.URL != "https://docs.api.greennode.ai/service-docs/vcr.html" || metadata.Source.Title != "vCR API" || len(metadata.Source.SHA256) != 64 {
		t.Fatal("invalid public fixture provenance")
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func assertOperationContract(t *testing.T, op operation, record catalogRecord) {
	t.Helper()
	if op.OperationID != record.OperationID {
		t.Errorf("%s operation ID = %q, want %q", op.Path, op.OperationID, record.OperationID)
	}

	wantParameters := make(map[string]catalogParameter)
	var headers []catalogParameter
	for _, parameter := range record.Parameters {
		if parameter.In == "header" {
			headers = append(headers, parameter)
			continue
		}
		wantParameters[parameter.In+":"+parameter.Name] = parameter
	}
	if len(headers) != 1 || headers[0].Name != "Authorization" || !headers[0].Required {
		t.Errorf("%s %s headers = %#v, want one required Authorization header", op.Method, op.Path, headers)
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
	}
	for name, parameter := range wantParameters {
		if required, ok := gotParameters[name]; !ok || required != parameter.Required {
			t.Errorf("%s %s parameter %s required = %v/%v, want present/%v", op.Method, op.Path, name, ok, required, parameter.Required)
		}
	}

	if record.RequestBody == nil {
		if op.Body != nil {
			t.Errorf("%s %s has body contract %q, want no body", op.Method, op.Path, op.Body.Name)
		}
	} else {
		if op.Body == nil {
			t.Errorf("%s %s has no body contract", op.Method, op.Path)
		} else {
			schema := record.RequestBody.Content["application/json"].Schema
			if !record.RequestBody.Required || schema.Type != "object" {
				t.Errorf("%s %s request body is not a required JSON object", op.Method, op.Path)
			}
			if op.Body.Name != schema.Title {
				t.Errorf("%s %s body name = %q, want %q", op.Method, op.Path, op.Body.Name, schema.Title)
			}
			assertStringSet(t, op.Method+" "+op.Path+" required body fields", op.Body.RequiredFields, schema.Required)
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

func assertStringSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
