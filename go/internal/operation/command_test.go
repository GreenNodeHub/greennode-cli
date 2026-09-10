package operation

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

type fakeAPI struct{}

func runTestCommand(t *testing.T, cmd *cobra.Command, flags map[string]string) error {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s=%s: %v", name, value, err)
		}
	}
	return cmd.RunE(cmd, nil)
}

func TestDryRunNeverConstructsClientOrCallsAPI(t *testing.T) {
	clientCalls := 0
	apiCalls := 0
	spec := Spec[*fakeAPI]{
		ServiceName: "TestSvc",
		NewClient: func(*cobra.Command, Descriptor) (*fakeAPI, error) {
			clientCalls++
			return &fakeAPI{}, nil
		},
		Execute: func(*cobra.Command, *fakeAPI, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			apiCalls++
			return client.HTTPResponse{StatusCode: 200, Data: map[string]any{"ok": true}}, nil
		},
	}
	d := Descriptor{
		Use: "create", Short: "test", Method: "POST", Path: "/v1/things",
		Body:     &BodyContract{Kind: ObjectBody, Usage: "Request body as a JSON object"},
		Mutation: true,
	}
	cmd := NewCommand(spec, d)

	output := testutil.CaptureStdout(t, func() {
		if err := runTestCommand(t, cmd, map[string]string{"body": `{}`, "dry-run": "true"}); err != nil {
			t.Fatalf("RunE() error = %v, want a clean dry-run", err)
		}
	})

	if clientCalls != 0 {
		t.Fatalf("client factory calls = %d, want 0 on the dry-run path", clientCalls)
	}
	if apiCalls != 0 {
		t.Fatalf("API calls = %d, want 0 on the dry-run path", apiCalls)
	}
	if output == "" {
		t.Fatal("dry-run printed no preview")
	}
}

func TestDryRunMasksQueryAndBodyWithoutChangingWireValues(t *testing.T) {
	query := map[string]string{"api_key": "query-credential", "page": "0"}
	body := map[string]any{"password": "body-credential", "name": "fixture"}
	fields := DefaultDryRunFields(Descriptor{Method: "POST"}, "/fixture", query, body)
	output := testutil.CaptureStdout(t, func() { cli.PrintDryRun("create", "fixture", fields) })
	for _, secret := range []string{"query-credential", "body-credential"} {
		if strings.Contains(output, secret) {
			t.Fatalf("dry-run leaked %s", secret)
		}
	}
	if !strings.Contains(output, "[REDACTED]") || !strings.Contains(output, "fixture") {
		t.Fatal("preview lost masked or non-sensitive fields")
	}
	if query["api_key"] != "query-credential" || body["password"] != "body-credential" {
		t.Fatal("redaction modified transmitted values")
	}
}

func TestDryRunNeverInvokesRunDownloadOrWritesAFile(t *testing.T) {
	outputFile := t.TempDir() + "/out.zip"
	spec := Spec[*fakeAPI]{
		ServiceName: "TestSvc",
		NewClient: func(*cobra.Command, Descriptor) (*fakeAPI, error) {
			t.Fatal("NewClient invoked on the dry-run path")
			return nil, nil
		},
		Execute: func(*cobra.Command, *fakeAPI, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			t.Fatal("Execute invoked on the dry-run path")
			return client.HTTPResponse{}, nil
		},
		RunDownload: func(*cobra.Command, Descriptor, string, map[string]string) error {
			t.Fatal("RunDownload invoked on the dry-run path")
			return nil
		},
	}
	d := Descriptor{Use: "download", Short: "test", Method: "GET", Path: "/v1/certificate", Download: true}
	cmd := NewCommand(spec, d)

	testutil.CaptureStdout(t, func() {
		err := runTestCommand(t, cmd, map[string]string{"output-file": outputFile, "dry-run": "true"})
		if err != nil {
			t.Fatalf("RunE() error = %v, want a clean dry-run", err)
		}
	})

	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote %s: stat error = %v", outputFile, err)
	}
}

func TestDestructiveConfirmNonInteractiveRefusalFailsWithoutAPICall(t *testing.T) {
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })

	apiCalls := 0
	spec := Spec[*fakeAPI]{
		ServiceName: "TestSvc",
		NewClient: func(*cobra.Command, Descriptor) (*fakeAPI, error) {
			return &fakeAPI{}, nil
		},
		Execute: func(*cobra.Command, *fakeAPI, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			apiCalls++
			return client.HTTPResponse{StatusCode: 200, Data: map[string]any{"ok": true}}, nil
		},
	}
	d := Descriptor{Use: "delete", Short: "test", Method: "DELETE", Path: "/v1/things/{id}",
		Paths:       []PathParam{{Placeholder: "id", Flag: "id", Usage: "ID"}},
		Mutation:    true,
		Destructive: true,
	}
	cmd := NewCommand(spec, d)

	err := runTestCommand(t, cmd, map[string]string{"id": "thing-1"})
	if err == nil || err != cli.ConfirmationError() {
		t.Fatalf("RunE() error = %v, want the non-interactive confirmation failure", err)
	}
	if apiCalls != 0 {
		t.Fatalf("API calls = %d, want 0 when confirmation is refused", apiCalls)
	}
	if cli.ConfirmationError() == nil {
		t.Fatal("cli.ConfirmationError() = nil, want a non-interactive refusal recorded")
	}
}

