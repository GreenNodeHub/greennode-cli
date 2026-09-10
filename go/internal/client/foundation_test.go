package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/testutil"
)

func TestRequestWithStatusPreservesEmptySuccessfulResponseMetadata(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		body       string
	}{
		{name: "empty ok", method: http.MethodGet, statusCode: http.StatusOK},
		{name: "whitespace ok", method: http.MethodGet, statusCode: http.StatusOK, body: "  \n"},
		{name: "empty created", method: http.MethodPost, statusCode: http.StatusCreated},
		{name: "empty no content", method: http.MethodDelete, statusCode: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			tm := auth.NewMachineTokenProvider("id", "secret", "")
			tm.SetToken("test-token", time.Now().Add(time.Hour))
			c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

			response, err := c.RequestWithStatus(tt.method, "/v1/thing", nil, nil)
			if err != nil {
				t.Fatalf("RequestWithStatus() error = %v", err)
			}
			if response.StatusCode != tt.statusCode || !response.Empty {
				t.Fatalf("response = %#v, want empty HTTP %d", response, tt.statusCode)
			}
			if data, ok := response.Data.(map[string]any); !ok || len(data) != 0 {
				t.Fatalf("response data = %#v, want backward-compatible empty map", response.Data)
			}
		})
	}
}

func TestMethodSpecificHelperKeepsEmptyMapBehavior(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	result, err := c.Get("/v1/thing", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if data, ok := result.(map[string]any); !ok || len(data) != 0 {
		t.Fatalf("Get() = %#v, want empty map", result)
	}
}

func TestSetHeaderAddsServiceSpecificHeader(t *testing.T) {
	var gotPortalUserID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPortalUserID = r.Header.Get("portal-user-id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)
	c.SetHeader("portal-user-id", "12345")

	if _, err := c.Get("/v2/project/servers", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotPortalUserID != "12345" {
		t.Errorf("portal-user-id = %q, want %q", gotPortalUserID, "12345")
	}
}

func TestPostDoesNotRetryServerError(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	_, err := c.Post("/v2/project/servers", map[string]any{"name": "server-a"})
	if err == nil {
		t.Fatal("Post returned nil error for HTTP 500")
	}
	if requests != 1 {
		t.Errorf("POST request count = %d, want 1 (writes must not be retried)", requests)
	}
}

func TestRequestWithStatusNoRetryDoesNotReplayGET(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	_, err := c.RequestWithStatusNoRetry(http.MethodGet, "/v1/unsafe-get", nil, nil)
	if err == nil {
		t.Fatal("RequestWithStatusNoRetry returned nil error for HTTP 500")
	}
	if requests != 1 {
		t.Errorf("GET request count = %d, want 1 (unsafe GET must not be retried)", requests)
	}
}

func TestRequestWithStatusNoRetryReportsUnknownOutcomeAfterTransportFailure(t *testing.T) {
	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient("https://example.invalid", tm, time.Second, time.Second, false, false)
	c.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection dropped")
	})

	_, err := c.RequestWithStatusNoRetry(http.MethodGet, "/v1/unsafe-get", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "outcome may be unknown") || !strings.Contains(err.Error(), "not retried") {
		t.Fatalf("RequestWithStatusNoRetry() error = %v, want unknown-outcome no-retry warning", err)
	}
}

func TestRequestWithStatusNoRetrySensitiveRedactsDebugResponse(t *testing.T) {
	const secret = "rotated-secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`"` + secret + `"`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, true)

	debugOutput := testutil.CaptureStderr(t, func() {
		response, err := c.RequestWithStatusNoRetrySensitive(http.MethodGet, "/v1/user/user-123/refresh", nil, nil)
		if err != nil {
			t.Fatalf("RequestWithStatusNoRetrySensitive() error = %v", err)
		}
		if response.Data != secret {
			t.Fatalf("response data = %#v, want secret returned to caller", response.Data)
		}
	})
	if strings.Contains(debugOutput, secret) || !strings.Contains(debugOutput, RedactedValue) {
		t.Fatalf("debug output = %q, want a redacted response", debugOutput)
	}
}

func TestRequestWithStatusNoRetryRawAcceptsStringResponse(t *testing.T) {
	for _, body := range []string{"updated", `"updated"`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			tm := auth.NewMachineTokenProvider("id", "secret", "")
			tm.SetToken("test-token", time.Now().Add(time.Hour))
			c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

			response, err := c.RequestWithStatusNoRetryRaw(http.MethodPut, "/v1/thing", nil, nil)
			if err != nil {
				t.Fatalf("RequestWithStatusNoRetryRaw() error = %v", err)
			}
			if response.StatusCode != http.StatusOK || response.Empty || response.Data != "updated" {
				t.Fatalf("response = %#v, want string body", response)
			}
		})
	}
}

