package iam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type iamCatalogRecord struct {
	ServiceID  string `json:"service_id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Parameters []struct {
		In       string `json:"in"`
		Name     string `json:"name"`
		Required bool   `json:"required"`
	} `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema struct {
				Type     string   `json:"type"`
				Required []string `json:"required"`
			} `json:"schema"`
			Example json.RawMessage `json:"example"`
		} `json:"content"`
	} `json:"request_body"`
	Responses map[string]struct {
		Content map[string]json.RawMessage `json:"content"`
	} `json:"responses"`
}

func TestIAMMutationDescriptorsMatchOfficialCatalog(t *testing.T) {
	records := readIAMMutationCatalog(t)
	operations := allIAMMutationOperations()
	if len(operations) != 61 {
		t.Fatalf("mutation descriptor count = %d, want 61", len(operations))
	}

	seen := make(map[string]bool, len(operations))
	accounts, policies := 0, 0
	for _, op := range operations {
		signature := op.Method + " " + op.Path
		if seen[signature] {
			t.Errorf("duplicate mutation descriptor: %s", signature)
			continue
		}
		seen[signature] = true
		record, ok := records[signature]
		if !ok {
			t.Errorf("descriptor %s is absent from the official catalog", signature)
			continue
		}
		assertIAMMutationContract(t, op, record)
		if op.BuildClient == nil {
			t.Errorf("%s has no client factory", signature)
		}
		switch record.ServiceID {
		case "iam_accounts":
			accounts++
		case "iam_policies":
			policies++
		default:
			t.Errorf("%s service = %q, want IAM Accounts or Policies", signature, record.ServiceID)
		}
	}
	excluded := excludedIAMKeys()
	for signature := range records {
		if _, gated := excluded[signature]; gated {
			if seen[signature] {
				t.Errorf("%s is listed in excluded.go but still has a descriptor", signature)
			}
			continue
		}
		if !seen[signature] {
			t.Errorf("missing IAM mutation descriptor for %s", signature)
		}
	}
	for signature := range excluded {
		if _, documented := records[signature]; !documented {
			t.Errorf("excluded operation %s is not in the official catalog", signature)
		}
	}
	if accounts != 51 || policies != 10 {
		t.Errorf("account/policy mutation counts = %d/%d, want 51/10", accounts, policies)
	}
}

func TestIAMMutationSafetyFlags(t *testing.T) {
	for _, op := range allIAMMutationOperations() {
		cmd := newIAMMutationCommand(op)
		signature := op.Method + " " + op.Path
		for _, flag := range []string{"dry-run", "force"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s has no --%s", signature, flag)
			}
		}
		if op.Body != nil {
			flags := []string{"body-file"}
			if !op.SecretInput {
				flags = append(flags, "body")
			}
			for _, flag := range flags {
				if cmd.Flags().Lookup(flag) == nil {
					t.Errorf("%s has no --%s", signature, flag)
				}
			}
		}
		if op.SecretResponse && cmd.Flags().Lookup("show-secret") == nil {
			t.Errorf("%s has no --show-secret", signature)
		}
	}
}

func TestIAMMutationCommandsAreRegistered(t *testing.T) {
	operations := allIAMMutationOperations()
	if len(registeredIAMMutationCommands) != len(operations) {
		t.Fatalf("registered IAM mutation commands = %d, want %d", len(registeredIAMMutationCommands), len(operations))
	}
	for _, op := range operations {
		cmd, ok := registeredIAMMutationCommands[op.Key]
		if !ok {
			t.Errorf("%s is not registered in the IAM command tree", op.Key)
			continue
		}
		if cmd.Parent() == nil || cmd.RunE == nil {
			t.Errorf("%s registration is not a runnable IAM child command", op.Key)
		}
	}
}

