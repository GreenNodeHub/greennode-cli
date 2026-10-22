package vserver

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type ancillaryRecordedCall struct {
	method  string
	path    string
	query   map[string]string
	body    any
	noRetry bool
}

type ancillaryRecordingAPI struct {
	calls      []ancillaryRecordedCall
	result     any
	statusCode int
	empty      bool
}

func TestEveryAncillaryOperationExecutesReadOrOfflineDryRun(t *testing.T) {
	previous := newAncillaryClient
	t.Cleanup(func() { newAncillaryClient = previous })

	for _, op := range ancillaryOperations() {
		op := op
		t.Run(strings.Join(append(op.Parents, op.Use), "/"), func(t *testing.T) {
			fake := &ancillaryRecordingAPI{statusCode: op.SuccessStatus, empty: !op.ResponseBody}
			clientCalls := 0
			newAncillaryClient = func(*cobra.Command) (ancillaryAPI, *config.Config, error) {
				clientCalls++
				return fake, &config.Config{ProjectID: "project-1", Output: "json"}, nil
			}
			flags := validAncillaryOperationFlags(t, op)
			if op.Mutation {
				flags["dry-run"] = "true"
				testutil.CaptureStdout(t, func() { runAncillaryOperation(t, op, flags) })
				if clientCalls != 0 || len(fake.calls) != 0 {
					t.Fatalf("dry-run constructed client %d times and made calls %#v", clientCalls, fake.calls)
				}
				return
			}

			testutil.CaptureStdout(t, func() { runAncillaryOperation(t, op, flags) })
			if clientCalls != 1 || len(fake.calls) != 1 || fake.calls[0].method != op.Method || fake.calls[0].noRetry {
				t.Fatalf("read client calls = %d, requests = %#v; want one retryable %s request", clientCalls, fake.calls, op.Method)
			}
		})
	}
}

func validAncillaryOperationFlags(t *testing.T, op ancillaryOperation) map[string]string {
	t.Helper()
	flags := map[string]string{}
	parameters, err := ancillaryPathParameters(op.Path, op.PathKinds)
	if err != nil {
		t.Fatal(err)
	}
	for _, parameter := range parameters {
		if ancillaryProjectParameter(op, parameter.Name) {
			continue
		}
		flag := ancillaryFlagName(parameter.Name)
		if parameter.Kind == ancillaryPathInt32 {
			flags[flag] = "1"
		} else {
			flags[flag] = "value-1"
		}
	}
	for _, parameter := range op.Queries {
		if !parameter.Required {
			continue
		}
		flag := parameter.Flag
		if flag == "" {
			flag = ancillaryFlagName(parameter.Name)
		}
		switch parameter.Kind {
		case ancillaryQueryInteger:
			flags[flag] = "1"
		case ancillaryQueryBoolean:
			flags[flag] = "true"
		default:
			flags[flag] = "value"
		}
	}
	if op.Body != nil {
		body := map[string]any{}
		for _, field := range op.Body.RequiredFields {
			body[field] = "value"
		}
		switch op.OperationID {
		case "updateRulesUsingPUT":
			body["aclId"] = flags["acl-id"]
		case "updateAssociatedSubnetsUsingPUT":
			body["aclId"] = flags["uuid"]
		case "deletePersistentVolumeUsingDELETE":
			body["persistentVolumeId"] = flags["pv-id"]
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		flags["body"] = string(encoded)
	}
	return flags
}

func (f *ancillaryRecordingAPI) RequestWithStatus(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, false)
}

func (f *ancillaryRecordingAPI) RequestWithStatusNoRetry(method, path string, query map[string]string, body any) (client.HTTPResponse, error) {
	return f.record(method, path, query, body, true)
}

func (f *ancillaryRecordingAPI) record(method, path string, query map[string]string, body any, noRetry bool) (client.HTTPResponse, error) {
	f.calls = append(f.calls, ancillaryRecordedCall{method: method, path: path, query: query, body: body, noRetry: noRetry})
	result := f.result
	if result == nil {
		result = map[string]any{"data": []any{}}
	}
	return client.HTTPResponse{StatusCode: f.statusCode, Data: result, Empty: f.empty}, nil
}

func TestAncillaryReadBuildsConfiguredProjectPathAndTypedQuery(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "History", "listServerMigrationHistoryUsingGET")
	fake.statusCode = op.SuccessStatus
	runAncillaryOperation(t, op, map[string]string{"page": "2", "size": "50", "server-id": "server-1", "status": "SUCCESS"})

	want := ancillaryRecordedCall{
		method: http.MethodGet,
		path:   "/v1/project-1/histories/server-migration",
		query:  map[string]string{"page": "2", "size": "50", "serverId": "server-1", "status": "SUCCESS"},
	}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", fake.calls, want)
	}
}