func TestRequestWithStatusNoRetrySensitiveRawRedactsDebugResponse(t *testing.T) {
	const secret = "rotated-secret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(secret))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, true)

	debugOutput := testutil.CaptureStderr(t, func() {
		response, err := c.RequestWithStatusNoRetrySensitiveRaw(http.MethodPut, "/v1/user/user-123/refresh", nil, nil)
		if err != nil {
			t.Fatalf("RequestWithStatusNoRetrySensitiveRaw() error = %v", err)
		}
		if response.Data != secret {
			t.Fatalf("response data = %#v, want secret returned to caller", response.Data)
		}
	})
	if strings.Contains(debugOutput, secret) || !strings.Contains(debugOutput, RedactedValue) {
		t.Fatalf("debug output = %q, want a redacted response", debugOutput)
	}
}

func TestPostWithQuerySendsQueryAndBodyWithoutRetry(t *testing.T) {
	requests := 0
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		gotQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	_, err := c.PostWithQuery("/v1/temp-url", map[string]string{"viewMode": "inline", "expiredTime": "3600"}, map[string]any{"key": "value"})
	if err == nil {
		t.Fatal("PostWithQuery returned nil error for HTTP 500")
	}
	if requests != 1 {
		t.Errorf("POST request count = %d, want 1", requests)
	}
	if gotQuery != "expiredTime=3600&viewMode=inline" {
		t.Errorf("query = %q, want sorted vStorage query", gotQuery)
	}
	if gotBody != `{"key":"value"}` {
		t.Errorf("body = %q, want JSON body", gotBody)
	}
}

func TestRequestBytesPreservesMultipartBodyAndBinaryResponse(t *testing.T) {
	var gotMethod, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte{0x49, 0x44, 0x33})
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	response, err := c.RequestBytes(http.MethodPost, "/api/v1/speechtotext/sync", nil, []byte("multipart payload"), "multipart/form-data; boundary=test")
	if err != nil {
		t.Fatalf("RequestBytes() error = %v", err)
	}
	if gotMethod != http.MethodPost || gotContentType != "multipart/form-data; boundary=test" || string(gotBody) != "multipart payload" {
		t.Fatalf("request = %s %q %q, want POST multipart payload", gotMethod, gotContentType, gotBody)
	}
	if response.StatusCode != http.StatusOK || response.ContentType != "audio/mpeg" || !bytes.Equal(response.Data, []byte{0x49, 0x44, 0x33}) {
		t.Fatalf("response = %#v, want raw audio response", response)
	}
}

func TestRequestStreamUsesChunkedMultipartBody(t *testing.T) {
	var gotMethod, gotContentType string
	var gotTransferEncoding []string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotTransferEncoding = r.TransferEncoding
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transcript":"hello"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)
	bodyReader, bodyWriter := io.Pipe()
	go func() {
		_, _ = bodyWriter.Write([]byte("multipart payload"))
		_ = bodyWriter.Close()
	}()

	response, err := c.RequestStream(http.MethodPost, "/api/v1/speechtotext/sync", nil, bodyReader, "multipart/form-data; boundary=test")
	if err != nil {
		t.Fatalf("RequestStream() error = %v", err)
	}
	if gotMethod != http.MethodPost || gotContentType != "multipart/form-data; boundary=test" || string(gotBody) != "multipart payload" {
		t.Fatalf("request = %s %q %q, want streamed multipart POST", gotMethod, gotContentType, gotBody)
	}
	if len(gotTransferEncoding) != 1 || gotTransferEncoding[0] != "chunked" {
		t.Fatalf("transfer encoding = %#v, want chunked stream", gotTransferEncoding)
	}
	if response.StatusCode != http.StatusOK || string(response.Data) != `{"transcript":"hello"}` {
		t.Fatalf("response = %#v, want raw stream response", response)
	}
}

func TestRequestBytesDoesNotRetryWriteFailure(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	if _, err := c.RequestBytes(http.MethodPost, "/api/v1/texttospeech/sync", nil, []byte(`{"input":"hello"}`), "application/json"); err == nil {
		t.Fatal("RequestBytes() returned nil error for HTTP 500")
	}
	if requests != 1 {
		t.Fatalf("POST request count = %d, want 1", requests)
	}
}

