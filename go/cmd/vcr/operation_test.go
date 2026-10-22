package vcr

import (
	"encoding/json"
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
	method    string
	path      string
	query     map[string]string
	body      any
	noRetry   bool
	sensitive bool
	raw       bool
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
			fake := &recordingAPI{statusCode: op.Status}
			clientCalls := 0
			newClient = func(*cobra.Command) (vcrAPI, error) {
				clientCalls++
				return fake, nil
			}
			flags := validOperationFlags(t, op)
			if op.Mutation {
				flags["dry-run"] = "true"
				testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
				if clientCalls != 0 || len(fake.calls) != 0 {
					t.Fatalf("dry-run constructed client %d times and made calls %#v", clientCalls, fake.calls)
				}
				return
			}

			testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
			if clientCalls != 1 || len(fake.calls) != 1 || fake.calls[0].method != op.Method || fake.calls[0].noRetry {
				t.Fatalf("read client calls = %d, requests = %#v; want one retryable %s request", clientCalls, fake.calls, op.Method)
			}
		})
	}
}

func validOperationFlags(t *testing.T, op operation) map[string]string {
	t.Helper()
	flags := map[string]string{}
	for _, parameter := range op.Paths {
		flags[parameter.Flag] = "value-1"
	}
	for _, parameter := range op.Queries {
		if !parameter.Required {
			continue
		}
		if parameter.Kind == queryInteger {
			flags[parameter.Flag] = "1"
		} else {
			flags[parameter.Flag] = "value"
		}
	}
	if op.Body != nil {
		body := map[string]any{}
		for _, field := range op.Body.RequiredFields {
			body[field] = "value"
		}
		bindings, _ := op.Extra.([]bodyBinding)
		for _, binding := range bindings {
			body[binding.Field] = flags[binding.Flag]
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		flags["body"] = string(encoded)
	}
	return flags
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, false, false)
}

func (f *recordingAPI) RequestWithStatusNoRetry(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, false)
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitive(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, true)
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitiveRaw(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	response, err := f.record(method, path, query, body, true, true)
	f.calls[len(f.calls)-1].raw = true
	return response, err
}

func (f *recordingAPI) record(method, path string, query map[string]string, body any, noRetry, sensitive bool) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body, noRetry: noRetry, sensitive: sensitive})
	status := f.statusCode
	if status == 0 {
		status = http.StatusOK
	}
	result := f.result
	if result == nil {
		result = map[string]any{"listData": []any{}}
	}
	return client.HTTPResponse{StatusCode: status, Data: result, Empty: f.empty}, nil
}

func TestReadBuildsExactPathAndQuery(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "artifact", "list")
	runOperation(t, op, map[string]string{
		"repository-id": "repo-123",
		"image-name":    "api",
		"name":          "sha256",
		"page":          "2",
		"size":          "50",
	})

	want := recordedCall{
		method: http.MethodGet,
		path:   "/v1/repository/repo-123/images/artifacts",
		query:  map[string]string{"imageName": "api", "name": "sha256", "page": "2", "size": "50"},
	}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestValidationRunsBeforeClientConstruction(t *testing.T) {
	fake := installRecordingClient(t)

	op := findOperation(t, "image", "list")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"repository-id": "bad/id", "page": "1"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "repository-id") {
		t.Fatalf("RunE() error = %v, want repository ID validation error", err)
	}

	op = findOperation(t, "repository", "list")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"page": "one"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "32-bit integer") {
		t.Fatalf("RunE() error = %v, want integer validation error", err)
	}

	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"page": "0"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "at least 1") {
		t.Fatalf("RunE() error = %v, want one-based page validation error", err)
	}

	for _, value := range []string{"0", "-1"} {
		cmd = newOperationCommand(op)
		setFlags(t, cmd, map[string]string{"size": value})
		if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "at least 1") {
			t.Fatalf("RunE() error = %v, want positive size validation for %q", err, value)
		}
	}

	op = findOperation(t, "artifact", "list")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"repository-id": "repo-123", "image-name": ""})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "image-name") {
		t.Fatalf("RunE() error = %v, want required image name error", err)
	}

	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no API calls after validation failures", fake.calls)
	}
}

