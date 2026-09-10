package vserver

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type ancillaryCatalogRecord struct {
	ServiceID   string                      `json:"service_id"`
	OperationID string                      `json:"operation_id"`
	Method      string                      `json:"method"`
	Path        string                      `json:"path"`
	Parameters  []ancillaryCatalogParameter `json:"parameters"`
	Request     *ancillaryCatalogBody       `json:"request_body"`
	Responses   map[string]struct {
		Content map[string]json.RawMessage `json:"content"`
	} `json:"responses"`
}

type ancillaryCatalogParameter struct {
	In       string         `json:"in"`
	Name     string         `json:"name"`
	Required bool           `json:"required"`
	Schema   map[string]any `json:"schema"`
}

type ancillaryCatalogBody struct {
	Ref      string `json:"$ref"`
	Required bool   `json:"required"`
	Content  map[string]struct {
		Schema map[string]any `json:"schema"`
	} `json:"content"`
}

type ancillarySchemaCatalog struct {
	Components struct {
		RequestBodies map[string]ancillaryCatalogBody `json:"requestBodies"`
		Schemas       map[string]struct {
			Required []string `json:"required"`
		} `json:"schemas"`
	} `json:"components"`
}

func TestAncillaryOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	catalog := readVServerCatalog(t)
	schemas := readVServerSchemas(t)
	operations := ancillaryOperations()
	if got := len(operations); got != 68 {
		t.Fatalf("ancillary operation descriptor count = %d, want 68", got)
	}

	seen := make(map[string]bool, len(operations))
	for _, op := range operations {
		signature := op.Method + " " + op.Path
		if seen[signature] {
			t.Fatalf("duplicate ancillary operation descriptor: %s", signature)
		}
		seen[signature] = true
		record, ok := catalog[signature]
		if !ok {
			t.Errorf("descriptor is absent from official vServer catalog: %s", signature)
			continue
		}
		assertAncillaryContract(t, op, record, schemas)
	}
}

