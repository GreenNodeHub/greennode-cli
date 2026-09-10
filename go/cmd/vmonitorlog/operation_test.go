package vmonitorlog

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
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
	bytes     bool
	sensitive bool
}

type recordingAPI struct {
	calls         []recordedCall
	result        any
	empty         bool
	statusCode    int
	bytesResponse client.BytesResponse
	bytesError    error
	beforeBytes   func()
}

func TestEveryOperationExecutesReadOrOfflineDryRun(t *testing.T) {
	previous := newClient
	t.Cleanup(func() { newClient = previous })

	for _, op := range allOperations() {
		op := op
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			fake := &recordingAPI{}
			clientCalls := 0
			newClient = func(*cobra.Command) (vmonitorLogAPI, error) {
				clientCalls++
				return fake, nil
			}
			flags := validOperationFlags(t, op)
			if op.Mutation || op.Download {
				flags["dry-run"] = "true"
				if op.Download {
					flags["output-file"] = filepath.Join(t.TempDir(), "certificate.zip")
				}
				testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
				if clientCalls != 0 || len(fake.calls) != 0 {
					t.Fatalf("dry-run constructed client %d times and made calls %#v", clientCalls, fake.calls)
				}
				if op.Download {
					if _, err := os.Stat(flags["output-file"]); !os.IsNotExist(err) {
						t.Fatalf("download dry-run wrote output file: %v", err)
					}
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

func validOperationFlags(t *testing.T, op operation) map[string]string {
	t.Helper()
	flags := map[string]string{}
	parameters, err := pathParameters(op.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, parameter := range parameters {
		flag := flagName(parameter)
		switch flag {
		case "cdn-domain":
			flags[flag] = "logs.example.com"
		case "bucket-name":
			flags[flag] = "bucket-a"
		default:
			flags[flag] = "value-1"
		}
	}
	for _, parameter := range op.Queries {
		if parameter.Required {
			flags[queryFlagName(parameter)] = "value"
		}
	}
	if op.Body != nil {
		flags["body"] = `{}`
	}
	return flags
}

func TestEndpointUsesVerifiedLiveHost(t *testing.T) {
	if endpoint != "https://vmonitorapis.vngcloud.vn/log-api" {
		t.Fatalf("endpoint = %q, want live plural hostname", endpoint)
	}
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.recordResponse(method, path, query, body, false)
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitive(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.recordResponse(method, path, query, body, true)
}

func (f *recordingAPI) recordResponse(method, path string, query map[string]string, body any, sensitive bool) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body, sensitive: sensitive})
	result := f.result
	if result == nil {
		result = map[string]any{"ok": true}
	}
	statusCode := f.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return client.HTTPResponse{StatusCode: statusCode, Data: result, Empty: f.empty}, nil
}

func (f *recordingAPI) RequestBytes(method, path string, query map[string]string, _ []byte, _ string) (client.BytesResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, bytes: true})
	if f.beforeBytes != nil {
		f.beforeBytes()
	}
	if f.bytesError != nil {
		return client.BytesResponse{}, f.bytesError
	}
	if f.bytesResponse.Data == nil {
		f.bytesResponse = client.BytesResponse{StatusCode: http.StatusOK, Data: []byte("certificate-zip"), ContentType: "application/zip"}
	}
	return f.bytesResponse, nil
}

func TestQueriesUsePublishedWireNamesAndValidateRequiredValues(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "vcdn-mapping", "list"), map[string]string{
		"search":     "example",
		"sort-by":    "domain",
		"sort-order": "asc",
		"page":       "0",
		"size":       "20",
		"type":       "STATIC",
	})
	want := recordedCall{method: http.MethodGet, path: "/v1/vcdn-log-mapping", query: map[string]string{"query": "example", "sortBy": "domain", "sortOrder": "asc", "page": "0", "size": "20", "type": "STATIC"}}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
	listCommand := newOperationCommand(findOperation(t, "vcdn-mapping", "list"))
	if listCommand.Flags().Lookup("search") == nil || listCommand.Flags().Lookup("query") != nil {
		t.Fatal("API query filter must use local --search without shadowing the global --query output flag")
	}

	cmd := newOperationCommand(findOperation(t, "refill", "list"))
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "project-id") {
		t.Fatalf("RunE() error = %v, want required project-id error", err)
	}
}

