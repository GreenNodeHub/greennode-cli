package vstorage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type recordedCall struct {
	method string
	path   string
	query  map[string]string
	body   any
}

type recordingAPI struct {
	calls      []recordedCall
	result     any
	statusCode int
	empty      bool
}

func TestEveryOperationExecutesReadOrOfflineDryRun(t *testing.T) {
	previous := newClient
	t.Cleanup(func() { newClient = previous })

	for _, op := range allOperations() {
		op := op
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			fake := &recordingAPI{}
			clientCalls := 0
			newClient = func(*cobra.Command) (vstorageAPI, error) {
				clientCalls++
				return fake, nil
			}
			flags := validOperationFlags(op)
			if op.Mutation {
				flags["dry-run"] = "true"
				testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
				if clientCalls != 0 || len(fake.calls) != 0 {
					t.Fatalf("dry-run constructed client %d times and made calls %#v", clientCalls, fake.calls)
				}
				return
			}

			testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
			if clientCalls != 1 || len(fake.calls) != 1 || fake.calls[0].method != op.Method {
				t.Fatalf("read client calls = %d, requests = %#v; want one %s request", clientCalls, fake.calls, op.Method)
			}
		})
	}
}

func validOperationFlags(op operation) map[string]string {
	flags := map[string]string{}
	for _, parameter := range op.Paths {
		flags[parameter.Flag] = "value-1"
	}
	for _, parameter := range op.Queries {
		if parameter.Required {
			flags[parameter.Flag] = "value"
		}
	}
	switch op.Body {
	case requiredObjectBody:
		flags["body"] = `{}`
	case requiredArrayBody:
		flags["body"] = `[]`
	}
	return flags
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body})
	statusCode := f.statusCode
	if statusCode == 0 {
		statusCode = 200
	}
	result := any(map[string]any{"ok": true})
	if f.result != nil {
		result = f.result
	}
	return client.HTTPResponse{StatusCode: statusCode, Data: result, Empty: f.empty}, nil
}

func TestHTTP200FailureEnvelopeReturnsErrorMessage(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{
		"success":  false,
		"code":     float64(114),
		"errorMsg": "Could not send order request: Creating order failed",
		"data":     nil,
	}
	op := findOperation(t, "project", "create")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{
		"body": `{"projectName":"grn-cli-poc","projectType":"Gold","quotaInGBytes":30}`,
	})

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("RunE() error = nil, want application-level failure")
	}
	if got, want := err.Error(), "Could not send order request: Creating order failed"; got != want {
		t.Fatalf("RunE() error = %q, want %q", got, want)
	}
	if len(fake.calls) != 1 || fake.calls[0].method != "POST" {
		t.Fatalf("calls = %#v, want one POST", fake.calls)
	}
}

