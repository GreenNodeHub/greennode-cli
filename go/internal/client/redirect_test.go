package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
)

func TestRequestsRejectRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, mode := range []string{"read", "write", "sensitive", "stream"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				var origins, destinations atomic.Int32
				destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					destinations.Add(1)
					w.Write([]byte(`{}`))
				}))
				defer destination.Close()
				origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					origins.Add(1)
					w.Header().Set("Location", destination.URL)
					w.WriteHeader(status)
				}))
				defer origin.Close()
				provider := auth.NewMachineTokenProvider("fixture-client", "fixture-secret", "")
				provider.SetToken("fixture-token", time.Now().Add(time.Hour))
				c := NewGreennodeClient(origin.URL, provider, time.Second, time.Second, true, false)
				var err error
				switch mode {
				case "read":
					_, err = c.Get("/resource", nil)
				case "write":
					_, err = c.Post("/resource", map[string]any{"name": "fixture-resource"})
				case "sensitive":
					_, err = c.RequestWithStatusNoRetrySensitive("POST", "/resource", nil, map[string]any{"secret": "fixture-secret"})
				case "stream":
					_, err = c.RequestStream("POST", "/resource", nil, strings.NewReader("fixture-body"), "text/plain")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
					t.Errorf("error = %v, want HTTP %d", err, status)
				}
				if origins.Load() != 1 || destinations.Load() != 0 {
					t.Errorf("origin calls = %d, redirected calls = %d", origins.Load(), destinations.Load())
				}
			})
		}
	}
}