func TestCredentialResponsesAreSensitiveAndRedactedByDefault(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{"storageSettings": map[string]any{"accessKey": "credential-value", "bucket": "logs"}}

	cmd := newOperationCommand(findOperation(t, "archive", "get"))
	setFlags(t, cmd, map[string]string{"archive-id": "archive-1"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	output := testutil.CaptureStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if strings.Contains(output, "credential-value") || !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("default output = %q, want redacted credential", output)
	}
	if !strings.Contains(stderr.String(), "Credential response redacted") {
		t.Fatalf("stderr = %q, want redaction warning", stderr.String())
	}
	if len(fake.calls) != 1 || !fake.calls[0].sensitive {
		t.Fatalf("calls = %#v, want sensitive request path", fake.calls)
	}

	cmd = newOperationCommand(findOperation(t, "archive", "get"))
	setFlags(t, cmd, map[string]string{"archive-id": "archive-1", "show-secret": "true"})
	output = testutil.CaptureStdout(t, func() {
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if !strings.Contains(output, "credential-value") {
		t.Fatalf("--show-secret output = %q, want credential", output)
	}
}

func TestReadOnlyPostCallsAPIWithoutMutationFlags(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "log", "search")
	if op.Mutation || newOperationCommand(op).Flags().Lookup("dry-run") != nil {
		t.Fatal("log search must stay read-only despite using POST")
	}
	runOperation(t, op, map[string]string{"project-id": "project-1", "body": `{"query":"level:error"}`})
	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPost || fake.calls[0].path != "/v1/projects/project-1/search-logs" {
		t.Fatalf("calls = %#v, want one read-only search POST", fake.calls)
	}
}

func TestDryRunSkipsClientAndRedactsSecrets(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorLogAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "archive", "create"), map[string]string{
			"body":    `{"name":"archive","accessKey":"secret-value","nested":{"authorization":"token-value"}}`,
			"dry-run": "true",
		})
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "secret-value") || strings.Contains(output, "token-value") || !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "POST") {
		t.Fatalf("dry-run output = %q, want redacted POST preview", output)
	}
}

func TestPathParametersAreValidatedBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorLogAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newOperationCommand(findOperation(t, "project", "get"))
	setFlags(t, cmd, map[string]string{"id": "bad/id"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("RunE() error = %v, want invalid id error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestPublishedDomainAndBucketNamesBuildPaths(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "vcdn-mapping", "update"), map[string]string{"cdn-domain": "static.example.com", "body": `{"projectId":"project-1"}`})
	runOperation(t, findOperation(t, "vstorage-bucket-mapping", "update"), map[string]string{"bucket-name": "logs.archive", "body": `{"enabled":true}`})
	if len(fake.calls) != 2 || fake.calls[0].path != "/v1/vcdn-log-mapping/edit/static.example.com" || fake.calls[1].path != "/v1/vstorage-bucket-log-mappings/logs.archive" {
		t.Fatalf("calls = %#v, want published domain and bucket paths", fake.calls)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	cmd := newOperationCommand(findOperation(t, "pipeline", "delete"))
	setFlags(t, cmd, map[string]string{"pipeline-id": "pipeline-1"})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected non-interactive refusal")
	}
	if len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("calls = %#v, confirmation error = %v; want no call and non-interactive refusal", fake.calls, cli.ConfirmationError())
	}
}

func TestCertificateDownloadWritesProtectedOutput(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "certificate.zip")
	fake := installRecordingClient(t)
	fake.bytesResponse = client.BytesResponse{StatusCode: http.StatusOK, Data: []byte("PK certificate"), ContentType: "application/zip"}
	runOperation(t, findOperation(t, "certificate", "download"), map[string]string{"project-id": "project-1", "cert-id": "cert-1", "output-file": outputFile})
	data, err := os.ReadFile(outputFile)
	if err != nil || string(data) != "PK certificate" {
		t.Fatalf("downloaded file = %q, %v", data, err)
	}
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("downloaded file mode = %v; want 0600", info.Mode().Perm())
	}
	if len(fake.calls) != 1 || !fake.calls[0].bytes || fake.calls[0].path != "/v1/downloads/certificates/projects/project-1/cert-1" {
		t.Fatalf("calls = %#v, want one binary certificate download", fake.calls)
	}
}

func TestCertificateDownloadDryRunIsOffline(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "certificate.zip")
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorLogAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "certificate", "download"), map[string]string{
			"project-id":  "project-1",
			"cert-id":     "cert-1",
			"output-file": outputFile,
			"dry-run":     "true",
		})
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("output file exists after dry run: %v", err)
	}
	if !strings.Contains(output, outputFile) || !strings.Contains(output, http.MethodGet) {
		t.Fatalf("dry-run output = %q, want method and output target", output)
	}
}

func TestCertificateDownloadWillNotOverwriteNonInteractive(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "certificate.zip")
	if err := os.WriteFile(outputFile, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	cmd := newOperationCommand(findOperation(t, "certificate", "download"))
	setFlags(t, cmd, map[string]string{"project-id": "project-1", "cert-id": "cert-1", "output-file": outputFile})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected non-interactive refusal")
	}
	data, err := os.ReadFile(outputFile)
	if err != nil || string(data) != "existing" || len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("file = %q, calls = %#v, confirmation error = %v", data, fake.calls, cli.ConfirmationError())
	}
}