func TestHTTP200FailureEnvelopePropagatesThroughCobraExecute(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{
		"success":  false,
		"code":     float64(114),
		"errorMsg": "Could not send order request: Creating order failed",
	}
	root := &cobra.Command{Use: "grn", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(newVStorageCommand())
	root.SetArgs([]string{
		"vstorage", "project", "create",
		"--body", `{"projectName":"grn-cli-poc","projectType":"Gold","quotaInGBytes":30}`,
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil; the root process would incorrectly exit zero")
	}
	if got, want := err.Error(), "Could not send order request: Creating order failed"; got != want {
		t.Fatalf("Execute() error = %q, want %q", got, want)
	}
}

func TestEmptyHTTP200PropagatesThroughCobraExecute(t *testing.T) {
	fake := installRecordingClient(t)
	fake.statusCode = http.StatusOK
	fake.empty = true
	root := &cobra.Command{Use: "grn", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(newVStorageCommand())
	root.SetArgs([]string{"vstorage", "project", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil; an undocumented empty HTTP 200 would incorrectly exit zero")
	}
	want := "vStorage API returned an empty HTTP 200 response for GET /api/v1/projects; expected JSON"
	if got := err.Error(); got != want {
		t.Fatalf("Execute() error = %q, want %q", got, want)
	}
}

func TestDocumentedEmptyMutationResponsesRemainSuccessful(t *testing.T) {
	tests := []struct {
		name       string
		op         operation
		statusCode int
	}{
		{name: "created by post", op: findOperation(t, "project", "create"), statusCode: http.StatusCreated},
		{name: "created by put", op: findOperation(t, "project", "resize"), statusCode: http.StatusCreated},
		{name: "no content by delete", op: findOperation(t, "project", "delete"), statusCode: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := client.HTTPResponse{StatusCode: tt.statusCode, Data: map[string]any{}, Empty: true}
			if err := responseError(response, tt.op); err != nil {
				t.Fatalf("responseError() = %v, want documented empty response to succeed", err)
			}
		})
	}
}

func TestUndocumentedEmptySuccessStatusesFail(t *testing.T) {
	tests := []struct {
		name       string
		op         operation
		statusCode int
	}{
		{name: "get 200", op: findOperation(t, "project", "list"), statusCode: http.StatusOK},
		{name: "post 200", op: findOperation(t, "project", "create"), statusCode: http.StatusOK},
		{name: "get 204", op: findOperation(t, "project", "list"), statusCode: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := client.HTTPResponse{StatusCode: tt.statusCode, Data: map[string]any{}, Empty: true}
			if err := responseError(response, tt.op); err == nil || !strings.Contains(err.Error(), "expected JSON") {
				t.Fatalf("responseError() = %v, want unexpected-empty-response error", err)
			}
		})
	}
}

func TestFailureEnvelopeWithoutMessageReturnsCode(t *testing.T) {
	op := findOperation(t, "project", "create")
	err := responseError(client.HTTPResponse{StatusCode: 200, Data: map[string]any{"success": false, "code": float64(114)}}, op)
	if err == nil || !strings.Contains(err.Error(), "114") {
		t.Fatalf("responseError() = %v, want fallback containing code 114", err)
	}
}

func TestSuccessfulOrUnwrappedResponsesRemainOutputData(t *testing.T) {
	op := findOperation(t, "project", "list")
	for _, result := range []any{
		map[string]any{"success": true, "data": nil},
		map[string]any{"items": []any{}},
		[]any{},
	} {
		if err := responseError(client.HTTPResponse{StatusCode: 200, Data: result}, op); err != nil {
			t.Errorf("responseError(%#v) = %v, want nil", result, err)
		}
	}
}

func TestReadEscapesStorageNamesAndPreservesQueryNames(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "bucket/object", "get")
	runOperation(t, op, map[string]string{
		"project-id": "pro-123",
		"bucket":     "bucket name",
		"object":     "dir/my file.txt",
		"version-id": "version-7",
	})

	want := recordedCall{
		method: "GET",
		path:   "/api/v1/ceph/projects/pro-123/buckets/bucket%20name/objects/dir%2Fmy%20file.txt/details",
		query:  map[string]string{"versionId": "version-7"},
	}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestQueryVersionIDPreservesOpaqueProviderValue(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "bucket/object", "get")
	version := "3/L4kqtJlcpXroDTDmJ+rmSpXd3dIbrHY+MTRCxf3vjVBH40Nr8X8gdRQBpUMLUo"
	runOperation(t, op, map[string]string{
		"project-id": "project-123",
		"bucket":     "bucket-a",
		"object":     "object.txt",
		"version-id": version,
	})
	if len(fake.calls) != 1 || fake.calls[0].query["versionId"] != version {
		t.Fatalf("call = %#v, want opaque version ID forwarded unchanged", fake.calls)
	}
}

func TestWritePreservesArrayBodyAndJSONNumbers(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "bucket", "update-cors")
	runOperation(t, op, map[string]string{
		"project-id": "pro-123",
		"bucket":     "bucket-a",
		"body":       `[{"maxAge":9007199254740993}]`,
	})

	if len(fake.calls) != 1 || fake.calls[0].method != "PUT" {
		t.Fatalf("calls = %#v, want one PUT", fake.calls)
	}
	body, ok := fake.calls[0].body.([]any)
	if !ok || len(body) != 1 {
		t.Fatalf("body = %#v, want one-element JSON array", fake.calls[0].body)
	}
	item := body[0].(map[string]any)
	if got, ok := item["maxAge"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("maxAge = %#v, want lossless json.Number", item["maxAge"])
	}
}

