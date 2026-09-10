package vmonitorlog

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"testing"
)

type catalogRecord struct {
	OperationID string             `json:"operation_id"`
	Method      string             `json:"method"`
	Path        string             `json:"path"`
	Parameters  []catalogParameter `json:"parameters"`
	RequestBody *struct {
		Required       bool           `json:"required"`
		MediaType      string         `json:"media_type"`
		Type           string         `json:"type"`
		RequiredFields []string       `json:"required_fields"`
		Example        map[string]any `json:"example"`
	} `json:"request_body"`
	Responses []responseFixture `json:"responses"`
}

type catalogParameter struct {
	In       string `json:"in"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Format   string `json:"format"`
}

func TestOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	catalog := readVMonitorLogCatalog(t)
	operations := allOperations()
	if got := len(operations); got != 63 {
		t.Fatalf("operation descriptor count = %d, want 63", got)
	}
	if got := len(catalog); got != 63 {
		t.Fatalf("catalog operation count = %d, want 63", got)
	}

	seen := make(map[string]bool, len(operations))
	for _, op := range operations {
		signature := op.Method + " " + op.Path
		if seen[signature] {
			t.Fatalf("duplicate operation descriptor: %s", signature)
		}
		seen[signature] = true
		record, ok := catalog[signature]
		if !ok {
			t.Errorf("descriptor is absent from official catalog: %s", signature)
			continue
		}
		assertOperationContract(t, op, record)
	}
	for signature := range catalog {
		if !seen[signature] {
			t.Errorf("missing descriptor for %s", signature)
		}
	}
}

func TestStateChangingOperationsExposeRequiredSafetyFlags(t *testing.T) {
	mutations := 0
	dryRunOperations := 0
	destructive := 0
	downloads := 0
	secretResponses := 0
	searchFilters := 0
	for _, op := range allOperations() {
		cmd := newOperationCommand(op)
		if op.Mutation {
			mutations++
		}
		if op.Mutation || op.Download {
			dryRunOperations++
			if cmd.Flags().Lookup("dry-run") == nil {
				t.Errorf("%s %s has no --dry-run", op.Method, op.Path)
			}
		} else if cmd.Flags().Lookup("dry-run") != nil {
			t.Errorf("read-only %s %s unexpectedly exposes --dry-run", op.Method, op.Path)
		}
		if op.Destructive {
			destructive++
			if cmd.Flags().Lookup("force") == nil {
				t.Errorf("%s %s has no --force", op.Method, op.Path)
			}
		}
		if op.Download {
			downloads++
			if cmd.Flags().Lookup("output-file") == nil || cmd.Flags().Lookup("force") == nil {
				t.Errorf("download %s has no output or overwrite-protection flag", op.Path)
			}
		}
		if op.SecretResponse {
			secretResponses++
			if cmd.Flags().Lookup("show-secret") == nil {
				t.Errorf("%s %s has no --show-secret", op.Method, op.Path)
			}
		}
		for _, parameter := range op.Queries {
			if parameter.WireName != "query" {
				continue
			}
			searchFilters++
			if queryFlagName(parameter) != "search" || cmd.Flags().Lookup("search") == nil || cmd.Flags().Lookup("query") != nil {
				t.Errorf("%s %s must expose API query as --search without a local --query", op.Method, op.Path)
			}
		}
	}
	if mutations != 35 {
		t.Fatalf("state-changing operation count = %d, want 35", mutations)
	}
	if dryRunOperations != 36 {
		t.Fatalf("dry-run operation count = %d, want 36", dryRunOperations)
	}
	if destructive != 6 {
		t.Fatalf("destructive operation count = %d, want 6", destructive)
	}
	if downloads != 1 {
		t.Fatalf("download operation count = %d, want 1", downloads)
	}
	if secretResponses != 8 {
		t.Fatalf("credential-bearing response count = %d, want 8", secretResponses)
	}
	if searchFilters != 10 {
		t.Fatalf("API search-filter count = %d, want 10", searchFilters)
	}
}

func TestPublishedReadOnlyPostsStayQueries(t *testing.T) {
	for _, signature := range []string{
		"POST /v1/archives/test-connection",
		"POST /v1/projects/{projectId}/search-logs",
		"POST /v1/projects/{projectId}/search-logs/default",
		"POST /v1/processors/debug-grok-parser",
		"POST /v1/refills/test-connection",
	} {
		op := findOperationBySignature(t, signature)
		if op.Mutation || newOperationCommand(op).Flags().Lookup("dry-run") != nil {
			t.Errorf("%s must stay read-only", signature)
		}
	}
}

type responseFixture struct {
	Status    int    `json:"status"`
	MediaType string `json:"media_type"`
	Type      string `json:"type"`
	Format    string `json:"format"`
}

func readVMonitorLogCatalog(t *testing.T) map[string]catalogRecord {
	t.Helper()
	raw, err := os.ReadFile("testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			Version     string `json:"version"`
			RetrievedAt string `json:"retrieved_at"`
			SHA256      string `json:"sha256"`
		} `json:"source"`
		PublishedServers []string `json:"published_servers"`
		Authentication   struct {
			Header string `json:"header"`
			Scheme string `json:"scheme"`
		} `json:"authentication"`
		Operations []catalogRecord `json:"operations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing fixture data")
	}
	if fixture.SchemaVersion != 1 || fixture.Source.URL != "https://docs.api.greennode.ai/service-docs/log-api.html" || fixture.Source.Title == "" || fixture.Source.Version == "" || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(fixture.Source.SHA256) || len(fixture.PublishedServers) == 0 || fixture.Authentication.Header != "Authorization" || fixture.Authentication.Scheme != "Bearer" {
		t.Fatal("invalid fixture provenance")
	}
	result := make(map[string]catalogRecord)
	ids := map[string]bool{}
	for _, record := range fixture.Operations {
		key := record.Method + " " + record.Path
		if _, exists := result[key]; exists || record.OperationID == "" || ids[record.OperationID] {
			t.Fatal("duplicate fixture operation")
		}
		result[key] = record
		ids[record.OperationID] = true
	}
	return result
}

