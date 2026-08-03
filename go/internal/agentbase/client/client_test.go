package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// fakeTokenProvider is a test stub satisfying coreclient.TokenProvider
// (GetToken/RefreshToken, ctx-less) — the same seam vks/vserver exercise. It
// proves client.Do drives the Bearer header from any TokenProvider
// implementor, not just the concrete auth providers.
type fakeTokenProvider struct {
	token    string
	err      error
	getCalls atomic.Int32
}

func (f *fakeTokenProvider) GetToken() (string, error) {
	f.getCalls.Add(1)
	return f.token, f.err
}

func (f *fakeTokenProvider) RefreshToken() (string, error) {
	return f.token, f.err
}

type stubResponse struct {
	Message string `json:"message"`
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, &fakeTokenProvider{token: "test-token"})
	return srv, c
}

func TestGetSuccess(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "ok"})
	})
	var out stubResponse
	if err := c.Get(context.Background(), "/test", nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Message != "ok" {
		t.Errorf("expected ok, got %s", out.Message)
	}
}

func TestGetWithQuery(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("expected page=2, got %s", r.URL.Query().Get("page"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "paged"})
	})
	q := url.Values{}
	q.Set("page", "2")
	var out stubResponse
	if err := c.Get(context.Background(), "/test", q, &out); err != nil {
		t.Fatal(err)
	}
}

func TestPostSuccess(t *testing.T) {
	type body struct{ Name string }
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var b body
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Name != "test" {
			t.Errorf("expected name=test, got %s", b.Name)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "created"})
	})
	var out stubResponse
	if err := c.Post(context.Background(), "/test", body{Name: "test"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Message != "created" {
		t.Errorf("expected created, got %s", out.Message)
	}
}

func TestDeleteSuccess(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Delete(context.Background(), "/test", nil); err != nil {
		t.Fatal(err)
	}
}

func TestAPIErrorReturned(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	err := c.Get(context.Background(), "/missing", nil, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

// TestAuthHeaderInjected: Do attaches `Bearer <token>` from the TokenProvider —
// the seam contract. Also asserts GetToken is called exactly once per request
// (so the provider drives auth, not a baked-in header).
func TestAuthHeaderInjected(t *testing.T) {
	fp := &fakeTokenProvider{token: "test-token"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, fp)
	if err := c.Get(context.Background(), "/test", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := fp.getCalls.Load(); got != 1 {
		t.Errorf("GetToken calls=%d, want 1 per request", got)
	}
}

// TestDo_GetTokenErrorSurfaces: a provider error (e.g. IAM down) surfaces from
// Do without hitting the network — no silent fallback.
func TestDo_GetTokenErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called when GetToken fails")
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, &fakeTokenProvider{err: errSentinel})
	if err := c.Get(context.Background(), "/test", nil, nil); err != errSentinel {
		t.Errorf("expected provider error to surface, got %v", err)
	}
}

var errSentinel = errors.New("sentinel provider error")

func TestPatchSuccess(t *testing.T) {
	type body struct{ Value int }
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "patched"})
	})
	var out stubResponse
	if err := c.Patch(context.Background(), "/test", nil, body{Value: 1}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestPutSuccess(t *testing.T) {
	type body struct{ Name string }
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "updated"})
	})
	var out stubResponse
	if err := c.Put(context.Background(), "/test", body{Name: "x"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Message != "updated" {
		t.Errorf("expected updated, got %s", out.Message)
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 422, Body: `{"detail":"invalid"}`}
	msg := e.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
	if msg != "API error (HTTP 422): {\"detail\":\"invalid\"}" {
		t.Errorf("unexpected error message: %s", msg)
	}
}

// TestDoWithHeaders_ExtraHeadersApplied: DoWithHeaders forwards caller-supplied
// headers (e.g. If-Match for OCC PUTs) while still applying the standard
// Authorization/Content-Type/Accept set. Reserved headers in the map are
// ignored so a caller cannot clobber auth.
func TestDoWithHeaders_ExtraHeadersApplied(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.Header.Get("If-Match"); got != `"v3"` {
			t.Errorf("expected If-Match=\"v3\", got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization not applied: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type not applied: %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept not applied: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stubResponse{Message: "ok"})
	})
	headers := map[string]string{
		"If-Match":      `"v3"`,
		"X-Custom":      "yes",
		"Authorization": "Bearer should-be-ignored", // reserved — must not clobber
	}
	var out stubResponse
	if err := c.DoWithHeaders(context.Background(), http.MethodPut, "/test", nil, headers, map[string]string{"k": "v"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Message != "ok" {
		t.Errorf("expected ok, got %s", out.Message)
	}
}

// TestDo_UnchangedByHeaderSeam: factoring Do through doReq must not change Do's
// behavior — no extra headers, standard set still applied.
func TestDo_UnchangedByHeaderSeam(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization not applied: %q", got)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}
