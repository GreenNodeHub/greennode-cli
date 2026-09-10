package vdb

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
	userType  string
}

type recordingAPI struct {
	calls      []recordedCall
	result     any
	statusCode int
	empty      bool
	userType   string
}

func TestEveryOperationExecutesReadOrOfflineDryRun(t *testing.T) {
	previous := newClient
	t.Cleanup(func() { newClient = previous })

	for _, op := range allOperations() {
		op := op
		t.Run(op.Parent+"/"+op.Use, func(t *testing.T) {
			fake := &recordingAPI{statusCode: http.StatusOK}
			clientCalls := 0
			newClient = func(*cobra.Command, bool) (vdbAPI, error) {
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
			if clientCalls != 1 || len(fake.calls) != 1 || fake.calls[0].method != op.Method || fake.calls[0].noRetry != op.SecretResponse {
				t.Fatalf("read client calls = %d, requests = %#v; want one %s request with sensitive no-retry=%t", clientCalls, fake.calls, op.Method, op.SecretResponse)
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
		if !parameter.Required {
			continue
		}
		flag := parameter.Flag
		if flag == "" {
			flag = flagName(parameter.WireName)
		}
		switch parameter.Kind {
		case queryInteger:
			flags[flag] = "1"
		case queryBoolean:
			flags[flag] = "true"
		case queryObject:
			flags[flag] = `{}`
		default:
			flags[flag] = "value"
		}
	}
	if op.Body != nil {
		if op.Body.Kind == bodyArray {
			flags["body"] = `[]`
		} else {
			flags["body"] = `{}`
		}
	}
	return flags
}

func (f *recordingAPI) SetHeader(name, value string) {
	if name == "user-type" {
		f.userType = value
	}
}

func (f *recordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, false, false, false)
}

func (f *recordingAPI) RequestWithStatusNoRetry(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, false, false)
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitive(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, true, false)
}

func (f *recordingAPI) RequestWithStatusNoRetryRaw(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, false, true)
}

func (f *recordingAPI) RequestWithStatusNoRetrySensitiveRaw(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true, true, true)
}

func (f *recordingAPI) record(method, path string, query map[string]string, body any, noRetry, sensitive, raw bool) (client.HTTPResponse, error) {
	f.calls = append(f.calls, recordedCall{method: method, path: path, query: query, body: body, noRetry: noRetry, sensitive: sensitive, raw: raw, userType: f.userType})
	status := f.statusCode
	if status == 0 {
		status = http.StatusOK
	}
	result := f.result
	if result == nil {
		result = map[string]any{"data": []any{}}
	}
	return client.HTTPResponse{StatusCode: status, Data: result, Empty: f.empty}, nil
}

func TestReadBuildsExactPathAndQuery(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "relational-instance", "get-history-db")
	runOperation(t, op, map[string]string{"instance-id": "db-123", "page-number": "2", "page-size": "50"})

	want := recordedCall{method: http.MethodGet, path: "/vdb-relational/v1/database-instances/db-123/histories", query: map[string]string{"pageNumber": "2", "pageSize": "50"}}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestObjectQueryAndTypedQueryValidation(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "memory-instance", "get-database-instances-by-user")
	runOperation(t, op, map[string]string{"filter": `{"status":["ACTIVE"],"name":"db"}`, "page-number": "1", "page-size": "20"})
	if got := fake.calls[0].query["filterRequest"]; got != `{"name":"db","status":["ACTIVE"]}` {
		t.Fatalf("filterRequest = %q", got)
	}

	op = findOperation(t, "kafka-cluster", "update-authentication")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"cluster-id": "cluster-1", "mtls-authen": "maybe", "sasl-authen": "true", "dry-run": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("RunE() error = %v, want boolean validation", err)
	}
}

func TestInstanceListsRejectZeroBasedPaginationBeforeCallingProvider(t *testing.T) {
	for _, parent := range []string{"memory-instance", "relational-instance"} {
		t.Run(parent, func(t *testing.T) {
			fake := installRecordingClient(t)
			cmd := newOperationCommand(findOperation(t, parent, "get-database-instances-by-user"))
			setFlags(t, cmd, map[string]string{"filter": `{}`, "page-number": "0", "page-size": "20"})
			if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "at least 1") {
				t.Fatalf("RunE() error = %v, want one-based pagination validation", err)
			}
			if len(fake.calls) != 0 {
				t.Fatalf("provider calls = %#v, want none", fake.calls)
			}
		})
	}
}

func TestBooleanQueriesAreCanonicalized(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "kafka-cluster", "update-authentication")
	runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "mtls-authen": "1", "sasl-authen": "t"})

	if got := fake.calls[0].query; !reflect.DeepEqual(got, map[string]string{"mtlsAuthen": "true", "saslAuthen": "true"}) {
		t.Fatalf("query = %#v, want canonical booleans", got)
	}
}