func TestPostDoesNotRetryUnauthorizedResponse(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"expired token"}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	_, err := c.Post("/v2/project/servers", map[string]any{"name": "server-a"})
	if err == nil {
		t.Fatal("Post returned nil error for HTTP 401")
	}
	if requests != 1 {
		t.Errorf("POST request count = %d, want 1 (writes must not be replayed after 401)", requests)
	}
}

type fakeTokenProvider struct {
	token        string
	refreshed    string
	refreshCalls int
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (f *fakeTokenProvider) GetToken() (string, error) {
	return f.token, nil
}

func (f *fakeTokenProvider) RefreshToken() (string, error) {
	f.refreshCalls++
	f.token = f.refreshed
	return f.token, nil
}

func TestGetUsesRefreshedTokenAfterTransientRetriesAreExhausted(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests <= maxRetries {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"temporary failure"}`))
			return
		}
		if requests == maxRetries+1 {
			if got := r.Header.Get("Authorization"); got != "Bearer old-token" {
				t.Errorf("Authorization before refresh = %q", got)
			}
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"expired token"}`))
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer new-token" {
			t.Errorf("Authorization after refresh = %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	provider := &fakeTokenProvider{token: "old-token", refreshed: "new-token"}
	c := NewGreennodeClient(srv.URL, provider, time.Second, time.Second, false, false)
	c.sleep = func(time.Duration) {}

	result, err := c.Get("/v2/project/servers", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if requests != maxRetries+2 {
		t.Errorf("request count = %d, want %d", requests, maxRetries+2)
	}
	if provider.refreshCalls != 1 {
		t.Errorf("refresh calls = %d, want 1", provider.refreshCalls)
	}
	resultMap, ok := result.(map[string]any)
	if !ok || resultMap["ok"] != true {
		t.Errorf("result = %#v, want successful refreshed response", result)
	}
}

// --- context cancellation ---

func TestContextCancelAbortsMidRequest(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, 5*time.Second, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	c.SetBaseContext(ctx)

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Get("/v1/thing", nil)
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("Get() error = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get() did not return after its context was canceled mid-request")
	}
}

func TestContextCancelAbortsBackoffWait(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, 5*time.Second, false, false)

	blocked := make(chan struct{})
	c.sleep = func(time.Duration) {
		close(blocked)
		time.Sleep(500 * time.Millisecond)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.SetBaseContext(ctx)

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Get("/v1/thing", nil)
		errCh <- err
	}()

	<-blocked
	cancel()

	select {
	case err := <-errCh:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("Get() error = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get() did not return after its context was canceled during the retry backoff wait")
	}
	if requests != 1 {
		t.Errorf("request count = %d, want 1 (canceled while waiting to retry, before a second request)", requests)
	}
}

// --- jitter bounds ---

func TestFullJitterWithinBounds(t *testing.T) {
	base := 4 * time.Second
	for _, r := range []float64{0, 0.25, 0.5, 0.75, 0.999999} {
		got := fullJitter(base, func() float64 { return r })
		if got < base/2 || got > base {
			t.Errorf("fullJitter(%v, rand=%v) = %v, want in [%v, %v]", base, r, got, base/2, base)
		}
	}
}

func TestJitteredBackoffExponentialBaseWithJitterBounds(t *testing.T) {
	tests := []struct {
		attempt  int
		wantBase time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
	}
	for _, tt := range tests {
		if got := jitteredBackoff(tt.attempt, func() float64 { return 0 }); got != tt.wantBase/2 {
			t.Errorf("jitteredBackoff(%d, rand=0) = %v, want the lower bound %v", tt.attempt, got, tt.wantBase/2)
		}
		got := jitteredBackoff(tt.attempt, func() float64 { return 0.999999999 })
		if got < tt.wantBase/2 || got > tt.wantBase {
			t.Errorf("jitteredBackoff(%d, rand~1) = %v, want in [%v, %v]", tt.attempt, got, tt.wantBase/2, tt.wantBase)
		}
	}
}

// --- truncated body ---

type errorReadCloser struct{ err error }

func (e errorReadCloser) Read([]byte) (int, error) { return 0, e.err }
func (e errorReadCloser) Close() error             { return nil }

func TestTruncatedResponseBodyReturnsErrorInsteadOfEmptySuccess(t *testing.T) {
	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient("https://example.invalid", tm, time.Second, time.Second, false, false)
	c.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       errorReadCloser{err: io.ErrUnexpectedEOF},
		}, nil
	})

	_, err := c.Get("/v1/thing", nil)
	if err == nil {
		t.Fatal("Get() returned nil error for a response body that failed to read")
	}
	if !strings.Contains(err.Error(), "response body read failed") {
		t.Errorf("Get() error = %v, want it to explain the response was discarded", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Get() error = %v, want it to wrap the underlying read error", err)
	}
}