func TestAncillaryProjectGetUsesExplicitProjectID(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Project", "getProjectUsingGET")
	fake.statusCode = op.SuccessStatus
	cmd := newAncillaryOperationCommand(op)
	projectID := cmd.Flags().Lookup("project-id")
	if projectID == nil || len(projectID.Annotations[cobra.BashCompOneRequiredFlag]) == 0 {
		t.Fatal("project get must expose a required --project-id resource flag")
	}
	setAncillaryFlags(t, cmd, map[string]string{"project-id": "project-other"})
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	want := ancillaryRecordedCall{method: http.MethodGet, path: "/v1/projects/project-other"}
	if len(fake.calls) != 1 || !reflect.DeepEqual(fake.calls[0], want) {
		t.Fatalf("calls = %#v, want %#v", fake.calls, want)
	}
}

func TestAncillaryIntegerPathParameterUsesOfficialType(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Market-place", "getAppTemplateUsingGET")
	fake.statusCode = op.SuccessStatus
	runAncillaryOperation(t, op, map[string]string{"app-template-id": "1"})

	if len(fake.calls) != 1 || fake.calls[0].path != "/v1/app-template/1" {
		t.Fatalf("calls = %#v, want request for numeric app template ID", fake.calls)
	}

	fake.calls = nil
	cmd := newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"app-template-id": "abc"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "32-bit integer") {
		t.Fatalf("RunE() error = %v, want integer path validation error", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no API call for invalid path", fake.calls)
	}
}

func TestAncillaryWriteUsesNoRetryAndCanonicalBoolean(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Market-place-migration", "migrateMpUsingPOST")
	fake.statusCode = op.SuccessStatus
	runAncillaryOperation(t, op, map[string]string{"all": "1"})
	if len(fake.calls) != 1 || !fake.calls[0].noRetry || fake.calls[0].query["all"] != "true" {
		t.Fatalf("calls = %#v, want no-retry request with canonical boolean", fake.calls)
	}

	fake.calls = nil
	op = findAncillaryOperation(t, "Interconnect", "createUsingPOST_1")
	fake.statusCode = op.SuccessStatus
	runAncillaryOperation(t, op, map[string]string{"body": `{"name":"probe","packageId":"pkg-1","typeId":"EXTERNAL","capacity":9007199254740993}`})
	body := fake.calls[0].body.(map[string]any)
	if got, ok := body["capacity"].(json.Number); !ok || got.String() != "9007199254740993" {
		t.Fatalf("capacity = %#v, want lossless json.Number", body["capacity"])
	}
}

func TestAncillaryBodyValidationIncludesComponentRequestBodies(t *testing.T) {
	installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Interconnect", "createUsingPOST_1")
	cmd := newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"body": `{"packageId":"pkg-1","typeId":"EXTERNAL"}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), `missing required field "name"`) {
		t.Fatalf("RunE() error = %v, want required field error", err)
	}

	op = findAncillaryOperation(t, "Virtual-ip-address", "createPrivateAddressPairUsingPOST")
	cmd = newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"virtual-ip-address-id": "vip-1", "body": `{}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), `internalNetworkInterfaceId`) {
		t.Fatalf("RunE() error = %v, want component request-body field error", err)
	}
}

func TestAncillaryBodyTargetMustMatchPath(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Persistent-volume", "deletePersistentVolumeUsingDELETE")
	cmd := newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"pv-id": "pv-1", "body": `{"persistentVolumeId":"pv-2"}`, "force": "true"})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --pv-id") {
		t.Fatalf("RunE() error = %v, want body target mismatch", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want no API call", fake.calls)
	}

	op = findAncillaryOperation(t, "Network-acl", "updateAssociatedSubnetsUsingPUT")
	cmd = newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"uuid": "acl-1", "body": `{"aclId":"acl-2"}`})
	if err := cmd.RunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "must match --uuid") {
		t.Fatalf("RunE() error = %v, want ACL target mismatch", err)
	}
}

