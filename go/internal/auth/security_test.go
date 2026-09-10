package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fixtureProvider(mode, endpoint string) contextualProvider {
	if mode == "user" {
		return NewLoginTokenProvider("fixture-refresh", "fixture-client", "", endpoint, nil)
	}
	return NewMachineTokenProvider("fixture-client", "fixture-secret", endpoint)
}

func TestTokenProvidersRejectRedirects(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			for _, resetTimeout := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/reset=%t", mode, status, resetTimeout), func(t *testing.T) {
					var origins, destinations atomic.Int32
					destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						destinations.Add(1)
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"access_token":"fixture-access","expires_in":3600}`)
					}))
					defer destination.Close()
					origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						origins.Add(1)
						if r.Method != http.MethodPost {
							t.Error("token grant must use POST")
						}
						id, secret, ok := r.BasicAuth()
						wantSecret := "fixture-secret"
						if mode == "user" {
							wantSecret = ""
						}
						if !ok || id != "fixture-client" || secret != wantSecret {
							t.Error("token grant changed Basic authentication")
						}
						if err := r.ParseForm(); err != nil {
							t.Error(err)
						}
						wantGrant := "client_credentials"
						if mode == "user" {
							wantGrant = "refresh_token"
							if r.Form.Get("refresh_token") != "fixture-refresh" {
								t.Error("refresh grant changed")
							}
						}
						if r.Form.Get("grant_type") != wantGrant || r.Form.Get("client_secret") != "" {
							t.Error("grant or credential placement changed")
						}
						http.Redirect(w, r, destination.URL, status)
					}))
					defer origin.Close()
					provider := fixtureProvider(mode, origin.URL)
					if resetTimeout {
						provider.SetHTTPTimeout(time.Second)
					}
					if _, err := provider.GetToken(); err == nil {
						t.Error("redirected token grant succeeded")
					}
					if origins.Load() != 1 || destinations.Load() != 0 {
						t.Fatalf("requests: origin=%d destination=%d", origins.Load(), destinations.Load())
					}
				})
			}
		}
	}
}

func TestTokenProviderErrorsSuppressResponseDetails(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		for _, payload := range []string{`{"error":"fixture-secret","error_description":"fixture-secret","error_uri":"fixture-secret"}`, `{"access_token":"fixture-secret"}`, "fixture-secret"} {
			t.Run(mode+"/"+payload, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = io.WriteString(w, payload)
				}))
				defer srv.Close()
				_, err := fixtureProvider(mode, srv.URL).GetToken()
				if err == nil || strings.Contains(err.Error(), "fixture-secret") {
					t.Fatalf("unsafe token error: %v", err)
				}
				if calls.Load() != 1 {
					t.Fatalf("token grant repeated %d times", calls.Load())
				}
			})
		}
	}
}

func TestTokenProvidersPreserveContextErrors(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		provider := fixtureProvider(mode, "http://127.0.0.1:1/fixture-token")
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		provider.SetBaseContext(ctx)
		_, err := provider.GetToken()
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("%s deadline error: %v", mode, err)
		}
	}
}

func TestTokenProvidersSuppressMalformedSuccessDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"fixture-access","expires_in":1e9999}`)
	}))
	defer srv.Close()
	for _, mode := range []string{"machine", "user"} {
		_, err := fixtureProvider(mode, srv.URL).GetToken()
		if err == nil || strings.Contains(err.Error(), "1e9999") || strings.Contains(err.Error(), "fixture-access") {
			t.Fatalf("%s malformed response disclosed: %v", mode, err)
		}
	}
}

func TestRotationPersistenceWarningSuppressesCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"fixture-access","refresh_token":"fixture-rotated","expires_in":3600}`)
	}))
	defer srv.Close()
	provider := NewLoginTokenProvider("fixture-refresh", "fixture-client", "", srv.URL, func(token string, _ time.Time) error {
		return fmt.Errorf("cannot persist %s", token)
	})
	previous := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer func() { os.Stderr = previous }()
	os.Stderr = writer
	token, tokenErr := provider.GetToken()
	_ = writer.Close()
	os.Stderr = previous
	warning, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if tokenErr != nil || token != "fixture-access" || provider.refreshToken != "fixture-rotated" {
		t.Fatalf("rotation state changed: %v", tokenErr)
	}
	if !strings.Contains(string(warning), "failed to persist") || strings.Contains(string(warning), "fixture-rotated") {
		t.Fatalf("unsafe persistence warning: %s", warning)
	}
}
