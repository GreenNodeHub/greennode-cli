package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeProvider is a TokenProvider that is NOT *auth.MachineTokenProvider — it proves the
// GreennodeClient seam accepts any implementor and that GetToken/RefreshToken
// drive the Bearer header (and the 401 retry) as the contract expects.
type fakeProvider struct {
	token      string
	getCalls   atomic.Int32
	refreshCnt atomic.Int32
	refreshTo  string // token adopted on RefreshToken; "" → keep current
}

func (f *fakeProvider) GetToken() (string, error) {
	f.getCalls.Add(1)
	return f.token, nil
}

func (f *fakeProvider) RefreshToken() (string, error) {
	f.refreshCnt.Add(1)
	if f.refreshTo != "" {
		f.token = f.refreshTo
	}
	return f.token, nil
}

func TestGreennodeClient_UsesTokenProviderBearer(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	fp := &fakeProvider{token: "fake-bearer"}
	c := NewGreennodeClient(srv.URL, fp, 5*time.Second, 5*time.Second, false, false)

	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer fake-bearer" {
		t.Errorf("Authorization=%q, want Bearer fake-bearer", gotAuth)
	}
	if fp.getCalls.Load() != 1 {
		t.Errorf("GetToken calls=%d, want 1", fp.getCalls.Load())
	}
}

func TestGreennodeClient_401CallsRefreshTokenAndRetries(t *testing.T) {
	t.Parallel()
	var firstAuth, retryAuth string
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			firstAuth = r.Header.Get("Authorization")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		retryAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	// RefreshToken adopts a NEW token so the retry carries a different Bearer.
	fp := &fakeProvider{token: "stale", refreshTo: "refreshed"}
	c := NewGreennodeClient(srv.URL, fp, 5*time.Second, 5*time.Second, false, false)

	if _, err := c.Get("/v1/thing", nil); err != nil {
		t.Fatalf("Get after 401-retry: %v", err)
	}
	if firstAuth != "Bearer stale" {
		t.Errorf("first attempt Authorization=%q, want Bearer stale", firstAuth)
	}
	if retryAuth != "Bearer refreshed" {
		t.Errorf("retry Authorization=%q, want Bearer refreshed (after RefreshToken)", retryAuth)
	}
	if fp.refreshCnt.Load() != 1 {
		t.Errorf("RefreshToken calls=%d, want 1", fp.refreshCnt.Load())
	}
}
