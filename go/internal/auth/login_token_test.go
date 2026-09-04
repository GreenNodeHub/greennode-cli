package auth

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// tokenServerBuilder is a configurable /token test server. It counts hits and
// can serve a sequence of bodies (so a rotation test can return a fresh
// refresh_token on the first refresh and the same on the next).
type tokenServerBuilder struct {
	hits   atomic.Int32
	mu     sync.Mutex
	bodies []string // sequence; each refresh pops the head (or reuses the last)
	status int      // status to return (200 if 0)
	bearer string   // the Basic header the server received (for assertions)
}

// newTokenServer starts a /token test server. bodies is a sequence consumed one
// per request (the last is reused if more requests arrive); status overrides
// the response status (200 when 0).
func newTokenServer(t *testing.T, bodies []string, status int) (*httptest.Server, *tokenServerBuilder) {
	t.Helper()
	tb := &tokenServerBuilder{bodies: bodies, status: status}
	srv := httptest.NewServer(tb)
	t.Cleanup(srv.Close)
	return srv, tb
}

func (tb *tokenServerBuilder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tb.hits.Add(1)
	tb.mu.Lock()
	tb.bearer = r.Header.Get("Authorization")
	body := tb.bodies[0]
	if len(tb.bodies) > 1 {
		tb.bodies = tb.bodies[1:]
	}
	status := tb.status
	tb.mu.Unlock()

	if status != 0 && status != http.StatusOK {
		http.Error(w, `{"error":"invalid_grant"}`, status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func TestLoginTokenProvider_GetToken_RefreshesAndReturnsAccessToken(t *testing.T) {
	t.Parallel()
	body := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-orig","expires_in":3600}`
	srv, _ := newTokenServer(t, []string{body}, 0)

	var persisted []struct {
		rt  string
		exp time.Time
	}
	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, func(rt string, exp time.Time) error {
		persisted = append(persisted, struct {
			rt  string
			exp time.Time
		}{rt, exp})
		return nil
	})

	got, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got != "at-1" {
		t.Errorf("token=%q, want at-1", got)
	}
	// Response carried the SAME refresh_token we sent → no rotation → no persist.
	if len(persisted) != 0 {
		t.Errorf("persist called %d times for a non-rotating response, want 0", len(persisted))
	}
}

func TestLoginTokenProvider_GetToken_CacheHitSkipsHTTP(t *testing.T) {
	t.Parallel()
	body := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-orig","expires_in":3600}`
	srv, tb := newTokenServer(t, []string{body}, 0)

	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, nil)

	if _, err := p.GetToken(); err != nil {
		t.Fatalf("first GetToken: %v", err)
	}
	if _, err := p.GetToken(); err != nil {
		t.Fatalf("second GetToken: %v", err)
	}
	if got := tb.hits.Load(); got != 1 {
		t.Errorf("token endpoint hits=%d, want 1 (second GetToken served from cache)", got)
	}
}

func TestLoginTokenProvider_RefreshToken_ForcesNewHTTPCall(t *testing.T) {
	t.Parallel()
	body := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-orig","expires_in":3600}`
	srv, tb := newTokenServer(t, []string{body, body}, 0)

	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, nil)

	if _, err := p.GetToken(); err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if _, err := p.RefreshToken(); err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if got := tb.hits.Load(); got != 2 {
		t.Errorf("token endpoint hits=%d, want 2 (RefreshToken force-refreshes)", got)
	}
}

func TestLoginTokenProvider_Rotation_PersistsAndAdoptsNewRefreshToken(t *testing.T) {
	t.Parallel()
	// First refresh returns a NEW refresh_token (rotation); second returns the
	// same one (steady state).
	first := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-ROTATED","expires_in":3600}`
	second := `{"access_token":"at-2","token_type":"Bearer","refresh_token":"rt-ROTATED","expires_in":3600}`
	srv, tb := newTokenServer(t, []string{first, second}, 0)

	var persisted []string
	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, func(rt string, _ time.Time) error {
		persisted = append(persisted, rt)
		return nil
	})

	if _, err := p.GetToken(); err != nil {
		t.Fatalf("first GetToken: %v", err)
	}
	if len(persisted) != 1 || persisted[0] != "rt-ROTATED" {
		t.Fatalf("persist calls=%v, want [rt-ROTATED]", persisted)
	}

	// Force a second refresh: the response now carries the SAME refresh_token the
	// provider already holds → persist must NOT fire again.
	if _, err := p.RefreshToken(); err != nil {
		t.Fatalf("second RefreshToken: %v", err)
	}
	if len(persisted) != 1 {
		t.Errorf("persist fired again on a non-rotating response; calls=%v, want 1", persisted)
	}
	if tb.hits.Load() != 2 {
		t.Errorf("hits=%d, want 2", tb.hits.Load())
	}
}

