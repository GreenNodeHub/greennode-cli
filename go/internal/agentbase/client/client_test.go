package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
)

type stubResponse struct {
	Message string `json:"message"`
}

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(func() {
		srv.Close()
		tokenSrv.Close()
	})
	provider := auth.NewProvider("id", "secret", tokenSrv.URL)
	c := New(srv.URL, provider)
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

func TestAuthHeaderInjected(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
	})
	_ = c.Get(context.Background(), "/test", nil, nil)
}

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
