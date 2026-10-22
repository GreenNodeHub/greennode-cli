package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMutationsNeverReplay(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, status := range []int{401, 429, 503, 307, 308} {
			t.Run(fmt.Sprintf("%s/%d", method, status), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", "/fixture-redirect")
					w.WriteHeader(status)
				}))
				defer server.Close()
				provider := &fakeTokenProvider{token: "fixture-token"}
				c := New(server.URL, provider)
				if err := c.Do(context.Background(), method, "/fixture", nil, map[string]string{"name": "fixture"}, nil); err == nil {
					t.Fatal("accepted failed mutation")
				}
				if calls.Load() != 1 || provider.refreshCalls.Load() != 0 {
					t.Fatal("mutation replayed or refreshed")
				}
			})
		}
	}
}

func TestCredentialErrorsAreAlwaysMasked(t *testing.T) {
	for _, body := range []string{`{"accessToken":"fixture-secret"}`, `"fixture-secret"`, `{"message":"fixture-secret"}`, "fixture-secret"} {
		_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403); fmt.Fprint(w, body) })
		err := c.Get(context.Background(), "/fixture", nil, nil)
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Body != body || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal("unsafe error or lost raw error data")
		}
	}
}

func TestGetOnceNeverRefreshes(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(401)
	}))
	defer server.Close()
	provider := &fakeTokenProvider{token: "fixture-token"}
	c := New(server.URL, provider)
	if err := c.GetOnce(context.Background(), "/fixture", nil, nil); err == nil || calls.Load() != 1 || provider.refreshCalls.Load() != 0 {
		t.Fatal("credential read replayed")
	}
}

type failingTransport func(*http.Request) (*http.Response, error)

func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTransportFailureDoesNotReplayOrExposeURL(t *testing.T) {
	var calls int
	c := New("https://fixture.example", &fakeTokenProvider{token: "fixture-token"})
	cause := errors.New("fixture-secret")
	c.httpClient.Transport = failingTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })
	err := c.Post(context.Background(), "/fixture?token=fixture-secret", nil, nil)
	if calls != 1 || !errors.Is(err, cause) || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("unsafe transport error")
	}
}

func TestAuthErrorsAreMasked(t *testing.T) {
	cause := errors.New("fixture-secret")
	c := New("https://fixture.example", &fakeTokenProvider{err: cause})
	err := c.Get(context.Background(), "/fixture", nil, nil)
	if !errors.Is(err, cause) || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("unsafe authentication error")
	}
}

func TestReservedHeadersAreCaseInsensitive(t *testing.T) {
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
			t.Error("reserved header overwritten")
		}
	})
	if err := c.DoWithHeaders(context.Background(), "POST", "/fixture", nil, map[string]string{"authorization": "fixture-other", "content-type": "text/plain", "accept": "text/plain"}, map[string]string{}, nil); err != nil {
		t.Fatal(err)
	}
}