func TestCertificateDownloadRejectsFileCreatedDuringRequest(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "certificate.zip")
	fake := installRecordingClient(t)
	fake.beforeBytes = func() {
		if err := os.WriteFile(outputFile, []byte("created-by-another-process"), 0o644); err != nil {
			t.Fatalf("create raced output: %v", err)
		}
	}
	cmd := newOperationCommand(findOperation(t, "certificate", "download"))
	setFlags(t, cmd, map[string]string{"project-id": "project-1", "cert-id": "cert-1", "output-file": outputFile, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("RunE() error = %v, want raced output refusal", err)
	}
	data, err := os.ReadFile(outputFile)
	if err != nil || string(data) != "created-by-another-process" {
		t.Fatalf("raced file = %q, %v; want untouched contents", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(outputFile))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v, %v", entries, err)
	}
}

func TestProtectedWriteTightensExistingFileMode(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "certificate.zip")
	if err := os.WriteFile(outputFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeProtectedFile(outputFile, []byte("new"), false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v; want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(outputFile)
	if err != nil || string(data) != "new" {
		t.Fatalf("file = %q, %v; want new contents", data, err)
	}
}

func TestProtectedWriteReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	path, alias := filepath.Join(dir, "certificate.zip"), filepath.Join(dir, "previous.zip")
	if err := os.WriteFile(path, []byte("fixture-previous"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	if err := writeProtectedFile(path, []byte("fixture-replacement"), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(alias)
	if err != nil || string(data) != "fixture-previous" {
		t.Fatalf("previous inode changed: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("temporary file leaked: %v, %v", entries, err)
	}
}

func TestCertificateDownloadRejectsNonregularOutput(t *testing.T) {
	for _, kind := range []string{"symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path, target := filepath.Join(dir, "certificate.zip"), filepath.Join(dir, "target.zip")
			if err := os.WriteFile(target, []byte("fixture-preserved"), 0o644); err != nil {
				t.Fatal(err)
			}
			if kind == "symlink" {
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
			calls := 0
			previous := newClient
			newClient = func(*cobra.Command) (vmonitorLogAPI, error) { calls++; return &recordingAPI{}, nil }
			t.Cleanup(func() { newClient = previous })
			cmd := newOperationCommand(findOperation(t, "certificate", "download"))
			setFlags(t, cmd, map[string]string{"project-id": "fixture-project", "cert-id": "fixture-cert", "output-file": path, "force": "true"})
			if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "regular file") {
				t.Fatalf("nonregular output accepted: %v", err)
			}
			if calls != 0 {
				t.Fatal("constructed a client for invalid output")
			}
			if err := writeProtectedFile(path, []byte("fixture-replacement"), false); err == nil {
				t.Fatal("writer accepted nonregular output")
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "fixture-preserved" {
				t.Fatalf("symlink target changed: %q, %v", data, err)
			}
		})
	}
}

func TestCertificateDownloadFailurePreservesOutput(t *testing.T) {
	for _, failure := range []string{"request", "empty"} {
		t.Run(failure, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "certificate.zip")
			if err := os.WriteFile(path, []byte("fixture-preserved"), 0o644); err != nil {
				t.Fatal(err)
			}
			fake := installRecordingClient(t)
			if failure == "request" {
				fake.bytesError = errors.New("fixture response failure")
			} else {
				fake.bytesResponse.Data = []byte{}
			}
			cmd := newOperationCommand(findOperation(t, "certificate", "download"))
			setFlags(t, cmd, map[string]string{"project-id": "fixture-project", "cert-id": "fixture-cert", "output-file": path, "force": "true"})
			if err := cmd.RunE(cmd, nil); err == nil {
				t.Fatal("failed response accepted")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "fixture-preserved" {
				t.Fatalf("previous output changed: %q, %v", data, err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0o644 {
				t.Fatalf("previous permissions changed: %v", err)
			}
		})
	}
}

func TestEmptySuccessDoesNotRenderJSONNull(t *testing.T) {
	fake := installRecordingClient(t)
	fake.empty = true
	fake.statusCode = http.StatusNoContent
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "project", "create-certificate"), map[string]string{"id": "project-1"})
	})
	if output != "" {
		t.Fatalf("empty response rendered %q, want no output", output)
	}
}

func TestUndocumentedEmptyResponseFailsContract(t *testing.T) {
	fake := installRecordingClient(t)
	fake.empty = true
	op := findOperation(t, "archive", "list")
	cmd := newOperationCommand(op)
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "expected JSON") {
		t.Fatalf("RunE() error = %v, want undocumented-empty-response contract error", err)
	}
}

func TestDestructiveDownloadOperationRegistersForceOnce(t *testing.T) {
	op := operation{
		Parent:      "certificate",
		Use:         "download",
		Short:       "test destructive download",
		Method:      http.MethodDelete,
		Path:        "/v1/test/{id}",
		Mutation:    true,
		Destructive: true,
		Download:    true,
	}

	var cmd *cobra.Command
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("newOperationCommand panicked registering --force twice: %v", r)
			}
		}()
		cmd = newOperationCommand(op)
	}()

	if got := cmd.Flags().Lookup("force"); got == nil {
		t.Fatal("expected a single --force flag to be registered")
	}
	if cmd.Flags().Lookup("output-file") == nil {
		t.Fatal("expected --output-file to still be registered for a Download operation")
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorLogAPI, error) { return fake, nil }
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