func TestLoginTokenProvider_Non2xx_HardErrors(t *testing.T) {
	t.Parallel()
	srv, _ := newTokenServer(t, []string{`{"error":"invalid_grant"}`}, http.StatusBadRequest)

	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, nil)
	_, err := p.GetToken()
	if err == nil {
		t.Fatal("GetToken succeeded, want error")
	}
	if !errors.Is(err, ErrLoginTokenRefreshFailed) {
		t.Errorf("err=%v, want errors.Is ErrLoginTokenRefreshFailed", err)
	}
	if !strings.Contains(err.Error(), "grn login") {
		t.Errorf("err=%v, want guidance mentioning `grn login`", err)
	}
}

// TestLoginTokenProvider_Non2xx_DoesNotExposeIAMDetails: a refresh-grant
// failure (here the 500 {"errors":[]} IAM returns for an expired/revoked RT)
// must surface as just the re-login guidance — no status code, no raw body —
// so the user-facing error isn't noise that looks like a server incident.
func TestLoginTokenProvider_Non2xx_DoesNotExposeIAMDetails(t *testing.T) {
	t.Parallel()
	srv, _ := newTokenServer(t, []string{`{"errors":[]}`}, http.StatusInternalServerError)

	p := NewLoginTokenProvider("rt-orig", "cid", "", srv.URL, nil)
	_, err := p.GetToken()
	if !errors.Is(err, ErrLoginTokenRefreshFailed) {
		t.Errorf("err=%v, want ErrLoginTokenRefreshFailed", err)
	}
	for _, leak := range []string{"500", "errors", "status"} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("err=%q leaks IAM detail %q; want just the re-login guidance", err.Error(), leak)
		}
	}
}

// TestLoginTokenProvider_TransportError_JustRequestsLogin: a transport-level
// refresh failure (IAM unreachable) also surfaces as just the re-login
// guidance — no connection/dial details leaked.
func TestLoginTokenProvider_TransportError_JustRequestsLogin(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close() // tear down → the next dial is refused

	p := NewLoginTokenProvider("rt-orig", "cid", "", addr, nil)
	_, err := p.GetToken()
	if !errors.Is(err, ErrLoginTokenRefreshFailed) {
		t.Errorf("err=%v, want ErrLoginTokenRefreshFailed", err)
	}
	for _, leak := range []string{"connect", "refused", "dial", addr} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("err=%q leaks transport detail %q; want just the re-login guidance", err.Error(), leak)
		}
	}
}

func TestLoginTokenProvider_RefreshRequestShape(t *testing.T) {
	t.Parallel()
	body := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-orig","expires_in":3600}`
	srv, tb := newTokenServer(t, []string{body}, 0)

	// Send a non-empty secret to confirm it travels in the Basic header (a public
	// client sends Basic(client_id, "") — the login path).
	p := NewLoginTokenProvider("rt-orig", "cid-secret", "cs", srv.URL, nil)
	if _, err := p.GetToken(); err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	if !strings.HasPrefix(tb.bearer, "Basic ") {
		t.Fatalf("Authorization header=%q, want Basic ...", tb.bearer)
	}
	dec, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(tb.bearer, "Basic "))
	if err != nil {
		t.Fatalf("decode basic: %v", err)
	}
	// client_id is url.QueryEscape'd in the username (tokencx.go:104).
	wantUser := url.QueryEscape("cid-secret")
	if !strings.HasPrefix(string(dec), wantUser+":") {
		t.Errorf("basic user=%q, want prefix %q", string(dec), wantUser+":")
	}
}
