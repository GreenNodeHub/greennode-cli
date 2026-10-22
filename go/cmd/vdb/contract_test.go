package vdb

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

type catalogRecord struct {
	OperationID string             `json:"operation_id"`
	Method      string             `json:"method"`
	Path        string             `json:"path"`
	Parameters  []catalogParameter `json:"parameters"`
	Request     *struct {
		Required bool `json:"required"`
		Content  map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"request_body"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

type catalogParameter struct {
	In       string         `json:"in"`
	Name     string         `json:"name"`
	Required bool           `json:"required"`
	Schema   map[string]any `json:"schema"`
}

type schemaCatalog struct {
	Components struct {
		Schemas map[string]struct {
			Required []string `json:"required"`
		} `json:"schemas"`
	} `json:"components"`
}

func TestOperationDescriptorsMatchOfficialCatalog(t *testing.T) {
	catalog := readVDBCatalog(t)
	schemas := readVDBSchemas(t)
	operations := allOperations()
	if got := len(operations); got != 139 {
		t.Fatalf("operation descriptor count = %d, want 139", got)
	}
	if got := len(catalog); got != 139 {
		t.Fatalf("catalog operation count = %d, want 139", got)
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
		assertOperationContract(t, op, record, schemas)
	}
	for signature := range catalog {
		if !seen[signature] {
			t.Errorf("missing descriptor for %s", signature)
		}
	}
}

func TestStateChangingOperationsExposeSafetyFlags(t *testing.T) {
	mutations := 0
	destructive := 0
	secrets := 0
	for _, op := range allOperations() {
		cmd := newOperationCommand(op)
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
		if op.SecretResponse {
			secrets++
			if cmd.Flags().Lookup("show-secret") == nil {
				t.Errorf("%s %s has no --show-secret", op.Method, op.Path)
			}
		}
	}
	if mutations != 66 || destructive != 22 || secrets != 3 {
		t.Fatalf("operation classes = mutations:%d destructive:%d secrets:%d, want 66/22/3", mutations, destructive, secrets)
	}
}

func readVDBCatalog(t *testing.T) map[string]catalogRecord {
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

func readVDBSchemas(t *testing.T) schemaCatalog {
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
	if metadata.SchemaVersion != 1 || metadata.Source.URL != "https://docs.api.greennode.ai/service-docs/vdb-api.html" || metadata.Source.Title != "vDB API" || len(metadata.Source.SHA256) != 64 {
		t.Fatal("invalid public fixture provenance")
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func assertOperationContract(t *testing.T, op operation, record catalogRecord, schemas schemaCatalog) {
	t.Helper()
	if op.OperationID != record.OperationID {
		t.Errorf("%s %s operation ID = %q, want %q", op.Method, op.Path, op.OperationID, record.OperationID)
	}
	wantPaths, err := pathParameters(record.Path)
	if err != nil {
		t.Fatalf("catalog path %q: %v", record.Path, err)
	}
	gotPaths, err := pathParameters(op.Path)
	if err != nil {
		t.Fatalf("operation path %q: %v", op.Path, err)
	}
	if !equalStrings(gotPaths, wantPaths) {
		t.Errorf("%s %s path parameters = %v, want %v", op.Method, op.Path, gotPaths, wantPaths)
	}

	wantQueries := make(map[string]queryParameter)
	wantPortalUserID := false
	wantUserType := false
	for _, parameter := range record.Parameters {
		if parameter.In == "header" {
			switch parameter.Name {
			case "portal-user-id":
				wantPortalUserID = true
				if !parameter.Required || parameter.Schema["type"] != "integer" || parameter.Schema["format"] != "int32" {
					t.Errorf("%s %s portal-user-id = %#v, want required int32", op.Method, op.Path, parameter)
				}
			case "user-type":
				wantUserType = true
			default:
				t.Errorf("%s %s has unsupported header %q", op.Method, op.Path, parameter.Name)
			}
		}
		if parameter.In != "query" {
			continue
		}
		kind := queryString
		switch parameter.Schema["type"] {
		case "integer":
			kind = queryInteger
		case "boolean":
			kind = queryBoolean
		case "object":
			kind = queryObject
		}
		wantQueries[parameter.Name] = queryParameter{WireName: parameter.Name, Required: parameter.Required, Kind: kind}
	}
	if gotPortalUserID := !op.NoPortalUserID; gotPortalUserID != wantPortalUserID {
		t.Errorf("%s %s portal-user-id support = %t, want %t", op.Method, op.Path, gotPortalUserID, wantPortalUserID)
	}
	gotQueries := make(map[string]queryParameter)
	for _, parameter := range op.Queries {
		gotQueries[parameter.WireName] = queryParameter{WireName: parameter.WireName, Required: parameter.Required, Kind: parameter.Kind}
	}
	if len(gotQueries) != len(wantQueries) {
		t.Errorf("%s %s query count = %d, want %d", op.Method, op.Path, len(gotQueries), len(wantQueries))
	}
	for name, want := range wantQueries {
		if got, ok := gotQueries[name]; !ok || got.Required != want.Required || got.Kind != want.Kind {
			t.Errorf("%s %s query %s = %#v/present:%t, want %#v", op.Method, op.Path, name, got, ok, want)
		}
	}
	if op.UserType != wantUserType {
		t.Errorf("%s %s user-type support = %t, want %t", op.Method, op.Path, op.UserType, wantUserType)
	}
	if (op.Body != nil) != (record.Request != nil && record.Request.Required) {
		t.Errorf("%s %s request body = %t, want %t", op.Method, op.Path, op.Body != nil, record.Request != nil && record.Request.Required)
	} else if op.Body != nil {
		wantKind := bodyObject
		wantSchema := ""
		for contentType, content := range record.Request.Content {
			if contentType != "application/json" {
				t.Errorf("%s %s request content type = %q, want application/json", op.Method, op.Path, contentType)
			}
			wantSchema = schemaReferenceName(content.Schema)
			if content.Schema["type"] == "array" {
				wantKind = bodyArray
			}
			break
		}
		if op.Body.Name != wantSchema {
			t.Errorf("%s %s body schema = %q, want %q", op.Method, op.Path, op.Body.Name, wantSchema)
		}
		if op.Body.Kind != wantKind {
			t.Errorf("%s %s body kind = %d, want %d", op.Method, op.Path, op.Body.Kind, wantKind)
		}
		if wantKind == bodyArray && !equalStringSets(op.Body.RequiredItemFields, schemas.Components.Schemas[wantSchema].Required) {
			t.Errorf("%s %s required array item fields = %v, want %v", op.Method, op.Path, op.Body.RequiredItemFields, schemas.Components.Schemas[wantSchema].Required)
		}
		if wantKind == bodyObject && len(schemas.Components.Schemas[wantSchema].Required) > 0 {
			t.Errorf("%s %s has required object fields not represented by the runtime contract: %v", op.Method, op.Path, schemas.Components.Schemas[wantSchema].Required)
		}
	}

	wantSuccesses := map[int]bool{200: true, 202: false, 204: false}
	gotSuccesses := make(map[int]bool)
	for status, response := range record.Responses {
		code, err := strconv.Atoi(status)
		if err == nil && code >= 200 && code < 300 {
			gotSuccesses[code] = len(response.Content) > 0
		}
	}
	if !equalStatusContracts(gotSuccesses, wantSuccesses) {
		t.Errorf("%s %s success responses = %#v, runtime expects %#v", op.Method, op.Path, gotSuccesses, wantSuccesses)
	}
	wantStringResponse := record.Responses["200"].Content["*/*"].Schema["type"] == "string"
	if op.RawResponse != wantStringResponse {
		t.Errorf("%s %s string response = %t, want %t", op.Method, op.Path, op.RawResponse, wantStringResponse)
	}

	if op.Mutation != (op.Method != "GET") {
		t.Errorf("%s %s mutation = %t, want %t", op.Method, op.Path, op.Mutation, op.Method != "GET")
	}
	if op.Destructive != isDestructiveOperation(op) {
		t.Errorf("%s %s destructive = %t, want %t", op.Method, op.Path, op.Destructive, isDestructiveOperation(op))
	}
	if op.SecretResponse != isSecretOperation(op) {
		t.Errorf("%s %s secret response = %t, want %t", op.Method, op.Path, op.SecretResponse, isSecretOperation(op))
	}
}

func schemaReferenceName(schema map[string]any) string {
	if reference, ok := schema["$ref"].(string); ok {
		return strings.TrimPrefix(reference, "#/components/schemas/")
	}
	items, _ := schema["items"].(map[string]any)
	reference, _ := items["$ref"].(string)
	return strings.TrimPrefix(reference, "#/components/schemas/")
}

func equalStringSets(left, right []string) bool {
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

func equalStatusContracts(left, right map[int]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for status, body := range right {
		if left[status] != body {
			return false
		}
	}
	return true
}

func isDestructiveOperation(op operation) bool {
	return strings.HasPrefix(op.Use, "delete") || op.Use == "restore-backup" || op.Use == "detach-replica" || op.Use == "reboot-database-instances" || op.Use == "stop-database-instances" || op.Use == "regenerate-user-authen-credential"
}

func isSecretOperation(op operation) bool {
	return op.OperationID == "createUser" || op.OperationID == "getUserAuthenCredential" || op.OperationID == "regenerateUserAuthenCredential"
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