func TestProjectCreatePoCAddsTheDocumentedIsPocField(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "project", "create")
	runOperation(t, op, map[string]string{
		"body": `{"projectName":"poc-project","projectType":"Gold","quotaInGBytes":30}`,
		"poc":  "true",
	})
	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPost {
		t.Fatalf("calls = %#v, want one project-create POST", fake.calls)
	}
	body := fake.calls[0].body.(map[string]any)
	if body["isPoc"] != true {
		t.Fatalf("body = %#v, want isPoc=true", body)
	}
	if newOperationCommand(findOperation(t, "project", "resize")).Flags().Lookup("poc") != nil {
		t.Fatal("project resize exposes --poc although its published schema has no isPoc field")
	}
}

func TestProjectCreatePoCRejectsContradictoryBody(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "project", "create")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{
		"body": `{"projectName":"poc-project","isPoc":false}`,
		"poc":  "true",
	})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("RunE() error = %v, want PoC/body conflict", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no request after PoC conflict", fake.calls)
	}
}

func TestDeleteWithDocumentedBodyUsesDeleteWithBody(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "bucket/object", "delete")
	runOperation(t, op, map[string]string{
		"project-id": "pro-123",
		"bucket":     "bucket-a",
		"object":     "object.txt",
		"body":       `{"versionId":"version-1"}`,
		"force":      "true",
	})

	if len(fake.calls) != 1 || fake.calls[0].method != "DELETE" {
		t.Fatalf("calls = %#v, want one DELETE", fake.calls)
	}
	wantBody := map[string]any{"versionId": "version-1"}
	if !reflect.DeepEqual(fake.calls[0].body, wantBody) {
		t.Fatalf("body = %#v, want %#v", fake.calls[0].body, wantBody)
	}
}

func TestPostWithQueryUsesExactWireNames(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "container/object", "generate-directory-download-url")
	runOperation(t, op, map[string]string{
		"project-id":   "pro-123",
		"container":    "container-a",
		"directory":    "reports/2026",
		"expired-time": "3600",
		"view-mode":    "inline",
	})

	if len(fake.calls) != 1 || fake.calls[0].method != "POST" {
		t.Fatalf("calls = %#v, want one POST", fake.calls)
	}
	wantQuery := map[string]string{"expiredTime": "3600", "viewMode": "inline"}
	if !reflect.DeepEqual(fake.calls[0].query, wantQuery) {
		t.Fatalf("query = %#v, want %#v", fake.calls[0].query, wantQuery)
	}
}

func TestDryRunIsOfflineAndRedactsCredentialFields(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vstorageAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "container/object", "generate-download-url")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{
			"project-id": "pro-123",
			"container":  "container-a",
			"object":     "report.txt",
			"body":       `{"accessKey":"access-do-not-print","access_key":"snake-access-do-not-print","secretKey":"secret-do-not-print","nested":{"authToken":"also-secret"}}`,
			"dry-run":    "true",
		})
	})

	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "access-do-not-print") || strings.Contains(output, "snake-access-do-not-print") || strings.Contains(output, "secret-do-not-print") || strings.Contains(output, "also-secret") {
		t.Fatalf("dry-run leaked a credential: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "POST") {
		t.Fatalf("dry-run output = %q, want redacted POST preview", output)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	op := findOperation(t, "container", "delete")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{
		"project-id": "pro-123",
		"container":  "container-a",
	})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected confirmation refusal")
	}

	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want none after confirmation refusal", fake.calls)
	}
	if cli.ConfirmationError() == nil {
		t.Fatal("ConfirmationError() = nil, want non-interactive refusal")
	}
}