func TestWriteUsesNoRetryAndPreservesNumbersAndUserType(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "postgresql-cluster", "create-postgre-cluster")
	runOperation(t, op, map[string]string{"body": `{"volumeSize":9007199254740993}`, "user-type": "IAM_USER"})
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || fake.calls[0].userType != "IAM_USER" {
		t.Fatalf("calls = %#v, want no-retry request with IAM_USER", fake.calls)
	}
	body := fake.calls[0].body.(map[string]any)
	if got, ok := body["volumeSize"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("volumeSize = %#v, want lossless json.Number", body["volumeSize"])
	}
}

func TestPoCUsesDocumentedAutomaticPaymentFields(t *testing.T) {
	fake := installRecordingClient(t)

	direct := findOperation(t, "postgresql-cluster", "create-postgre-cluster")
	runOperation(t, direct, map[string]string{"body": `{"name":"poc-cluster"}`, "poc": "true"})
	if len(fake.calls) != 1 || fake.calls[0].userType != "IAM_USER" {
		t.Fatalf("direct PoC call = %#v, want one automatic-payment call", fake.calls)
	}
	if body := fake.calls[0].body.(map[string]any); body["isPoc"] != true {
		t.Fatalf("direct PoC body = %#v, want isPoc=true", body)
	}

	fake.calls = nil
	nested := findOperation(t, "relational-instance", "resize-instance")
	runOperation(t, nested, map[string]string{
		"instance-id": "db-1",
		"body":        `{"action":"resize","databaseInstances":[{"instancesId":"db-1","config":{"packageId":"pkg-1"}}]}`,
		"poc":         "true",
	})
	if len(fake.calls) != 1 || fake.calls[0].userType != "IAM_USER" {
		t.Fatalf("nested PoC call = %#v, want one automatic-payment call", fake.calls)
	}
	body := fake.calls[0].body.(map[string]any)
	instance := body["databaseInstances"].([]any)[0].(map[string]any)
	if config := instance["config"].(map[string]any); config["poc"] != true {
		t.Fatalf("nested PoC config = %#v, want poc=true", config)
	}
}

func TestPoCRejectsCheckoutHeaderAndIncompleteNestedBody(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "relational-instance", "create-relational-database-instance")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"body": `{}`, "poc": "true", "user-type": "ROOT_USER"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "automatic payment") {
		t.Fatalf("RunE() error = %v, want automatic-payment conflict", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no request after PoC conflict", fake.calls)
	}

	op = findOperation(t, "memory-instance", "resize-instance")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"db-instance-id": "db-1", "body": `{}`, "poc": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "databaseInstances") {
		t.Fatalf("RunE() error = %v, want nested-body validation", err)
	}
}

func TestPoCIsLimitedToDocumentedVDBOperations(t *testing.T) {
	var supported []string
	for _, op := range allOperations() {
		cmd := newOperationCommand(op)
		hasFlag := cmd.Flags().Lookup("poc") != nil
		if op.Extra == nil {
			if hasFlag {
				t.Errorf("%s %s exposes undocumented --poc", op.Method, op.Path)
			}
			continue
		}
		if !op.UserType || !hasFlag {
			t.Errorf("%s %s PoC contract requires --user-type and --poc", op.Method, op.Path)
		}
		supported = append(supported, op.OperationID)
	}
	if got, want := len(supported), 11; got != want {
		t.Fatalf("documented PoC operation count = %d (%v), want %d", got, supported, want)
	}
}

func TestBodyKindRequiredItemAndActionValidation(t *testing.T) {
	installRecordingClient(t)
	op := findOperation(t, "memory-backup", "delete-backups")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"body": `{}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "JSON array") {
		t.Fatalf("RunE() error = %v, want array error", err)
	}
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"body": `[{}]`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "backupId") {
		t.Fatalf("RunE() error = %v, want required item field error", err)
	}

	op = findOperation(t, "relational-instance", "stop-database-instances")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"instance-id": "db-1", "body": `{"action":"start"}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), `must be "stop"`) {
		t.Fatalf("RunE() error = %v, want fixed action error", err)
	}

	op = findOperation(t, "memory-backup", "restore-backup")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"backup-id": "backup-1", "body": `{"action":"restore"}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), `must be "restore_backup"`) {
		t.Fatalf("RunE() error = %v, want documented restore action error", err)
	}
}

func TestBodyTargetMustMatchPath(t *testing.T) {
	fake := installRecordingClient(t)
	op := findOperation(t, "relational-instance", "update-database-setting")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"instance-id": "db-1", "body": `{"dbInstanceId":"db-2"}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --instance-id") {
		t.Fatalf("RunE() error = %v, want body target mismatch", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no API call", fake.calls)
	}

	op = findOperation(t, "memory-instance", "reboot-database-instances")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"db-instance-id": "db-1", "body": `{"action":"reboot","databaseInstances":[{"instancesId":"db-2"}]}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --db-instance-id") {
		t.Fatalf("RunE() nested error = %v, want body target mismatch", err)
	}

	op = findOperation(t, "relational-backup", "restore-backup")
	cmd = newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"id": "backup-1", "body": `{"action":"restore_backup","databaseInstances":[{"config":{"backupId":"backup-2"}}]}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --id") {
		t.Fatalf("RunE() restore error = %v, want backup target mismatch", err)
	}
}

