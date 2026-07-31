package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// v2TokenHandler responds to a client_credentials token POST with a valid
// RFC 6749 token body and counts hits so cache/refresh behavior is observable.
func v2TokenHandler(t *testing.T, hits *atomic.Int32, body string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})
}

// TestMachineTokenProvider_MintsAndCaches: a cached token is reused until the
// expiry skew, so repeat GetToken calls hit the token server once. Mirrors the
// former TokenManager cache contract (now via clientcredentials).
func TestMachineTokenProvider_MintsAndCaches(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(v2TokenHandler(t, &hits,
		`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	t.Cleanup(srv.Close)

	p := NewMachineTokenProvider("cid", "cs", srv.URL)
	if _, err := p.GetToken(); err != nil {
		t.Fatalf("first GetToken: %v", err)
	}
	if _, err := p.GetToken(); err != nil {
		t.Fatalf("second GetToken (cache hit): %v", err)
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("token server hits=%d, want 1 (second GetToken should hit cache)", got)
	}
}

// TestMachineTokenProvider_RefreshTokenForceMints: RefreshToken re-mints even
// when the cache is valid — the 401-retry seam GreennodeClient uses.
func TestMachineTokenProvider_RefreshTokenForceMints(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(v2TokenHandler(t, &hits,
		`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	t.Cleanup(srv.Close)

	p := NewMachineTokenProvider("cid", "cs", srv.URL)
	if _, err := p.GetToken(); err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if _, err := p.RefreshToken(); err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if got := hits.Load(); got != 2 {
		t.Errorf("token server hits=%d, want 2 (RefreshToken must force re-mint)", got)
	}
}

// TestMachineTokenProvider_MintError: a non-2xx from IAM surfaces as an error
// (no silent fallback — machine mode has no alternate token source).
func TestMachineTokenProvider_MintError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	p := NewMachineTokenProvider("cid", "cs", srv.URL)
	if _, err := p.GetToken(); err == nil {
		t.Fatal("GetToken succeeded, want error on IAM 401")
	}
}

// TestMachineTokenProvider_SetTokenSeedBypassesIAM: the test-only SetToken seeds
// a captive token so fixtures (e.g. internal/client/client_test.go) never hit
// IAM. GetToken returns the seeded value and makes zero server calls.
func TestMachineTokenProvider_SetTokenSeedBypassesIAM(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(v2TokenHandler(t, &hits,
		`{"access_token":"should-not-be-used","token_type":"Bearer","expires_in":3600}`))
	t.Cleanup(srv.Close)

	p := NewMachineTokenProvider("cid", "cs", srv.URL)
	p.SetToken("captive-token", time.Now().Add(1*time.Hour))
	got, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got != "captive-token" {
		t.Errorf("GetToken=%q, want captive-token", got)
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("token server hits=%d, want 0 (SetToken seed must bypass IAM)", got)
	}
}