func TestBodyShapeAndContainerValidation(t *testing.T) {
	op := findOperation(t, "bucket", "update-cors")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "bucket": "bucket-a", "body": `{}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "JSON array") {
		t.Fatalf("RunE error = %v, want JSON array error", err)
	}

	op = findOperation(t, "container", "get")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "container": "bad/name"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "container-name") {
		t.Fatalf("RunE error = %v, want container-name validation error", err)
	}
}

func TestBucketAndLifecycleRuleConstraintsFailBeforeAPI(t *testing.T) {
	fake := installRecordingClient(t)
	tests := []struct {
		name   string
		parent string
		use    string
		flags  map[string]string
		want   string
	}{
		{
			name:   "bucket character",
			parent: "bucket",
			use:    "get",
			flags:  map[string]string{"project-id": "pro-123", "bucket": "bad%bucket"},
			want:   "bucket-name",
		},
		{
			name:   "bucket length",
			parent: "bucket",
			use:    "get",
			flags:  map[string]string{"project-id": "pro-123", "bucket": "ab"},
			want:   "3 to 235",
		},
		{
			name:   "lifecycle rule character",
			parent: "bucket",
			use:    "get-lifecycle-rule",
			flags:  map[string]string{"project-id": "pro-123", "bucket": "bucket-a", "rule-name": "bad.rule"},
			want:   "lifecycle-rule-name",
		},
		{
			name:   "lifecycle rule length",
			parent: "bucket",
			use:    "get-lifecycle-rule",
			flags:  map[string]string{"project-id": "pro-123", "bucket": "bucket-a", "rule-name": "rule"},
			want:   "5 to 50",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := findOperation(t, tt.parent, tt.use)
			cmd := newOperationCommand(op)
			setFlags(t, cmd, tt.flags)
			if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RunE() error = %v, want product-constraint error containing %q", err, tt.want)
			}
		})
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want invalid names rejected before API use", fake.calls)
	}
}

func TestRegionalFamilyMismatchFailsBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vstorageAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "bucket", "list")
	cmd := newOperationCommand(op)
	addRegionalTestFlags(cmd)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "region": "HCM-3"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "HAN or HCM-4") {
		t.Fatalf("RunE error = %v, want Ceph-region error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestEndpointOverrideAllowsExplicitFamilyTesting(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "bucket", "list")
	cmd := newOperationCommand(op)
	addRegionalTestFlags(cmd)
	setFlags(t, cmd, map[string]string{
		"project-id":   "pro-123",
		"region":       "HCM-3",
		"endpoint-url": "https://test.vstorage.vngcloud.vn",
	})
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("calls = %#v, want one explicit-endpoint request", fake.calls)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vstorageAPI, error) { return fake, nil }
	t.Cleanup(func() { newClient = previous })
	return fake
}

func findOperation(t *testing.T, parent, use string) operation {
	t.Helper()
	for _, op := range allOperations() {
		if op.Parent == parent && op.Use == use {
			return op
		}
	}
	t.Fatalf("operation %s %s not found", parent, use)
	return operation{}
}

func runOperation(t *testing.T, op operation, flags map[string]string) {
	t.Helper()
	cmd := newOperationCommand(op)
	setFlags(t, cmd, flags)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func setFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}

func addRegionalTestFlags(cmd *cobra.Command) {
	cmd.Flags().String("region", "", "test region")
	cmd.Flags().String("profile", "", "test profile")
	cmd.Flags().String("endpoint-url", "", "test endpoint")
}

func TestReferencedCephBodiesThroughCobra(t *testing.T) {
	for _, command := range []struct{ parent, use string }{
		{"bucket", "create-lifecycle-rule"}, {"bucket", "update-lifecycle-rule"},
		{"bucket", "create-notification"}, {"bucket", "update-notification"},
		{"bucket/object", "copy"}, {"bucket/object", "move"},
	} {
		op := findOperation(t, command.parent, command.use)
		for _, raw := range []string{"", "{}", "[]", "\"text\"", "true", "null"} {
			for _, dryRun := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/%s/dry-run=%t", op.Parent, op.Use, raw, dryRun), func(t *testing.T) {
					fake := installRecordingClient(t)
					clients := 0
					newClient = func(*cobra.Command) (vstorageAPI, error) { clients++; return fake, nil }
					cmd := newOperationCommand(op)
					cmd.SilenceUsage, cmd.SilenceErrors = true, true
					flags := validOperationFlags(op)
					delete(flags, "body")
					setFlags(t, cmd, flags)
					args := []string{}
					if raw != "" {
						args = append(args, "--body", raw)
					}
					if dryRun {
						args = append(args, "--dry-run")
					}
					cmd.SetArgs(args)
					var err error
					testutil.CaptureStdout(t, func() { err = cmd.Execute() })
					valid := raw == "{}"
					if (err == nil) != valid {
						t.Fatalf("error = %v, valid = %v", err, valid)
					}
					wantCalls := 0
					if valid && !dryRun {
						wantCalls = 1
					}
					if clients != wantCalls || len(fake.calls) != wantCalls {
						t.Fatalf("clients/calls = %d/%d, want %d", clients, len(fake.calls), wantCalls)
					}
				})
			}
		}
	}
}