func TestIAMMutationDryRunIsOfflineAndRedactsSecrets(t *testing.T) {
	clientCalls := 0
	previous := accountsClientFactory
	accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		clientCalls++
		return nil, nil
	}
	t.Cleanup(func() { accountsClientFactory = previous })

	bodyFile := filepath.Join(t.TempDir(), "user.json")
	if err := os.WriteFile(bodyFile, []byte(`{"username":"robot","password":"do-not-print"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newIAMMutationCommand(findIAMMutation(t, "iam-user-create"))
	setIAMMutationFlags(t, cmd, map[string]string{
		"body-file": bodyFile,
		"dry-run":   "true",
	})
	output := testutil.CaptureStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "do-not-print") || !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("dry-run output = %q, want redacted password", output)
	}
}

func TestEveryIAMMutationDryRunIsOffline(t *testing.T) {
	clientCalls := 0
	previousAccounts := accountsClientFactory
	previousPolicies := policiesClientFactory
	accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		clientCalls++
		return nil, fmt.Errorf("accounts client must not be created during dry run")
	}
	policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		clientCalls++
		return nil, fmt.Errorf("policies client must not be created during dry run")
	}
	t.Cleanup(func() {
		accountsClientFactory = previousAccounts
		policiesClientFactory = previousPolicies
	})

	for _, op := range allIAMMutationOperations() {
		op := op
		t.Run(op.Key, func(t *testing.T) {
			cmd := newIAMMutationCommand(op)
			flags := validIAMMutationFlags(t, op)
			flags["dry-run"] = "true"
			setIAMMutationFlags(t, cmd, flags)
			testutil.CaptureStdout(t, func() {
				if err := cmd.RunE(cmd, nil); err != nil {
					t.Fatalf("RunE() error = %v", err)
				}
			})
		})
	}
	if clientCalls != 0 {
		t.Fatalf("IAM dry runs created %d clients", clientCalls)
	}
}

func validIAMMutationFlags(t *testing.T, op iamMutationOperation) map[string]string {
	t.Helper()
	flags := map[string]string{}
	for _, parameter := range op.Paths {
		flags[parameter.Flag] = "value-1"
	}
	if op.Body == nil {
		return flags
	}
	var body any
	if op.Body.Shape == iamMutationBodyArray {
		body = []any{}
	} else {
		object := map[string]any{}
		for _, field := range op.Body.RequiredFields {
			object[field] = "value"
		}
		body = object
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if op.SecretInput {
		bodyFile := filepath.Join(t.TempDir(), "body.json")
		if err := os.WriteFile(bodyFile, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		flags["body-file"] = bodyFile
	} else {
		flags["body"] = string(encoded)
	}
	return flags
}

func TestPolicyComposeAcceptsOnlyItsDocumentedStatementArray(t *testing.T) {
	statementArray := `[{"effect":"allow","actions":["iam:ListPolicies"],"resources":["*"]}]`
	compose := newIAMMutationCommand(findIAMMutation(t, "policy-compose"))
	setIAMMutationFlags(t, compose, map[string]string{"body": statementArray, "dry-run": "true"})

	body, err := parseIAMMutationBody(compose, findIAMMutation(t, "policy-compose"))
	if err != nil {
		t.Fatalf("parseIAMMutationBody() error = %v", err)
	}
	if _, ok := body.([]any); !ok {
		t.Fatalf("compose body type = %T, want []any", body)
	}
	output := testutil.CaptureStdout(t, func() {
		if err := compose.RunE(compose, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if !strings.Contains(output, "iam:ListPolicies") {
		t.Fatalf("dry-run output = %q, want statement array", output)
	}

	create := newIAMMutationCommand(findIAMMutation(t, "policy-create"))
	setIAMMutationFlags(t, create, map[string]string{"body": statementArray, "dry-run": "true"})
	if err := create.RunE(create, nil); err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("policy create RunE() error = %v, want JSON object error", err)
	}
}

func TestIAMMutationSecretBodyFileMustBePrivate(t *testing.T) {
	bodyFile := filepath.Join(t.TempDir(), "user.json")
	if err := os.WriteFile(bodyFile, []byte(`{"email":"fixture-user@example.invalid"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newIAMMutationCommand(findIAMMutation(t, "iam-user-create"))
	setIAMMutationFlags(t, cmd, map[string]string{"body-file": bodyFile, "dry-run": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "group or others") {
		t.Fatalf("RunE() error = %v, want private body-file error", err)
	}
}

func TestIAMMutationRejectsEndpointOverrideBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := accountsClientFactory
	accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) {
		clientCalls++
		return nil, nil
	}
	t.Cleanup(func() { accountsClientFactory = previous })

	cmd := newIAMMutationCommand(findIAMMutation(t, "s3-key-delete"))
	cmd.Flags().String("endpoint-url", "", "")
	setIAMMutationFlags(t, cmd, map[string]string{"s3-key-id": "key-123", "endpoint-url": "https://override.greennode.ai"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "--endpoint-url is not supported") {
		t.Fatalf("RunE() error = %v, want endpoint override rejection", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestIAMMutationSecretResponseRedactedUnlessRequested(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/s3-keys" {
			t.Errorf("request = %s %s, want POST /v1/s3-keys", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"accessKey":"access-key","secretKey":"secret-key"}`))
	}))
	t.Cleanup(server.Close)
	installIAMMutationClient(t, "accounts", server.URL)

	op := findIAMMutation(t, "s3-key-create")
	run := func(showSecret bool) string {
		cmd := newIAMMutationCommand(op)
		var warnings bytes.Buffer
		cmd.SetErr(&warnings)
		flags := map[string]string{"body": `{"regionId":"region-1","projectId":"project-1"}`, "force": "true"}
		if showSecret {
			flags["show-secret"] = "true"
		}
		setIAMMutationFlags(t, cmd, flags)
		output := testutil.CaptureStdout(t, func() {
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatalf("RunE() error = %v", err)
			}
		})
		if strings.Contains(strings.ToLower(warnings.String()), "re-run") {
			t.Fatal("successful mutation must not advise another write")
		}
		return output
	}

	redacted := run(false)
	if strings.Contains(redacted, "access-key") || strings.Contains(redacted, "secret-key") || !strings.Contains(redacted, client.RedactedValue) {
		t.Fatalf("redacted output = %q", redacted)
	}
	if output := run(true); !strings.Contains(output, "secret-key") {
		t.Fatalf("show-secret output = %q, want secret", output)
	}
}

func TestIAMMutationRefusesDirectCurrentIdentityTarget(t *testing.T) {
	requests := 0
	installIAMTestClients(t, func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != http.MethodGet || request.URL.Path != "/v1/auth/userinfo" {
			t.Errorf("identity request = %s %s, want GET /v1/auth/userinfo", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"userType":"iam-user","userId":"user-123"}`))
	}, func(http.ResponseWriter, *http.Request) {
		t.Fatal("policy client must not be used")
	})

	cmd := newIAMMutationCommand(findIAMMutation(t, "iam-user-delete"))
	setIAMMutationFlags(t, cmd, map[string]string{"iam-user-id": "user-123", "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "refusing to delete") {
		t.Fatalf("RunE() error = %v, want current identity refusal", err)
	}
	if requests != 1 {
		t.Fatalf("identity requests = %d, want 1", requests)
	}
}

func TestIAMMutationResponseContractRejectsUnexpectedStatusAndBody(t *testing.T) {
	op := findIAMMutation(t, "service-account-reset-secret")
	if err := iamMutationResponseError(op, client.HTTPResponse{StatusCode: http.StatusCreated, Data: map[string]any{}}); err == nil || !strings.Contains(err.Error(), "expected HTTP 200") {
		t.Fatalf("iamMutationResponseError() = %v, want status error", err)
	}
	if err := iamMutationResponseError(op, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err == nil || !strings.Contains(err.Error(), "expected JSON") {
		t.Fatalf("iamMutationResponseError() = %v, want response body error", err)
	}

	op = findIAMMutation(t, "policy-update")
	if err := iamMutationResponseError(op, client.HTTPResponse{StatusCode: http.StatusNoContent, Empty: true}); err != nil {
		t.Fatalf("iamMutationResponseError() = %v, want documented empty response accepted", err)
	}

	op = findIAMMutation(t, "service-account-create")
	if err := iamMutationResponseError(op, client.HTTPResponse{StatusCode: http.StatusCreated, Data: map[string]any{"id": "sa-1", "clientSecret": "must-not-print"}}); err != nil {
		t.Fatalf("iamMutationResponseError() = %v, want live provider payload tolerated", err)
	}
}

func TestServiceAccountCreateSuppressesUndocumentedProviderPayload(t *testing.T) {
	installIAMTestClients(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/service-accounts" {
			t.Errorf("request = %s %s, want POST /v1/service-accounts", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"id":"sa-1","clientSecret":"must-not-print"}`))
	}, func(http.ResponseWriter, *http.Request) {
		t.Fatal("policies client must not be used")
	})

	cmd := newIAMMutationCommand(findIAMMutation(t, "service-account-create"))
	setIAMMutationFlags(t, cmd, map[string]string{"body": `{"name":"disposable"}`, "force": "true"})
	var output, errors bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&errors)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if output.Len() != 0 || strings.Contains(errors.String(), "must-not-print") {
		t.Fatalf("provider payload leaked to output: stdout=%q stderr=%q", output.String(), errors.String())
	}
	if !strings.Contains(errors.String(), "undocumented success payload") {
		t.Fatalf("stderr = %q, want undocumented-payload warning", errors.String())
	}
}

func TestUnexpectedBodyToleranceNeverSuppressesDocumentedResponses(t *testing.T) {
	response := client.HTTPResponse{StatusCode: http.StatusCreated, Data: map[string]any{"id": "resource-1"}}
	if !iamToleratesUnexpectedBody(iamMutationOperation{AllowUnexpectedBody: true}, response) {
		t.Fatal("unexpected empty-contract body was not tolerated")
	}
	if iamToleratesUnexpectedBody(iamMutationOperation{AllowUnexpectedBody: true, ResponseBody: true}, response) {
		t.Fatal("documented response body would be suppressed")
	}
	if iamToleratesUnexpectedBody(iamMutationOperation{AllowUnexpectedBody: true}, client.HTTPResponse{Empty: true}) {
		t.Fatal("empty response was classified as an unexpected body")
	}
}

func readIAMMutationCatalog(t *testing.T) map[string]iamCatalogRecord {
	t.Helper()
	records := map[string]iamCatalogRecord{}
	for _, record := range readIAMFixtures(t) {
		if isMissingIAMWrite(record) {
			records[record.Method+" "+record.Path] = record
		}
	}
	return records
}
func readIAMFixtures(t *testing.T) []iamCatalogRecord {
	t.Helper()
	var records []iamCatalogRecord
	for _, id := range []string{"iam_accounts", "iam_policies"} {
		data, err := os.ReadFile(filepath.Join("testdata", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var fixture struct {
			SchemaVersion int `json:"schema_version"`
			Source        struct{ ID, URL, SHA256 string }
			Operations    []iamCatalogRecord `json:"operations"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatal(err)
		}
		if fixture.SchemaVersion != 1 || fixture.Source.ID != id || len(fixture.Source.SHA256) != 64 || fixture.Source.URL == "" {
			t.Fatal("invalid fixture provenance")
		}
		records = append(records, fixture.Operations...)
	}
	return records
}

func isMissingIAMWrite(record iamCatalogRecord) bool {
	if record.Method != http.MethodPost && record.Method != http.MethodPatch && record.Method != http.MethodPut && record.Method != http.MethodDelete {
		return false
	}
	signature := record.Method + " " + record.Path
	switch record.ServiceID {
	case "iam_accounts":
		return signature != "POST /v2/auth/token"
	case "iam_policies":
		return !map[string]bool{
			"POST /v1/groups/{groupId}/iam-users/{userId}":           true,
			"DELETE /v1/groups/{groupId}/iam-users/{userId}":         true,
			"POST /v1/policies/{policyId}/groups/{groupId}":          true,
			"DELETE /v1/policies/{policyId}/groups/{groupId}":        true,
			"POST /v1/policies/{policyId}/iam-users/{userId}":        true,
			"DELETE /v1/policies/{policyId}/iam-users/{userId}":      true,
			"POST /v1/policies/{policyId}/service-accounts/{saId}":   true,
			"DELETE /v1/policies/{policyId}/service-accounts/{saId}": true,
		}[signature]
	default:
		return false
	}
}

func assertIAMMutationContract(t *testing.T, op iamMutationOperation, record iamCatalogRecord) {
	t.Helper()
	signature := op.Method + " " + op.Path
	wantPaths := make([]string, 0)
	for _, parameter := range record.Parameters {
		if parameter.In == "path" {
			wantPaths = append(wantPaths, parameter.Name)
		}
	}
	gotPaths := make([]string, 0, len(op.Paths))
	for _, parameter := range op.Paths {
		gotPaths = append(gotPaths, parameter.Placeholder)
	}
	sort.Strings(wantPaths)
	sort.Strings(gotPaths)
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("%s path placeholders = %v, want %v", signature, gotPaths, wantPaths)
	}

	if (op.Body == nil) != (record.RequestBody == nil) {
		t.Errorf("%s body contract mismatch", signature)
	} else if op.Body != nil {
		content, ok := record.RequestBody.Content["application/json"]
		if !ok {
			t.Errorf("%s has no application/json request body", signature)
			return
		}
		wantRequired := append([]string(nil), content.Schema.Required...)
		gotRequired := append([]string(nil), op.Body.RequiredFields...)
		sort.Strings(wantRequired)
		sort.Strings(gotRequired)
		if !reflect.DeepEqual(gotRequired, wantRequired) {
			t.Errorf("%s required body fields = %v, want %v", signature, gotRequired, wantRequired)
		}
		if gotShape, wantShape := op.Body.Shape, iamMutationBodyShapeFromCatalog(t, content); gotShape != wantShape {
			t.Errorf("%s body shape = %s, want %s", signature, iamMutationBodyShapeLabel(gotShape), iamMutationBodyShapeLabel(wantShape))
		}
	}

	statuses := make(map[int]bool)
	for status, response := range record.Responses {
		var code int
		if _, err := fmt.Sscanf(status, "%d", &code); err == nil && code >= 200 && code < 300 {
			statuses[code] = len(response.Content) > 0
		}
	}
	if len(statuses) != 1 {
		t.Errorf("%s documented success responses = %#v, want one", signature, statuses)
		return
	}
	if responseBody, ok := statuses[op.Status]; !ok || responseBody != op.ResponseBody {
		t.Errorf("%s response = HTTP %d body=%t, want %#v", signature, op.Status, op.ResponseBody, statuses)
	}
}

func iamMutationBodyShapeFromCatalog(t *testing.T, content struct {
	Schema struct {
		Type     string   `json:"type"`
		Required []string `json:"required"`
	} `json:"schema"`
	Example json.RawMessage `json:"example"`
}) iamMutationBodyShape {
	t.Helper()
	switch content.Schema.Type {
	case "", "object":
	case "array":
		return iamMutationBodyArray
	default:
		t.Fatalf("unsupported IAM request body schema type %q", content.Schema.Type)
	}
	if len(content.Example) != 0 {
		var example any
		if err := json.Unmarshal(content.Example, &example); err != nil {
			t.Fatalf("decode IAM request example: %v", err)
		}
		if _, ok := example.([]any); ok {
			return iamMutationBodyArray
		}
	}
	return iamMutationBodyObject
}

func findIAMMutation(t *testing.T, key string) iamMutationOperation {
	t.Helper()
	for _, op := range allIAMMutationOperations() {
		if op.Key == key {
			return op
		}
	}
	t.Fatalf("IAM mutation %q not found", key)
	return iamMutationOperation{}
}

func setIAMMutationFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}

func installIAMMutationClient(t *testing.T, service, endpoint string) {
	t.Helper()
	newClient := func() *client.GreennodeClient {
		tokens := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "http://127.0.0.1:1/fixture-token")
		tokens.SetToken("offline-token", time.Now().Add(time.Hour))
		return client.NewGreennodeClient(endpoint, tokens, 0, time.Second, true, false)
	}
	if service == "accounts" {
		previous := accountsClientFactory
		accountsClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) { return newClient(), nil }
		t.Cleanup(func() { accountsClientFactory = previous })
		return
	}
	previous := policiesClientFactory
	policiesClientFactory = func(*cobra.Command) (*client.GreennodeClient, error) { return newClient(), nil }
	t.Cleanup(func() { policiesClientFactory = previous })
}