// --- invalid URL ---

func TestInvalidRequestURLReturnsErrorNamingTheURL(t *testing.T) {
	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))

	invalidBase := "https://example.invalid/\x7f"
	c := NewGreennodeClient(invalidBase, tm, time.Second, time.Second, false, false)

	_, err := c.Get("/v1/thing", map[string]string{"page": "0"})
	if err == nil {
		t.Fatal("Get() returned nil error for an unparseable request URL")
	}
	if !strings.Contains(err.Error(), "https://example.invalid") || !strings.Contains(err.Error(), "/v1/thing") {
		t.Errorf("Get() error = %v, want it to name the invalid URL", err)
	}
}

// --- exact URL/query emission ---

func TestGetEmitsExactPathForNoParams(t *testing.T) {
	var gotPath, gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	if _, err := c.Get("/v1/clusters/cls-1/node-groups", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotPath != "/v1/clusters/cls-1/node-groups" || gotRawQuery != "" {
		t.Errorf("path = %q query = %q, want the exact path and no query string", gotPath, gotRawQuery)
	}
}

func TestGetEmitsExactQueryForParams(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	if _, err := c.Get("/v1/clusters", map[string]string{"page": "0", "pageSize": "50"}); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotRawQuery != "page=0&pageSize=50" {
		t.Errorf("query = %q, want the exact sorted VKS pagination query", gotRawQuery)
	}
}

// --- formatError redaction ---

func TestFormatErrorRedactsCredentialBearingFallbackField(t *testing.T) {
	body := []byte(`{"status":"error","token":"super-secret-leaked-value","requestId":"req-1"}`)
	got := formatError(http.StatusBadRequest, body)
	if strings.Contains(got, "super-secret-leaked-value") {
		t.Fatalf("formatError leaked a credential-bearing field: %q", got)
	}
	if !strings.Contains(got, RedactedValue) {
		t.Errorf("formatError = %q, want the %s sentinel in place of the token field", got, RedactedValue)
	}
	if !strings.Contains(got, "req-1") {
		t.Errorf("formatError = %q, want the non-sensitive requestId preserved", got)
	}
}

// --- 429 / Retry-After ---

func TestRetryAfterDelay(t *testing.T) {
	fallback := 750 * time.Millisecond
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty falls back", "", fallback},
		{"unparseable falls back", "not-a-valid-value", fallback},
		{"seconds", "2", 2 * time.Second},
		{"seconds capped at 30s", "3600", retryAfterCap},
		{"negative seconds clamped to zero", "-5", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryAfterDelay(tt.header, fallback); got != tt.want {
				t.Errorf("retryAfterDelay(%q, %v) = %v, want %v", tt.header, fallback, got, tt.want)
			}
		})
	}
}

func TestRetryAfterDelayHTTPDate(t *testing.T) {
	fallback := time.Second

	future := time.Now().Add(10 * time.Second)
	if got := retryAfterDelay(future.UTC().Format(http.TimeFormat), fallback); got < 8*time.Second || got > 10*time.Second {
		t.Errorf("retryAfterDelay(future HTTP-date) = %v, want ~10s", got)
	}

	past := time.Now().Add(-10 * time.Second)
	if got := retryAfterDelay(past.UTC().Format(http.TimeFormat), fallback); got != 0 {
		t.Errorf("retryAfterDelay(past HTTP-date) = %v, want 0 (clamped)", got)
	}
}

func TestGetRetriesTooManyRequestsHonoringRetryAfter(t *testing.T) {
	requests := 0
	var waited []time.Duration
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)
	var waitedMu sync.Mutex
	c.sleep = func(d time.Duration) {
		waitedMu.Lock()
		waited = append(waited, d)
		waitedMu.Unlock()
	}

	result, err := c.Get("/v1/thing", nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if requests != 2 {
		t.Errorf("request count = %d, want 2 (retried once after 429)", requests)
	}
	if len(waited) != 1 || waited[0] != 2*time.Second {
		t.Errorf("waited = %v, want a single 2s wait honoring Retry-After", waited)
	}
	if resultMap, ok := result.(map[string]any); !ok || resultMap["ok"] != true {
		t.Errorf("result = %#v, want the successful retried response", result)
	}
}