func TestAncillaryOperationsAreRegisteredAndExposeSafetyFlags(t *testing.T) {
	routes := make(map[string]bool)
	collectLeafRoutes(VServerCmd, nil, routes)
	mutations := 0
	destructive := 0
	for _, op := range ancillaryOperations() {
		route := strings.Join(append(append([]string{}, op.Parents...), op.Use), " ")
		if !routes[route] {
			t.Errorf("ancillary route %q is not registered", route)
		}
		cmd := newAncillaryOperationCommand(op)
		if op.Mutation {
			mutations++
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
	}
	if mutations != 29 || destructive != 11 {
		t.Fatalf("ancillary operation classes = mutations:%d destructive:%d, want 29/11", mutations, destructive)
	}
}

func readVServerCatalog(t *testing.T) map[string]ancillaryCatalogRecord {
	t.Helper()
	var fixture struct{ Operations []ancillaryCatalogRecord }
	readVServerFixture(t, &fixture)
	if len(fixture.Operations) != 209 {
		t.Fatalf("fixture operation count=%d, want209", len(fixture.Operations))
	}
	records := make(map[string]ancillaryCatalogRecord)
	for _, op := range fixture.Operations {
		key := op.Method + " " + op.Path
		if _, ok := records[key]; ok {
			t.Fatalf("duplicate fixture %s", key)
		}
		records[key] = op
	}
	return records
}
func readVServerSchemas(t *testing.T) ancillarySchemaCatalog {
	t.Helper()
	var schemas ancillarySchemaCatalog
	readVServerFixture(t, &schemas)
	return schemas
}
func readVServerFixture(t *testing.T, target any) {
	t.Helper()
	data, err := os.ReadFile("testdata/contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct{ URL, SHA256 string }
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.SchemaVersion != 1 || meta.Source.URL != "https://docs.api.greennode.ai/service-docs/vserver.html" || len(meta.Source.SHA256) != 64 {
		t.Fatal("invalid official fixture provenance")
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func assertAncillaryContract(t *testing.T, op ancillaryOperation, record ancillaryCatalogRecord, schemas ancillarySchemaCatalog) {
	t.Helper()
	if op.OperationID != record.OperationID {
		t.Errorf("%s %s operation ID = %q, want %q", op.Method, op.Path, op.OperationID, record.OperationID)
	}
	wantPathParameters, err := ancillaryPathParameters(op.Path, nil)
	if err != nil {
		t.Errorf("%s %s expected path parameters: %v", op.Method, op.Path, err)
	}
	wantPaths := make(map[string]ancillaryPathKind)
	for _, parameter := range wantPathParameters {
		wantPaths[parameter.Name] = parameter.Kind
	}
	for _, parameter := range record.Parameters {
		if parameter.In != "path" {
			continue
		}
		kind := ancillaryPathString
		switch parameter.Schema["type"] {
		case "string":
		case "integer":
			if parameter.Schema["format"] != "int32" {
				t.Errorf("%s %s path parameter %s has unsupported integer format %v", op.Method, op.Path, parameter.Name, parameter.Schema["format"])
			}
			kind = ancillaryPathInt32
		default:
			t.Errorf("%s %s path parameter %s has unsupported type %v", op.Method, op.Path, parameter.Name, parameter.Schema["type"])
		}
		wantPaths[parameter.Name] = kind
	}
	gotPathParameters, err := ancillaryPathParameters(op.Path, op.PathKinds)
	if err != nil {
		t.Errorf("%s %s path parameters: %v", op.Method, op.Path, err)
	}
	gotPaths := make(map[string]ancillaryPathKind)
	for _, parameter := range gotPathParameters {
		gotPaths[parameter.Name] = parameter.Kind
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("%s %s path parameter kinds = %#v, want %#v", op.Method, op.Path, gotPaths, wantPaths)
	}
	wantQueries := make(map[string]ancillaryQueryParameter)
	for _, parameter := range record.Parameters {
		if parameter.In != "query" {
			continue
		}
		kind := ancillaryQueryString
		switch parameter.Schema["type"] {
		case "integer":
			kind = ancillaryQueryInteger
		case "boolean":
			kind = ancillaryQueryBoolean
		}
		wantQueries[parameter.Name] = ancillaryQueryParameter{Name: parameter.Name, Required: parameter.Required, Kind: kind}
	}
	gotQueries := make(map[string]ancillaryQueryParameter)
	for _, parameter := range op.Queries {
		gotQueries[parameter.Name] = ancillaryQueryParameter{Name: parameter.Name, Required: parameter.Required, Kind: parameter.Kind}
	}
	if len(gotQueries) != len(wantQueries) {
		t.Errorf("%s %s query count = %d, want %d", op.Method, op.Path, len(gotQueries), len(wantQueries))
	}
	for name, want := range wantQueries {
		if got, ok := gotQueries[name]; !ok || got.Required != want.Required || got.Kind != want.Kind {
			t.Errorf("%s %s query %s = %#v/present:%t, want %#v", op.Method, op.Path, name, got, ok, want)
		}
	}

	wantBodyName, wantRequiredFields, wantBody := ancillaryBodyContractFromCatalog(record.Request, schemas)
	if (op.Body != nil) != wantBody {
		t.Errorf("%s %s request body = %t, want %t", op.Method, op.Path, op.Body != nil, wantBody)
	} else if op.Body != nil {
		if op.Body.Name != wantBodyName {
			t.Errorf("%s %s body schema = %q, want %q", op.Method, op.Path, op.Body.Name, wantBodyName)
		}
		if !sameStringSet(op.Body.RequiredFields, wantRequiredFields) {
			t.Errorf("%s %s required body fields = %v, want %v", op.Method, op.Path, op.Body.RequiredFields, wantRequiredFields)
		}
	}

	wantStatus, wantResponseBody := ancillarySuccessContract(t, record)
	if op.SuccessStatus != wantStatus || op.ResponseBody != wantResponseBody {
		t.Errorf("%s %s success contract = %d/body:%t, want %d/body:%t", op.Method, op.Path, op.SuccessStatus, op.ResponseBody, wantStatus, wantResponseBody)
	}
	if op.Mutation != (op.Method != "GET") {
		t.Errorf("%s %s mutation = %t, want %t", op.Method, op.Path, op.Mutation, op.Method != "GET")
	}
	if op.Destructive != (op.Method == "DELETE") {
		t.Errorf("%s %s destructive = %t, want %t", op.Method, op.Path, op.Destructive, op.Method == "DELETE")
	}
}

func ancillaryBodyContractFromCatalog(body *ancillaryCatalogBody, schemas ancillarySchemaCatalog) (string, []string, bool) {
	if body == nil {
		return "", nil, false
	}
	if body.Ref != "" {
		name := strings.TrimPrefix(body.Ref, "#/components/requestBodies/")
		resolved := schemas.Components.RequestBodies[name]
		schema := resolved.Content["application/json"].Schema
		return ancillarySchemaName(schema), ancillaryRequiredFields(schema, schemas), resolved.Required
	}
	schema := body.Content["application/json"].Schema
	return ancillarySchemaName(schema), ancillaryRequiredFields(schema, schemas), body.Required
}

func ancillarySchemaName(schema map[string]any) string {
	if reference, ok := schema["$ref"].(string); ok {
		return strings.TrimPrefix(reference, "#/components/schemas/")
	}
	if title, ok := schema["title"].(string); ok {
		return title
	}
	return "object"
}

func ancillaryRequiredFields(schema map[string]any, schemas ancillarySchemaCatalog) []string {
	if reference, ok := schema["$ref"].(string); ok {
		name := strings.TrimPrefix(reference, "#/components/schemas/")
		return schemas.Components.Schemas[name].Required
	}
	values, _ := schema["required"].([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if field, ok := value.(string); ok {
			result = append(result, field)
		}
	}
	return result
}

func ancillarySuccessContract(t *testing.T, record ancillaryCatalogRecord) (int, bool) {
	t.Helper()
	status := 0
	body := false
	for raw, response := range record.Responses {
		code, err := strconv.Atoi(raw)
		if err != nil || code < 200 || code >= 300 {
			continue
		}
		if status != 0 {
			t.Fatalf("%s %s has multiple success statuses", record.Method, record.Path)
		}
		status = code
		body = code != 202 && code != 204 && len(response.Content) > 0
	}
	return status, body
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	want := make(map[string]bool, len(right))
	for _, value := range right {
		want[value] = true
	}
	for _, value := range left {
		if !want[value] {
			return false
		}
	}
	return true
}
