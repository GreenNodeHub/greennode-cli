package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func newTestTokenServer(t *testing.T, token string, expiresIn int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
		})
	}))
}

func TestAccessTokenFetchSuccess(t *testing.T) {
	srv := newTestTokenServer(t, "test-token-123", 3600)
	defer srv.Close()

	p := NewProvider("client-id", "client-secret", srv.URL)
	tok, err := p.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "test-token-123" {
		t.Errorf("expected test-token-123, got %s", tok)
	}
}

func TestAccessTokenCached(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "cached-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	p := NewProvider("id", "secret", srv.URL)
	_, _ = p.AccessToken(context.Background())
	_, _ = p.AccessToken(context.Background())
	_, _ = p.AccessToken(context.Background())

	if callCount != 1 {
		t.Errorf("expected 1 token fetch (cached), got %d", callCount)
	}
}

func TestAccessTokenRefreshWhenExpired(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "refreshed-token",
			TokenType:   "Bearer",
			ExpiresIn:   1, // 1 second — will be immediately expired for the test
		})
	}))
	defer srv.Close()

	p := NewProvider("id", "secret", srv.URL)
	_, _ = p.AccessToken(context.Background())

	// Force expiry.
	p.cached.Expiry = time.Now().Add(-1 * time.Minute)

	_, _ = p.AccessToken(context.Background())
	if callCount != 2 {
		t.Errorf("expected 2 token fetches (expired cache), got %d", callCount)
	}
}

func TestAccessTokenServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := NewProvider("id", "secret", srv.URL)
	_, err := p.AccessToken(context.Background())
	if err == nil {
		t.Error("expected error from server error response")
	}
}

func TestTokenSource(t *testing.T) {
	srv := newTestTokenServer(t, "ts-token", 3600)
	defer srv.Close()

	p := NewProvider("id", "secret", srv.URL)
	ts := p.TokenSource(context.Background())
	if ts == nil {
		t.Fatal("expected non-nil TokenSource")
	}
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("TokenSource.Token() error: %v", err)
	}
	if tok.AccessToken != "ts-token" {
		t.Errorf("expected ts-token, got %s", tok.AccessToken)
	}
}

func TestTokenIsExpired(t *testing.T) {
	expired := &Token{Expiry: time.Now().Add(-1 * time.Hour)}
	if !expired.IsExpired() {
		t.Error("expected expired token to be expired")
	}

	fresh := &Token{Expiry: time.Now().Add(1 * time.Hour)}
	if fresh.IsExpired() {
		t.Error("expected fresh token to not be expired")
	}

	// Within 30s buffer
	almostExpired := &Token{Expiry: time.Now().Add(15 * time.Second)}
	if !almostExpired.IsExpired() {
		t.Error("expected token expiring within 30s to be considered expired")
	}
}