func TestDestructiveConfirmForceSkipsPromptAndCallsAPI(t *testing.T) {
	cli.SetNonInteractive(true)
	t.Cleanup(func() { cli.SetNonInteractive(false) })

	apiCalls := 0
	spec := Spec[*fakeAPI]{
		ServiceName: "TestSvc",
		NewClient: func(*cobra.Command, Descriptor) (*fakeAPI, error) {
			return &fakeAPI{}, nil
		},
		Execute: func(*cobra.Command, *fakeAPI, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			apiCalls++
			return client.HTTPResponse{StatusCode: 200, Data: map[string]any{"ok": true}}, nil
		},
	}
	d := Descriptor{Use: "delete", Short: "test", Method: "DELETE", Path: "/v1/things/{id}",
		Paths:       []PathParam{{Placeholder: "id", Flag: "id", Usage: "ID"}},
		Mutation:    true,
		Destructive: true,
	}
	cmd := NewCommand(spec, d)

	if err := runTestCommand(t, cmd, map[string]string{"id": "thing-1", "force": "true"}); err != nil {
		t.Fatalf("RunE() error = %v, want --force to bypass confirmation cleanly", err)
	}
	if apiCalls != 1 {
		t.Fatalf("API calls = %d, want exactly 1 with --force", apiCalls)
	}
}

func TestEmptyResponseErrorAllowsOnlyListedStatuses(t *testing.T) {
	d := Descriptor{Method: "DELETE", Path: "/v1/things/{id}", EmptySuccess: []int{204}}

	if err := EmptyResponseError("TestSvc", d, client.HTTPResponse{StatusCode: 204, Empty: true}); err != nil {
		t.Fatalf("EmptyResponseError() = %v, want the documented empty 204 accepted", err)
	}
	err := EmptyResponseError("TestSvc", d, client.HTTPResponse{StatusCode: 200, Empty: true})
	if err == nil {
		t.Fatal("EmptyResponseError() = nil, want an undocumented empty response rejected")
	}
	want := "TestSvc API returned an empty HTTP 200 response for DELETE /v1/things/{id}; expected JSON"
	if err.Error() != want {
		t.Fatalf("EmptyResponseError() error = %q, want %q", err.Error(), want)
	}
	if err := EmptyResponseError("TestSvc", d, client.HTTPResponse{StatusCode: 200, Empty: false, Data: map[string]any{}}); err != nil {
		t.Fatalf("EmptyResponseError() = %v, want a non-empty response always accepted", err)
	}
}

func TestCombinedDestructiveAndDownloadRegistersForceOnce(t *testing.T) {
	spec := Spec[*fakeAPI]{
		ServiceName: "TestSvc",
		NewClient:   func(*cobra.Command, Descriptor) (*fakeAPI, error) { return &fakeAPI{}, nil },
		Execute: func(*cobra.Command, *fakeAPI, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			return client.HTTPResponse{}, nil
		},
		RunDownload: func(*cobra.Command, Descriptor, string, map[string]string) error { return nil },
	}
	d := Descriptor{Use: "delete", Short: "test", Method: "DELETE", Path: "/v1/things",
		Mutation: true, Destructive: true, Download: true,
	}

	var cmd *cobra.Command
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewCommand panicked registering --force twice: %v", r)
			}
		}()
		cmd = NewCommand(spec, d)
	}()

	if cmd.Flags().Lookup("force") == nil {
		t.Fatal("expected a single --force flag to be registered")
	}
}

func TestBodyRequirednessThroughCobra(t *testing.T) {
	for _, kind := range []BodyKind{ObjectBody, ArrayBody, OptionalObjectBody, OptionalJSONBody} {
		cmd := NewCommand(Spec[*fakeAPI]{ServiceName: "test"}, Descriptor{Use: "create", Method: "POST", Path: "/things", Mutation: true, Body: &BodyContract{Kind: kind}})
		cmd.SetArgs([]string{"--dry-run"})
		testutil.CaptureStdout(t, func() {
			err := cmd.Execute()
			optional := kind == OptionalObjectBody || kind == OptionalJSONBody
			if (err == nil) != optional {
				t.Errorf("body kind %d: error = %v", kind, err)
			}
		})
	}
}

func TestSecretPathParameterFailsClosedWhenClientCannotMask(t *testing.T) {
	// Deliberately implements nothing: the point is a client without MaskPathValues.
	type plainClient struct{}
	spec := Spec[*plainClient]{
		ServiceName: "Test",
		NewClient:   func(*cobra.Command, Descriptor) (*plainClient, error) { return &plainClient{}, nil },
		Execute: func(*cobra.Command, *plainClient, Descriptor, string, map[string]string, any, any) (client.HTTPResponse, error) {
			t.Fatal("execute must not run when the client cannot mask a secret path parameter")
			return client.HTTPResponse{}, nil
		},
	}
	d := Descriptor{
		Use:    "delete-thing",
		Method: http.MethodDelete,
		Path:   "/v1/things/{key}",
		Paths:  []PathParam{{Placeholder: "key", Flag: "key", Usage: "Key", Secret: true, Validate: func(string, string) error { return nil }}},
	}

	cmd := NewCommand(spec, d)
	if err := cmd.Flags().Set("key", "live-secret"); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot mask them in debug output") {
		t.Fatalf("RunE() error = %v, want a fail-closed masking error", err)
	}
	if strings.Contains(err.Error(), "live-secret") {
		t.Errorf("error message leaked the secret: %v", err)
	}
}