func TestAncillaryDryRunIsOfflineAndUsesConfiguredProjectPlaceholder(t *testing.T) {
	clientCalls := 0
	previous := newAncillaryClient
	newAncillaryClient = func(*cobra.Command) (ancillaryAPI, *config.Config, error) {
		clientCalls++
		return &ancillaryRecordingAPI{}, &config.Config{ProjectID: "project-1"}, nil
	}
	t.Cleanup(func() { newAncillaryClient = previous })

	op := findAncillaryOperation(t, "Interconnect", "createUsingPOST_1")
	output := testutil.CaptureStdout(t, func() {
		cmd := newAncillaryOperationCommand(op)
		setAncillaryFlags(t, cmd, map[string]string{"body": `{"name":"probe","packageId":"pkg-1","typeId":"EXTERNAL","password":"do-not-print"}`, "dry-run": "true"})
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0", clientCalls)
	}
	if !strings.Contains(output, "<configured-project-id>") || strings.Contains(output, "do-not-print") || !strings.Contains(output, "REDACTED") {
		t.Fatalf("dry-run output = %q", output)
	}

	op = findAncillaryOperation(t, "Interconnect", "deleteUsingDELETE_1")
	output = testutil.CaptureStdout(t, func() {
		cmd := newAncillaryOperationCommand(op)
		setAncillaryFlags(t, cmd, map[string]string{"interconnect-id": "interconnect-1", "dry-run": "true"})
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})
	if clientCalls != 0 {
		t.Fatalf("client factory calls after delete dry-run = %d, want 0", clientCalls)
	}
	if !strings.Contains(output, configuredProjectPlaceholder) || strings.Contains(output, "[y/N]") {
		t.Fatalf("delete dry-run output = %q, want offline placeholder without confirmation", output)
	}
}

func TestAncillaryDestructiveNonInteractiveCommandDoesNotCallAPI(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })
	op := findAncillaryOperation(t, "Interconnect", "deleteUsingDELETE_1")
	cmd := newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, map[string]string{"interconnect-id": "interconnect-1"})
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected confirmation refusal")
	}
	if len(fake.calls) != 0 || cli.ConfirmationError() == nil {
		t.Fatalf("calls = %#v confirmation error = %v", fake.calls, cli.ConfirmationError())
	}
}

func TestAncillaryDestructiveConfirmationShowsResolvedProject(t *testing.T) {
	fake := installAncillaryRecordingClient(t)
	op := findAncillaryOperation(t, "Interconnect", "deleteUsingDELETE_1")
	output := testutil.CaptureStdout(t, func() {
		withAncillaryStdin(t, "n\n", func() {
			cmd := newAncillaryOperationCommand(op)
			setAncillaryFlags(t, cmd, map[string]string{"interconnect-id": "interconnect-1"})
			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatalf("RunE() error = %v", err)
			}
		})
	})
	if strings.Contains(output, configuredProjectPlaceholder) || !strings.Contains(output, "/v2/project-1/interconnects/interconnect-1") {
		t.Fatalf("confirmation output = %q, want resolved project path", output)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("calls = %#v, want declined confirmation to skip API call", fake.calls)
	}
}

func TestAncillaryResponseContractsPreserveExactStatusAndBodyShape(t *testing.T) {
	created := findAncillaryOperation(t, "Network-acl", "createNetworkAclUsingPOST")
	if err := ancillaryResponseError(created, client.HTTPResponse{StatusCode: http.StatusOK, Empty: false}); err == nil {
		t.Fatal("HTTP 200 accepted for documented HTTP 201 operation")
	}
	if err := ancillaryResponseError(created, client.HTTPResponse{StatusCode: http.StatusCreated, Empty: true}); err == nil {
		t.Fatal("empty HTTP 201 accepted for documented JSON response")
	}

	bodyless := findAncillaryOperation(t, "Route-table", "updateRouteTableDetailUsingPUT")
	if err := ancillaryResponseError(bodyless, client.HTTPResponse{StatusCode: http.StatusOK, Empty: true}); err != nil {
		t.Fatalf("bodyless HTTP 200 rejected: %v", err)
	}
	if err := ancillaryResponseError(bodyless, client.HTTPResponse{StatusCode: http.StatusOK, Empty: false}); err == nil {
		t.Fatal("unexpected HTTP 200 body accepted")
	}

	noContent := findAncillaryOperation(t, "Interconnect", "deleteUsingDELETE_1")
	if err := ancillaryResponseError(noContent, client.HTTPResponse{StatusCode: http.StatusNoContent, Empty: true}); err != nil {
		t.Fatalf("empty HTTP 204 rejected: %v", err)
	}
}

func installAncillaryRecordingClient(t *testing.T) *ancillaryRecordingAPI {
	t.Helper()
	fake := &ancillaryRecordingAPI{}
	previous := newAncillaryClient
	newAncillaryClient = func(*cobra.Command) (ancillaryAPI, *config.Config, error) {
		return fake, &config.Config{ProjectID: "project-1", Output: "json"}, nil
	}
	t.Cleanup(func() { newAncillaryClient = previous })
	return fake
}

func findAncillaryOperation(t *testing.T, tag, operationID string) ancillaryOperation {
	t.Helper()
	for _, op := range ancillaryOperations() {
		if op.Tag == tag && op.OperationID == operationID {
			return op
		}
	}
	t.Fatalf("ancillary operation %s/%s not found", tag, operationID)
	return ancillaryOperation{}
}

func runAncillaryOperation(t *testing.T, op ancillaryOperation, flags map[string]string) {
	t.Helper()
	cmd := newAncillaryOperationCommand(op)
	setAncillaryFlags(t, cmd, flags)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func setAncillaryFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
}

func withAncillaryStdin(t *testing.T, input string, run func()) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write.WriteString(input); err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	old := os.Stdin
	os.Stdin = read
	defer func() {
		os.Stdin = old
		_ = read.Close()
	}()
	run()
}
