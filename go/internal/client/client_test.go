package client

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vngcloud/greennode-cli/internal/auth"
)

func TestNewGreennodeClientWiresConnectTimeout(t *testing.T) {
	tm := auth.NewTokenManager("id", "secret")
	c := NewGreennodeClient("https://example.invalid", tm, 2*time.Second, 7*time.Second, true, false)

	if c.httpClient.Timeout != 7*time.Second {
		t.Errorf("read timeout: httpClient.Timeout = %v, want 7s", c.httpClient.Timeout)
	}
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, want *http.Transport", c.httpClient.Transport)
	}
	if tr.DialContext == nil {
		t.Error("connect timeout not wired: Transport.DialContext is nil")
	}
	if tr.TLSHandshakeTimeout != 2*time.Second {
		t.Errorf("connect timeout: TLSHandshakeTimeout = %v, want 2s", tr.TLSHandshakeTimeout)
	}
}

func TestPatchSendsPatchMethodAndBody(t *testing.T) {
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewTokenManager("id", "secret")
	// Pre-seed a static token so GetToken never calls the real IAM endpoint.
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))

	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, false)

	_, err := c.Patch("/v1/thing", map[string]interface{}{"enableAutoHealing": true})
	if err != nil {
		t.Fatalf("Patch returned error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotBody != `{"enableAutoHealing":true}` {
		t.Errorf("body = %q, want enableAutoHealing payload", gotBody)
	}
}

func TestFormatErrorSurfacesNestedErrorObject(t *testing.T) {
	// VKS returns errors as {"error": {"message": ...}} — a nested object, not a
	// string. The detail must still reach the user instead of being dropped.
	body := []byte(`{"error":{"message":"KubeConfig can only be requested when the cluster is ACTIVE."}}`)
	got := formatError(http.StatusBadRequest, body)
	if !strings.Contains(got, "cluster is ACTIVE") {
		t.Errorf("formatError = %q, want it to contain the nested error message", got)
	}
}

func TestFormatErrorUsesPlainStringMessage(t *testing.T) {
	body := []byte(`{"message":"boom"}`)
	got := formatError(http.StatusBadRequest, body)
	if !strings.Contains(got, "boom") {
		t.Errorf("formatError = %q, want it to contain %q", got, "boom")
	}
}

func TestGetReturnsTypedAPIErrorOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	defer srv.Close()

	tm := auth.NewTokenManager("id", "secret")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))
	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, false)

	_, err := c.Get("/v1/clusters/x", nil)
	if err == nil {
		t.Fatalf("expected an error for 404")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is not *APIError: %T (%v)", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "not found") {
		t.Errorf("Error() = %q, want it to contain the server message", apiErr.Error())
	}
}
