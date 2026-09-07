package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
)

func TestSetHeadersAppliesExtraHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))

	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, false).
		SetHeaders(map[string]string{"user-type": "IAM_USER"})

	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if v := got.Get("user-type"); v != "IAM_USER" {
		t.Errorf("user-type = %q, want IAM_USER", v)
	}
	// The extras must not displace the built-ins.
	if v := got.Get("Authorization"); v != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", v)
	}
	if v := got.Get("Content-Type"); v != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", v)
	}
}

func TestSetHeadersSkipsEmptyValues(t *testing.T) {
	// An unset --user-type must send no header at all, so the API applies its own
	// ROOT_USER default rather than us pinning a value.
	var present bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, present = r.Header["User-Type"]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))

	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, false).
		SetHeaders(map[string]string{"user-type": ""})

	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if present {
		t.Error("user-type header was sent despite an empty value")
	}
}

func TestSetHeadersSurviveTokenRefreshRetry(t *testing.T) {
	// The 401 path builds a second request from scratch; extras are easy to drop
	// there, which would silently change the billing flow on a retried call.
	var calls int
	var retryHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		retryHeader = r.Header.Get("user-type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	fp := &fakeProvider{token: "first", refreshTo: "second"}
	c := NewGreennodeClient(srv.URL, fp, 5*time.Second, 5*time.Second, false, false).
		SetHeaders(map[string]string{"user-type": "ROOT_USER"})

	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("server saw %d calls, want 2 (initial 401 + retry)", calls)
	}
	if retryHeader != "ROOT_USER" {
		t.Errorf("user-type on retry = %q, want ROOT_USER", retryHeader)
	}
}

func TestExistingCallersSendNoExtraHeaders(t *testing.T) {
	// A client that never calls SetHeaders must send exactly the three built-in
	// headers, unchanged — this is what keeps the vdb header seam invisible to
	// every other product.
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))

	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, false)
	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	// Accept-Encoding and Host are added by net/http itself, not by us.
	for _, h := range []string{"Authorization", "Content-Type", "User-Agent"} {
		if got.Get(h) == "" {
			t.Errorf("%s header missing", h)
		}
		got.Del(h)
	}
	got.Del("Accept-Encoding")
	if len(got) != 0 {
		t.Errorf("unexpected extra headers on a client without SetHeaders: %v", got)
	}
}