func assertOperationContract(t *testing.T, op operation, record catalogRecord) {
	t.Helper()
	wantPaths, err := pathParameters(record.Path)
	if err != nil {
		t.Fatalf("catalog path %q: %v", record.Path, err)
	}
	gotPaths, err := pathParameters(op.Path)
	if err != nil {
		t.Fatalf("operation path %q: %v", op.Path, err)
	}
	if len(gotPaths) != len(wantPaths) {
		t.Errorf("%s %s path parameters = %v, want %v", op.Method, op.Path, gotPaths, wantPaths)
	}
	for index, name := range wantPaths {
		if index >= len(gotPaths) || gotPaths[index] != name {
			t.Errorf("%s %s path parameter %d = %q, want %q", op.Method, op.Path, index, gotPaths[index], name)
		}
	}

	wantQueries := make(map[string]bool)
	for _, parameter := range record.Parameters {
		if parameter.In == "query" {
			wantQueries[parameter.Name] = parameter.Required
		}
	}
	gotQueries := make(map[string]bool)
	for _, parameter := range op.Queries {
		gotQueries[parameter.WireName] = parameter.Required
	}
	if len(gotQueries) != len(wantQueries) {
		t.Errorf("%s %s query parameters = %#v, want %#v", op.Method, op.Path, gotQueries, wantQueries)
	}
	for name, required := range wantQueries {
		if gotRequired, ok := gotQueries[name]; !ok || gotRequired != required {
			t.Errorf("%s %s query parameter %s required = %v/%v, want present/%v", op.Method, op.Path, name, ok, gotRequired, required)
		}
	}

	wantBody := record.RequestBody != nil && record.RequestBody.Required
	if (op.Body != nil) != wantBody {
		t.Errorf("%s %s request body = %t, want %t", op.Method, op.Path, op.Body != nil, wantBody)
	}
}

func findOperationBySignature(t *testing.T, signature string) operation {
	t.Helper()
	for _, op := range allOperations() {
		if op.Method+" "+op.Path == signature {
			return op
		}
	}
	t.Fatalf("operation %s not found", signature)
	return operation{}
}