func TestDryRunIsOfflineAndRedactsCredentials(t *testing.T) {
	clientCalls := 0
	previous := newClient
	newClient = func(*cobra.Command, bool) (vdbAPI, error) {
		clientCalls++
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "kafka-user", "create-user")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "body": `{"name":"robot","password":"do-not-print"}`, "dry-run": "true"})
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if strings.Contains(output, "do-not-print") || !strings.Contains(output, "REDACTED") || !strings.Contains(output, "<configured>") {
		t.Fatalf("dry-run output = %q", output)
	}
}

func TestOperationWithoutPortalHeaderDoesNotRequireOrPreviewIt(t *testing.T) {
	clientRequiresPortalUserID := true
	previous := newClient
	newClient = func(_ *cobra.Command, requirePortalUserID bool) (vdbAPI, error) {
		clientRequiresPortalUserID = requirePortalUserID
		return &recordingAPI{}, nil
	}
	t.Cleanup(func() { newClient = previous })

	op := findOperation(t, "memory-instance", "get-database-instances-by-user")
	runOperation(t, op, map[string]string{"filter": `{}`, "page-number": "1", "page-size": "20"})
	if clientRequiresPortalUserID {
		t.Fatal("operation required portal_user_id despite omitting the header in the official contract")
	}

	fields := dryRunFields(op, op.Path, nil, nil, "")
	if _, ok := fields["headers"]; ok {
		t.Fatalf("dry-run fields = %#v, want no undocumented headers", fields)
	}
}

func TestCredentialResponseIsSensitiveAndRedactedByDefault(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = []any{"username", "secret-value"}
	op := findOperation(t, "kafka-user", "get-user-authen-credential")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "user-id": "user-1"})
	})
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || !fake.calls[0].sensitive {
		t.Fatalf("calls = %#v, want sensitive no-retry request", fake.calls)
	}
	if strings.Contains(output, "secret-value") || !strings.Contains(output, client.RedactedValue) {
		t.Fatalf("output = %q, want redacted credentials", output)
	}

	output = testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "user-id": "user-1", "show-secret": "true"})
	})
	if !strings.Contains(output, "secret-value") {
		t.Fatalf("output = %q, want explicit credential output", output)
	}
}

func TestRawStringResponsesUseNoRetryClientPaths(t *testing.T) {
	fake := installRecordingClient(t)
	fake.result = "updated"
	op := findOperation(t, "kafka-cluster", "update-public-access")
	runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "enable": "true"})
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || !fake.calls[0].raw || fake.calls[0].sensitive {
		t.Fatalf("calls = %#v, want raw no-retry request", fake.calls)
	}

	fake.calls = nil
	fake.result = "rotated-secret"
	op = findOperation(t, "kafka-user", "regenerate-user-authen-credential")
	output := testutil.CaptureStdout(t, func() {
		runOperation(t, op, map[string]string{"cluster-id": "cluster-1", "user-id": "user-1", "force": "true", "show-secret": "true"})
	})
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || !fake.calls[0].raw || !fake.calls[0].sensitive {
		t.Fatalf("calls = %#v, want sensitive raw no-retry request", fake.calls)
	}
	if !strings.Contains(output, "rotated-secret") {
		t.Fatalf("output = %q, want explicit credential output", output)
	}
}

func TestDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	op := findOperation(t, "kafka-cluster", "delete-cluster")
	cmd := newOperationCommand(op)
	setFlags(t, cmd, map[string]string{"cluster-id": "cluster-1"})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected confirmation refusal")
	}
	if len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("calls = %#v confirmation error = %v", fake.calls, cli.ConfirmationError())
	}
}

func TestResponseContract(t *testing.T) {
	op := findOperation(t, "kafka-cluster", "list-clusters")
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err == nil {
		t.Fatal("empty HTTP 200 accepted")
	}
	stringOp := findOperation(t, "kafka-configuration", "delete-config-group")
	if err := responseError(stringOp, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err != nil {
		t.Fatalf("empty HTTP 200 string response rejected: %v", err)
	}
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusAccepted, Empty: true}); err != nil {
		t.Fatalf("empty HTTP 202 rejected: %v", err)
	}
	if err := responseError(op, client.HTTPResponse{StatusCode: http.StatusNoContent, Empty: true}); err != nil {
		t.Fatalf("empty HTTP 204 rejected: %v", err)
	}
}

func installRecordingClient(t *testing.T) *recordingAPI {
	t.Helper()
	fake := &recordingAPI{}
	previous := newClient
	newClient = func(*cobra.Command, bool) (vdbAPI, error) { return fake, nil }
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
	t.Fatalf("operation %s/%s not found", parent, use)
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
