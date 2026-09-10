package vmonitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func TestMetricAPIKeyResponsesRedactUnlessRequested(t *testing.T) {
	for _, use := range []string{"create-metric", "list-metric"} {
		t.Run(use, func(t *testing.T) {
			op := findOperation(t, "api-key", use)
			if !op.SecretResponse {
				t.Fatalf("%s must be marked SecretResponse", use)
			}

			run := func(showSecret bool) (string, *recordingAPI) {
				fake := &recordingAPI{result: map[string]any{"name": "deploy", "key": "fixture-api-key"}}
				previous := newClient
				t.Cleanup(func() { newClient = previous })
				newClient = func(*cobra.Command) (vmonitorAPI, error) { return fake, nil }

				flags := validOperationFlags(t, op)
				if showSecret {
					flags["show-secret"] = "true"
				}
				return testutil.CaptureStdout(t, func() { runOperation(t, op, flags) }), fake
			}

			redacted, fake := run(false)
			if strings.Contains(redacted, "fixture-api-key") {
				t.Errorf("default output leaked the key: %q", redacted)
			}
			if !strings.Contains(redacted, client.RedactedValue) {
				t.Errorf("default output = %q, want a redaction marker", redacted)
			}
			if fake.sensitiveCalls != 1 {
				t.Errorf("sensitive client calls = %d, want 1", fake.sensitiveCalls)
			}
			if shown, _ := run(true); !strings.Contains(shown, "fixture-api-key") {
				t.Errorf("--show-secret output = %q, want the key", shown)
			}
		})
	}
}

func TestDeleteMetricAPIKeyNeverPrintsTheKeyInItsPath(t *testing.T) {
	op := findOperation(t, "api-key", "delete-metric")
	if len(op.SecretPathParams) != 1 || op.SecretPathParams[0] != "key" {
		t.Fatalf("SecretPathParams = %#v, want [key]", op.SecretPathParams)
	}

	flags := validOperationFlags(t, op)
	flags["key"] = "fixture-api-key"
	flags["dry-run"] = "true"
	preview := testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
	if strings.Contains(preview, "fixture-api-key") {
		t.Errorf("dry-run preview leaked the key: %q", preview)
	}
	if !strings.Contains(preview, client.RedactedValue) {
		t.Errorf("dry-run preview = %q, want a redaction marker", preview)
	}
}

func TestDeleteMetricConfirmationPromptMasksTheKey(t *testing.T) {
	op := findOperation(t, "api-key", "delete-metric")
	fake := &recordingAPI{}
	previous := newClient
	t.Cleanup(func() { newClient = previous })
	newClient = func(*cobra.Command) (vmonitorAPI, error) { return fake, nil }

	flags := validOperationFlags(t, op)
	flags["key"] = "fixture-api-key"
	prompt := testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
	if strings.Contains(prompt, "fixture-api-key") {
		t.Errorf("confirmation prompt leaked the key: %q", prompt)
	}
	if !strings.Contains(prompt, client.RedactedValue) {
		t.Errorf("confirmation prompt = %q, want a redaction marker", prompt)
	}
}

func TestDeleteMetricDebugLoggingMasksTheKeyInTheURL(t *testing.T) {
	const secret = "fixture-api-key"
	var logged string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	tokens := fixtureTokenProvider{}
	real := client.NewGreennodeClient(server.URL, tokens, 0, time.Second, true, true)

	previous := newClient
	t.Cleanup(func() { newClient = previous })
	newClient = func(*cobra.Command) (vmonitorAPI, error) { return real, nil }

	op := findOperation(t, "api-key", "delete-metric")
	flags := validOperationFlags(t, op)
	flags["key"] = secret
	flags["force"] = "true"
	logged = testutil.CaptureStderr(t, func() {
		_ = testutil.CaptureStdout(t, func() { runOperation(t, op, flags) })
	})
	if strings.Contains(logged, secret) {
		t.Errorf("--debug leaked the key in the request URL: %q", logged)
	}
	if !strings.Contains(logged, client.RedactedValue) {
		t.Errorf("--debug output = %q, want the masked URL", logged)
	}
}

type fixtureTokenProvider struct{}

func (fixtureTokenProvider) GetToken() (string, error)     { return "fixture-token", nil }
func (fixtureTokenProvider) RefreshToken() (string, error) { return "fixture-token", nil }
