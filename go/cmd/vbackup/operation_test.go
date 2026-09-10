package vbackup

import (
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
			if op.Extra == responseArray {
				fake.result = []any{}
			}
			clientCalls := 0
			newClient = func(*cobra.Command) (vbackupAPI, error) {
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
	if op.Body != nil {
		flags["body"] = `{}`
	}
	return flags
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body})
	statusCode := f.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	result := f.result
	if result == nil {
		result = map[string]any{"ok": true}
	}
	return client.HTTPResponse{StatusCode: statusCode, Data: result, Empty: f.empty}, nil
}

func TestListUsesPublishedQueryNames(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "server", "list"), map[string]string{
		"backend-id": "backend-1",
		"project-id": "pro-123",
		"server-id":  "server-1",
		"page":       "2",
		"size":       "25",
	})
	want := recordedCall{method: http.MethodGet, path: "/v1/backup-instances", query: map[string]string{"backendId": "backend-1", "projectId": "pro-123", "serverId": "server-1", "page": "2", "size": "25"}}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestQueryIdentifiersAreValidatedBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newTestCommand(findOperation(t, "server", "list"))
	setFlags(t, cmd, map[string]string{"backend-id": "bad/id", "endpoint-url": "https://test.vngcloud.vn"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "backend-id") {
		t.Fatalf("RunE() error = %v, want backend ID validation error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestVolumeUsageIsAReadOnlyPost(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = []any{}
	op := findOperation(t, "volume", "usage")
	if op.Mutation {
		t.Fatal("volume usage must stay read-only despite using POST")
	}
	runOperation(t, op, map[string]string{"body": `{"backendId":"backend-1","projectId":"pro-123","volumeIds":["volume-1"]}`})
	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPost {
		t.Fatalf("calls = %#v, want one read-only POST", fake.calls)
	}
}

func TestDryRunSkipsClientAndRedactsSecrets(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "policy", "create"), map[string]string{
			"body":    `{"name":"preview","config":{"accessToken":"do-not-print"}}`,
			"dry-run": "true",
		})
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "do-not-print") || !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "POST") {
		t.Fatalf("dry-run output = %q, want redacted POST preview", output)
	}
}

func TestInvalidPathIDFailsBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newTestCommand(findOperation(t, "server", "get"))
	setFlags(t, cmd, map[string]string{"id": "bad/id", "endpoint-url": "https://test.vngcloud.vn"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("RunE() error = %v, want invalid ID error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestNonHCM3RegionFailsBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newTestCommand(findOperation(t, "backend", "list"))
	setFlags(t, cmd, map[string]string{"region": "HAN"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "HCM-3") {
		t.Fatalf("RunE() error = %v, want HCM-3 region error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	cmd := newTestCommand(findOperation(t, "policy", "delete"))
	setFlags(t, cmd, map[string]string{"id": "policy-1", "endpoint-url": "https://test.vngcloud.vn"})
	if err := cmd.RunE(cmd, nil); err == nil || err != cli.ConfirmationError() {
		t.Fatalf("RunE() error = %v, want confirmation refusal", err)
	}
	if len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("calls = %#v, confirmation error = %v; want no call and non-interactive refusal", fake.calls, cli.ConfirmationError())
	}
}

func TestDocumentedAndUndocumentedEmptyResponses(t *testing.T) {
	if err := responseError(findOperation(t, "server", "disable"), client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err != nil {
		t.Fatalf("documented empty disable response = %v", err)
	}
	if err := responseError(findOperation(t, "backend", "list"), client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err == nil || !strings.Contains(err.Error(), "expected JSON") {
		t.Fatalf("undocumented empty list response = %v, want error", err)
	}
}

func TestDocumentedEmptyResponseDoesNotRenderJSONNull(t *testing.T) {
	fake := installRecordingClient(t)
	fake.statusCode = http.StatusOK
	fake.empty = true
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "server", "disable"), map[string]string{"id": "backup-1"})
	})
	if output != "" {
		t.Fatalf("empty documented response rendered %q, want no output", output)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vbackupAPI, error) { return fake, nil }
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

func newTestCommand(op operation) *cobra.Command {
	cmd := newOperationCommand(op)
	cmd.Flags().String("region", "", "test region")
	cmd.Flags().String("profile", "", "test profile")
	cmd.Flags().String("endpoint-url", "", "test endpoint")
	return cmd
}

func runOperation(t *testing.T, op operation, flags map[string]string) {
	t.Helper()
	cmd := newTestCommand(op)
	if _, ok := flags["endpoint-url"]; !ok {
		flags = cloneFlags(flags)
		flags["endpoint-url"] = "https://test.vngcloud.vn"
	}
	setFlags(t, cmd, flags)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func cloneFlags(flags map[string]string) map[string]string {
	cloned := make(map[string]string, len(flags)+1)
	for key, value := range flags {
		cloned[key] = value
	}
	return cloned
}

func setFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}