func TestWritePreservesNumbersAndBindsBodyTarget(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "repository", "update-quota")
	runOperation(t, op, map[string]string{
		"repository-id": "repo-123",
		"body":          `{"quotaLimit":9007199254740993,"repoId":"repo-123"}`,
	})

	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPut || !fake.calls[0].noRetry {
		t.Fatalf("calls = %#v, want one no-retry PUT", fake.calls)
	}
	body := fake.calls[0].body.(map[string]any)
	if got, ok := body["quotaLimit"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("quotaLimit = %#v, want lossless json.Number", body["quotaLimit"])
	}

	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{
		"repository-id": "repo-123",
		"body":          `{"quotaLimit":5,"repoId":"repo-other"}`,
	})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --repository-id") {
		t.Fatalf("RunE() error = %v, want mismatched body target error", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("calls = %#v, want no second API call", fake.calls)
	}
}

func TestDryRunIsOfflineAndRedactsSecretShapedBodyFields(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vcrAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "user", "create")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{
			"body":    `{"name":"robot","permissionRequestList":[],"bootstrapSecret":"do-not-print"}`,
			"dry-run": "true",
		})
	})

	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "do-not-print") || !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("dry-run output = %q, want redacted secret", output)
	}
}

func TestRefreshSecretIsDestructiveNoRetryAndRedactedByDefault(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = "rotated-secret"
	op := findOperation(t, "user", "refresh-secret")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"user-id": "user-123", "force": "true"})
	})

	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodGet || !fake.calls[0].noRetry || !fake.calls[0].sensitive || !fake.calls[0].raw {
		t.Fatalf("calls = %#v, want one sensitive no-retry raw GET", fake.calls)
	}
	if strings.Contains(output, "rotated-secret") || !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("output = %q, want redacted secret", output)
	}

	fake.calls = nil
	output = testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"user-id": "user-123", "force": "true", "show-secret": "true"})
	})
	if !strings.Contains(output, "rotated-secret") {
		t.Fatalf("output = %q, want explicitly requested secret", output)
	}
}

func TestCreateUserObjectSecretIsRedactedByDefault(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{"id": "user-123", "secretKey": "created-secret"}
	op := findOperation(t, "user", "create")
	body := `{"name":"robot","permissionRequestList":[]}`

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"body": body})
	})
	if strings.Contains(output, "created-secret") || !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("output = %q, want redacted object secret", output)
	}
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || !fake.calls[0].sensitive || fake.calls[0].raw {
		t.Fatalf("calls = %#v, want one sensitive no-retry create", fake.calls)
	}

	output = testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"body": body, "show-secret": "true"})
	})
	if !strings.Contains(output, "created-secret") {
		t.Fatalf("output = %q, want explicitly requested object secret", output)
	}
}

func TestCreateUserCommonCredentialFieldsAreRedacted(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{
		"apiKey":     "api-key-secret",
		"apikey":     "lowercase-api-key-secret",
		"credential": "credential-secret",
		"accessKey":  "access-key-secret",
		"key":        "bare-key-secret",
	}
	op := findOperation(t, "user", "create")

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"body": `{"name":"robot","permissionRequestList":[]}`})
	})
	for _, secret := range []string{"api-key-secret", "lowercase-api-key-secret", "credential-secret", "access-key-secret", "bare-key-secret"} {
		if strings.Contains(output, secret) {
			t.Fatalf("output leaked %q: %s", secret, output)
		}
	}
	if !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("output = %q, want common credential fields redacted", output)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	op := findOperation(t, "repository", "delete")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"repository-id": "repo-123"})
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

func TestResponseContractRejectsUnexpectedStatusAndBody(t *testing.T) {
	op := findOperation(t, "repository", "get")
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusAccepted, Data: map[string]any{}}); err == nil || !strings.Contains(err.Error(), "expected HTTP 200") {
		t.Fatalf("responseError() = %v, want unexpected-status error", err)
	}
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err == nil || !strings.Contains(err.Error(), "expected JSON") {
		t.Fatalf("responseError() = %v, want missing-body error", err)
	}

	op = findOperation(t, "image", "delete")
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err != nil {
		t.Fatalf("responseError() = %v, want documented empty response accepted", err)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vcrAPI, error) { return fake, nil }
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
