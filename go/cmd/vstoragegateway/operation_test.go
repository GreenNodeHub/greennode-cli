package vstoragegateway

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

type recordedCall struct {
	method string
	path   string
	query  map[string]string
}

type recordingAPI struct {
	calls  []recordedCall
	result any
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, _ any) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query})
	result := f.result
	if result == nil {
		result = []any{}
	}
	return client.HTTPResponse{StatusCode: http.StatusOK, Data: result}, nil
}

func TestOperationsForwardPublishedQueries(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "container"), map[string]string{"region-id": "hcm03", "project-id": "project-1"})
	want := recordedCall{method: http.MethodGet, path: "/v1/containers", query: map[string]string{"region_id": "hcm03", "project_id": "project-1"}}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("calls = %#v, want %#v", fake.calls, want)
	}
}

func TestEveryReadOperationExecutesExactlyOnce(t *testing.T) {
	previous := newClient
	t.Cleanup(func() { newClient = previous })
	for _, op := range allOperations() {
		op := op
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			fake := &recordingAPI{}
			clientCalls := 0
			newClient = func(*cobra.Command) (gatewayAPI, error) {
				clientCalls++
				return fake, nil
			}
			flags := map[string]string{}
			for _, parameter := range op.Queries {
				if parameter.Required {
					flags[parameter.Flag] = "value-1"
				}
			}
			runOperation(t, op, flags)
			if clientCalls != 1 || len(fake.calls) != 1 || fake.calls[0].method != op.Method || fake.calls[0].path != op.Path {
				t.Fatalf("client calls = %d, requests = %#v; want one %s %s", clientCalls, fake.calls, op.Method, op.Path)
			}
		})
	}
}

func TestEndpointUsesVerifiedLiveHost(t *testing.T) {
	if endpoint != "https://vmonitorapis.vngcloud.vn/vstorage-gateway" {
		t.Fatalf("endpoint = %q, want live plural hostname", endpoint)
	}
}

func TestNonArrayJSONResponseFailsContract(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = map[string]any{"items": []any{}}
	cmd := newOperationCommand(findOperation(t, "region"))
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "expected an array") {
		t.Fatalf("RunE() error = %v, want response-array contract error", err)
	}
}

func TestRequiredQueryIsValidatedBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (gatewayAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newOperationCommand(findOperation(t, "project"))
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("RunE() returned nil without --region-id")
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestQueryIdentifiersAreValidatedBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (gatewayAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newOperationCommand(findOperation(t, "project"))
	if err := cmd.Flags().Set("region-id", "bad/id"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "region-id") {
		t.Fatalf("RunE() error = %v, want region ID validation error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (gatewayAPI, error) { return fake, nil }
	t.Cleanup(func() { newClient = previous })
	return fake
}

func findOperation(t *testing.T, parent string) operation {
	t.Helper()
	for _, op := range allOperations() {
		if op.Parent == parent {
			return op
		}
	}
	t.Fatalf("operation for %s not found", parent)
	return operation{}
}

func runOperation(t *testing.T, op operation, flags map[string]string) {
	t.Helper()
	cmd := newOperationCommand(op)
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}