func TestPostDoesNotRetryTooManyRequests(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, false)

	_, err := c.Post("/v2/project/servers", map[string]any{"name": "server-a"})
	if err == nil {
		t.Fatal("Post returned nil error for HTTP 429")
	}
	if requests != 1 {
		t.Errorf("POST request count = %d, want 1 (429 must not be retried on a write)", requests)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("error = %v, want *APIError with StatusCode 429", err)
	}
}

// --- debug output format ---

func TestDebugOutputIncludesAttemptCountAndResponseDuration(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, tm, time.Second, time.Second, false, true)
	c.sleep = func(time.Duration) {}

	output := testutil.CaptureStderr(t, func() {
		if _, err := c.Get("/v1/thing", nil); err != nil {
			t.Fatalf("Get returned error: %v", err)
		}
	})

	if !strings.Contains(output, "(attempt 1/5)") {
		t.Errorf("debug output = %q, want a first-attempt line with (attempt 1/5)", output)
	}
	if !strings.Contains(output, "(attempt 2/5)") {
		t.Errorf("debug output = %q, want a second-attempt line with (attempt 2/5)", output)
	}
	if !strings.Contains(output, "response 503 in ") {
		t.Errorf("debug output = %q, want the first response line to report status 503 with a duration", output)
	}
	if !strings.Contains(output, "response 200 in ") {
		t.Errorf("debug output = %q, want the second response line to report status 200 with a duration", output)
	}
}

// --- SetBaseContext ---

func TestSetBaseContextDefaultsToBackgroundWhenNil(t *testing.T) {
	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(time.Hour))
	c := NewGreennodeClient("https://example.invalid", tm, time.Second, time.Second, false, false)
	//lint:ignore SA1012 deliberately exercising SetBaseContext's documented nil handling
	c.SetBaseContext(nil)
	if got := c.context(); got != context.Background() {
		t.Errorf("context() = %v, want context.Background() after SetBaseContext(nil)", got)
	}
}

func TestOrdinaryRequestsStillLogTheirBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))
	c := NewGreennodeClient(srv.URL, tm, 5*time.Second, 5*time.Second, false, true)

	stderr := testutil.CaptureStderr(t, func() {
		_, _ = c.RequestWithStatusNoRetry(http.MethodPost, "/v1/thing", nil, map[string]any{"name": "visible-name"})
	})
	if !strings.Contains(stderr, "visible-name") {
		t.Errorf("ordinary request body was suppressed:\n%s", stderr)
	}
}

func TestBuildRequestURLPreservesAPathQuery(t *testing.T) {
	const base = "https://agentbase.api.vngcloud.vn/runtime"

	t.Run("no params returns the path untouched", func(t *testing.T) {
		got, err := buildRequestURL(base, "/v1/agent-runtimes:search-traces?tag=a&tag=b", nil)
		if err != nil {
			t.Fatalf("buildRequestURL error = %v", err)
		}
		if want := base + "/v1/agent-runtimes:search-traces?tag=a&tag=b"; got != want {
			t.Errorf("url = %q, want %q", got, want)
		}
	})

	t.Run("repeated names survive alongside params", func(t *testing.T) {
		got, err := buildRequestURL(base, "/v1/agent-runtimes:get-trace?tag=a&tag=b", map[string]string{"traceId": "abc123"})
		if err != nil {
			t.Fatalf("buildRequestURL error = %v", err)
		}
		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("parse %q: %v", got, err)
		}
		values := parsed.Query()
		if tags := values["tag"]; len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
			t.Errorf("tag = %v, want both values (url %q)", tags, got)
		}
		if values.Get("traceId") != "abc123" {
			t.Errorf("traceId = %q, want abc123 (url %q)", values.Get("traceId"), got)
		}
	})

	t.Run("an escaped value is not decoded into extra parameters", func(t *testing.T) {
		attached := url.Values{"q": {"a&limit=9999"}}.Encode()
		got, err := buildRequestURL(base, "/v1/agent-runtimes:search-traces?"+attached, map[string]string{"traceId": "abc123"})
		if err != nil {
			t.Fatalf("buildRequestURL error = %v", err)
		}
		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("parse %q: %v", got, err)
		}
		values := parsed.Query()
		if len(values) != 2 {
			t.Errorf("url %q has %d parameters, want exactly q and traceId", got, len(values))
		}
		if values.Get("q") != "a&limit=9999" {
			t.Errorf("q = %q, want the literal value (url %q)", values.Get("q"), got)
		}
		if values.Has("limit") {
			t.Errorf("url %q gained an injected limit parameter", got)
		}
	})
}
