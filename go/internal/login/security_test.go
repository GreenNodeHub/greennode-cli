package login

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginDebugSuppressesResponseDetails(t *testing.T) {
	restore := stubBrowser()
	defer restore()
	previousWriter, previousDebug := noisyStderr, debugMode
	var logs bytes.Buffer
	noisyStderr, debugMode = &logs, true
	defer func() { noisyStderr, debugMode = previousWriter, previousDebug }()
	for _, tc := range []struct {
		status int
		body   string
	}{
		{400, `{"error_description":"fixture-secret"}`},
		{200, `{"access_token":"fixture-access","token_type":"fixture-secret","refresh_token":"fixture-refresh","expires_in":3600}`},
		{200, `{"access_token":"fixture-access","expires_in":1e9999}`},
	} {
		logs.Reset()
		server := startFakeIAM(t, &fakeIAM{authCode: "fixture-code", tokenBody: tc.body, tokenStatus: tc.status})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := Login(ctx, newLoginTestCfg(server.URL, ""))
		cancel()
		server.Close()
		output := logs.String()
		if err != nil {
			output += err.Error()
		}
		for _, secret := range []string{"fixture-secret", "1e9999"} {
			if strings.Contains(output, secret) {
				t.Fatalf("response detail disclosed: %s", output)
			}
		}
	}
}

func TestTokenGrantsRejectRedirects(t *testing.T) {
	for _, grant := range []string{"exchange", "refresh"} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("%s/%d", grant, status), func(t *testing.T) {
				var calls atomic.Int32
				destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					_, _ = io.WriteString(w, `{"access_token":"fixture-access"}`)
				}))
				defer destination.Close()
				origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, destination.URL, status)
				}))
				defer origin.Close()
				client := New(time.Second)
				var endpointErr *Error
				var err error
				if grant == "exchange" {
					_, endpointErr, err = client.ExchangeCode(context.Background(), origin.URL, ExchangeParams{Code: "fixture-code", CodeVerifier: "fixture-verifier", ClientID: "fixture-client"})
				} else {
					_, endpointErr, err = client.Refresh(context.Background(), origin.URL, RefreshParams{RefreshToken: "fixture-refresh", ClientID: "fixture-client"})
				}
				if err == nil && endpointErr == nil || calls.Load() != 0 {
					t.Fatalf("redirect followed: calls=%d error=%v endpoint error=%v", calls.Load(), err, endpointErr)
				}
			})
		}
	}
}

func TestTokenEndpointErrorSuppressesBody(t *testing.T) {
	err := &Error{Status: http.StatusBadRequest, RawBody: []byte(`{"data":"fixture-secret"}`)}
	if strings.Contains(err.Error(), "fixture-secret") || !strings.Contains(err.Error(), "400") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestTokenTransportErrorSuppressesURL(t *testing.T) {
	client := New(time.Second)
	for _, endpoint := range []string{"http://%fixture-secret", "http://127.0.0.1:1/?code=fixture-secret"} {
		_, _, err := client.ExchangeCode(context.Background(), endpoint, ExchangeParams{})
		if err == nil || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := client.Refresh(ctx, "http://127.0.0.1:1/fixture-token", RefreshParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
