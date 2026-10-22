package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type contextualProvider interface {
	GetToken() (string, error)
	SetBaseContext(context.Context)
	SetHTTPTimeout(time.Duration)
}

func TestBothProvidersHonorCancellationAndTimeout(t *testing.T) {
	for _, mode := range []string{"machine", "user"} {
		for _, check := range []string{"canceled", "timeout"} {
			t.Run(mode+"/"+check, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					select {
					case <-r.Context().Done():
					case <-time.After(200 * time.Millisecond):
					}
				}))
				defer srv.Close()
				var p contextualProvider = NewMachineTokenProvider("client", "secret", srv.URL)
				if mode == "user" {
					p = NewLoginTokenProvider("refresh", "client", "", srv.URL, nil)
				}
				p.SetBaseContext(nil)
				p.SetHTTPTimeout(10 * time.Millisecond)
				if check == "canceled" {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					p.SetBaseContext(ctx)
				}
				start := time.Now()
				_, err := p.GetToken()
				if err == nil {
					t.Fatal("request unexpectedly succeeded")
				}
				if check == "canceled" {
					if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
						t.Fatalf("cancellation: %v, calls %d", err, calls.Load())
					}
				} else {
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("timeout sentinel lost: %v", err)
					}
					if time.Since(start) >= 150*time.Millisecond {
						t.Fatalf("IAM timeout ignored: %v", err)
					}
				}
			})
		}
	}
}
