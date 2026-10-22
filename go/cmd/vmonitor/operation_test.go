package vmonitor

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
	calls          []recordedCall
	result         any
	empty          bool
	sensitiveCalls int
}

func TestEveryOperationExecutesReadOrOfflineDryRun(t *testing.T) {
	previous := newClient
	t.Cleanup(func() { newClient = previous })

	for _, op := range allOperations() {
		op := op
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			fake := &recordingAPI{}
			clientCalls := 0
			newClient = func(*cobra.Command) (vmonitorAPI, error) {
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
		flags[flagName(parameter)] = "value-1"
	}
	for _, parameter := range op.Queries {
		if parameter.Required {
			flags[flagName(parameter.WireName)] = "value"
		}
	}
	if op.Body != nil {
		flags["body"] = `{}`
	}
	return flags
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body})
	result := f.result
	if result == nil {
		result = map[string]any{"ok": true}
	}
	return client.HTTPResponse{StatusCode: http.StatusOK, Data: result, Empty: f.empty}, nil
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitive(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	f.sensitiveCalls++
	return f.RequestWithStatus(method, path, query, body)
}

func TestQueriesUsePublishedWireNamesAndValidateRequiredValues(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "metric", "get-dimensions"), map[string]string{
		"name":       "cpu.util",
		"dimensions": "instance_id=ins-1",
		"end-time":   "2026-08-28T01:00:00Z",
	})
	want := recordedCall{method: http.MethodGet, path: "/api/v1/metrics/dimensions", query: map[string]string{"name": "cpu.util", "dimensions": "instance_id=ins-1", "end_time": "2026-08-28T01:00:00Z"}}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}

	cmd := newOperationCommand(findOperation(t, "metric", "get-dimensions"))
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("RunE() error = %v, want required name error", err)
	}
}

func TestInfrastructureHostListsDefaultPaginationForProviderCompatibility(t *testing.T) {
	fake := installRecordingClient(t)
	for _, test := range []struct {
		parent string
		use    string
		path   string
	}{
		{parent: "api-key", use: "list-metric", path: "/api/v1/apikeys/metric/list"},
		{parent: "infrastructure", use: "list-hosts", path: "/api/v1/infrastructure/hosts"},
		{parent: "infrastructure", use: "list-vserver-hosts", path: "/api/v1/infrastructure/vserver/hosts"},
	} {
		t.Run(test.parent+"/"+test.use, func(t *testing.T) {
			fake.calls = nil
			op := findOperation(t, test.parent, test.use)
			runOperation(t, op, nil)
			want := recordedCall{method: http.MethodGet, path: test.path, query: map[string]string{"page": "0", "size": "50"}}
			if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
				t.Fatalf("default call = %#v, want %#v", fake.calls, want)
			}

			fake.calls = nil
			runOperation(t, op, map[string]string{"page": "2", "size": "10"})
			want.query = map[string]string{"page": "2", "size": "10"}
			if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
				t.Fatalf("explicit call = %#v, want %#v", fake.calls, want)
			}
		})
	}
}

func TestStatisticsV2IsReadOnlyPost(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "statistic", "get-v2")
	if op.Mutation {
		t.Fatal("statistics v2 must stay read-only despite using POST")
	}
	if cmd := newOperationCommand(op); cmd.Flags().Lookup("dry-run") != nil {
		t.Fatal("statistics v2 must not expose mutation dry-run flags")
	}
	runOperation(t, op, map[string]string{"body": `{"name":"cpu.util","start_time":"2026-08-28T00:00:00Z"}`})
	if len(fake.calls) != 1 || fake.calls[0].method != http.MethodPost {
		t.Fatalf("calls = %#v, want one read-only POST", fake.calls)
	}
}

func TestDryRunSkipsClientAndRedactsSecrets(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "api-key", "create-metric"), map[string]string{
			"body":    `{"name":"preview","apiKey":"api-key-value","nested":{"authorization":"authorization-value"},"items":[{"credential":"credential-value"}]}`,
			"dry-run": "true",
		})
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "api-key-value") || strings.Contains(output, "authorization-value") || strings.Contains(output, "credential-value") || !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "POST") {
		t.Fatalf("dry-run output = %q, want redacted POST preview", output)
	}
}

func TestPathParametersAreValidatedBeforeClientConstruction(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	cmd := newOperationCommand(findOperation(t, "widget", "get"))
	setFlags(t, cmd, map[string]string{"dashboard-id": "bad/id", "widget-id": "widget-1"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "dashboard-id") {
		t.Fatalf("RunE() error = %v, want invalid dashboard-id error", err)
	}
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
}

func TestWidgetPathExposesUndeclaredDashboardID(t *testing.T) {
	fake := installRecordingClient(t)
	runOperation(t, findOperation(t, "widget", "get"), map[string]string{"dashboard-id": "dashboard-1", "widget-id": "widget-1"})
	if len(fake.calls) != 1 || fake.calls[0].path != "/api/v1/dashboards/dashboard-1/widgets/widget-1" {
		t.Fatalf("calls = %#v, want dashboard and widget path interpolation", fake.calls)
	}
}

func TestCamelCaseContractNamesUseKebabCaseFlags(t *testing.T) {
	cmd := newOperationCommand(findOperation(t, "dashboard", "delete-by-name"))
	if cmd.Flags().Lookup("resource-id") == nil || cmd.Flags().Lookup("resourceId") != nil {
		t.Fatalf("delete-by-name flags do not expose --resource-id exclusively")
	}
	if got := flagName("isDefault"); got != "is-default" {
		t.Fatalf("flagName(isDefault) = %q, want is-default", got)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	cmd := newOperationCommand(findOperation(t, "dashboard", "delete"))
	setFlags(t, cmd, map[string]string{"dashboard-id": "dashboard-1"})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected non-interactive refusal")
	}
	if len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("calls = %#v, confirmation error = %v; want no call and non-interactive refusal", fake.calls, cli.ConfirmationError())
	}
}

func TestEmptySuccessDoesNotRenderJSONNull(t *testing.T) {
	fake := installRecordingClient(t)
	fake.empty = true
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, findOperation(t, "dashboard", "delete"), map[string]string{"dashboard-id": "dashboard-1", "force": "true"})
	})
	if output != "" {
		t.Fatalf("empty response rendered %q, want no output", output)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command) (vmonitorAPI, error) { return fake, nil }
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
