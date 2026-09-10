package vlb

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
			fake := &recordingAPI{statusCode: op.Status}
			clientCalls := 0
			newClient = func(*cobra.Command) (vlbAPI, error) {
				clientCalls++
				return fake, nil
			}
			flags := validOperationFlags(t, op)
			if isMutation(op) {
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
		switch parameter.Kind {
		case queryInteger:
			flags[parameter.Flag] = "1"
		case queryObject:
			flags[parameter.Flag] = `{}`
		default:
			flags[parameter.Flag] = "value-1"
		}
	}
	if op.Body != nil {
		body := map[string]any{}
		for _, field := range op.Body.RequiredFields {
			body[field] = "value"
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
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body})
	status := f.statusCode
	if status == 0 {
		status = http.StatusOK
	}
	result := f.result
	if result == nil {
		result = map[string]any{"data": map[string]any{}}
	}
	return client.HTTPResponse{StatusCode: status, Data: result, Empty: f.empty}, nil
}

func TestReadBuildsExactPathAndObjectQueries(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "load-balancer", "list")
	runOperation(t, op, map[string]string{
		"project-id":   "pro-123",
		"filter":       `{ "name": "edge", "ids": ["lb-2", "lb-1"] }`,
		"page-request": `{ "size": 25, "page": 0 }`,
	})

	want := recordedCall{
		method: http.MethodGet,
		path:   "/v2/pro-123/loadBalancers",
		query: map[string]string{
			"filter":      `{"ids":["lb-2","lb-1"],"name":"edge"}`,
			"pageRequest": `{"page":0,"size":25}`,
		},
	}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestReadValidatesIDsAndTypedQueriesBeforeClient(t *testing.T) {
	fake := installRecordingClient(t)

	op := findOperation(t, "load-balancer", "list-packages")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "zone-id": "bad/zone"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "zone-id") {
		t.Fatalf("RunE() error = %v, want zone ID validation error", err)
	}

	op = findOperation(t, "certificate", "list")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "page": "one"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "32-bit integer") {
		t.Fatalf("RunE() error = %v, want integer validation error", err)
	}

	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no API call after validation failures", fake.calls)
	}
}

func TestWritePreservesNumbersAndValidatesRequiredFields(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "load-balancer", "create")
	runOperation(t, op, map[string]string{
		"project-id": "pro-123",
		"body":       `{"name":"edge","packageId":"pkg-1","scheme":"internet","subnetId":"sub-1","type":"L7","capacity":9007199254740993}`,
	})

	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPost {
		t.Fatalf("calls = %#v, want one POST", fake.calls)
	}
	body := fake.calls[0].body.(map[string]any)
	if got, ok := body["capacity"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("capacity = %#v, want lossless json.Number", body["capacity"])
	}

	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"project-id": "pro-123", "body": `{"name":"edge"}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), `"packageId"`) {
		t.Fatalf("RunE() error = %v, want missing packageId error", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("calls = %#v, want no second API call", fake.calls)
	}
}

func TestDryRunIsOfflineAndRedactsCertificateSecrets(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vlbAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "certificate", "import")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{
			"project-id": "pro-123",
			"body":       `{"name":"edge-cert","type":"PEM","privateKey":"do-not-print","passphrase":"also-secret","certificate":"public-cert"}`,
			"dry-run":    "true",
		})
	})

	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "do-not-print") || strings.Contains(output, "also-secret") {
		t.Fatalf("dry-run leaked certificate secrets: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "public-cert") {
		t.Fatalf("dry-run output = %q, want redacted private material and visible certificate", output)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	op := findOperation(t, "pool", "delete")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{
		"project-id":       "pro-123",
		"load-balancer-id": "lb-123",
		"pool-id":          "pool-123",
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

func TestResponseContractRejectsUnexpectedStatusAndEmptyBody(t *testing.T) {
	op := findOperation(t, "load-balancer", "get")
	if err := responseError(client.HTTPResponse{StatusCode: http.StatusAccepted, Data: map[string]any{}}, op); err == nil || !strings.Contains(err.Error(), "expected HTTP 200") {
		t.Fatalf("responseError() = %v, want unexpected-status error", err)
	}
	if err := responseError(client.HTTPResponse{StatusCode: http.StatusOK, Data: map[string]any{}, Empty: true}, op); err == nil || !strings.Contains(err.Error(), "expected JSON") {
		t.Fatalf("responseError() = %v, want unexpected-empty error", err)
	}

	op = findOperation(t, "listener", "update")
	if err := responseError(client.HTTPResponse{StatusCode: http.StatusOK, Data: map[string]any{}}, op); err == nil {
		t.Fatal("accepted a body for a documented empty response")
	}
	if err := responseError(client.HTTPResponse{StatusCode: http.StatusOK, Data: map[string]any{}, Empty: true}, op); err != nil {
		t.Fatalf("responseError() = %v, want documented empty response accepted", err)
	}
}

func TestResponseContractRejectsUndocumentedAsyncStatus(t *testing.T) {
	for _, target := range [][2]string{{"listener", "create"}, {"listener", "update"}, {"pool", "update"}} {
		op := findOperation(t, target[0], target[1])
		if err := responseError(client.HTTPResponse{StatusCode: http.StatusAccepted, Data: map[string]any{}}, op); err == nil {
			t.Fatalf("accepted undocumented HTTP 202 for %s/%s", target[0], target[1])
		}
	}
}

func TestHTTP200FailureEnvelopeReturnsProviderMessage(t *testing.T) {
	op := findOperation(t, "load-balancer", "create")
	err := responseError(client.HTTPResponse{
		StatusCode: http.StatusOK,
		Data: map[string]any{
			"success":  false,
			"code":     json.Number("114"),
			"errorMsg": "Creating load balancer failed",
		},
	}, op)
	if err == nil || err.Error() != "Creating load balancer failed" {
		t.Fatalf("responseError() = %v, want provider errorMsg", err)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vlbAPI, error) { return fake, nil }
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
