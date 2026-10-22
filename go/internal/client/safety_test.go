package client

import (
	"errors"
	"github.com/greennodehub/greennode-cli/internal/auth"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSensitiveErrorHidesResponseButPreservesTypedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("credential-material"))
	}))
	defer srv.Close()
	p := auth.NewMachineTokenProvider("client", "secret", "")
	p.SetToken("test", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, p, time.Second, time.Second, true, true)
	_, err := c.RequestWithStatusNoRetrySensitiveRaw("POST", "/secret", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Body != "credential-material" {
		t.Fatal("raw typed body lost")
	}
	if strings.Contains(err.Error(), "credential-material") {
		t.Fatal("displayed error leaked sensitive response")
	}
}

func TestTransportErrorHidesQueryAndPathCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	p := auth.NewMachineTokenProvider("client", "secret", "")
	p.SetToken("test", time.Now().Add(time.Hour))
	c := NewGreennodeClient(srv.URL, p, time.Second, time.Second, true, false)
	c.MaskPathValues([]string{"path-secret"})
	_, err := c.RequestWithStatusNoRetry("DELETE", "/keys/path-secret", map[string]string{"api_key": "query-secret"}, nil)
	if err == nil {
		t.Fatal("closed server succeeded")
	}
	if strings.Contains(err.Error(), "path-secret") || strings.Contains(err.Error(), "query-secret") {
		t.Fatalf("error leaked URL: %v", err)
	}
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		t.Fatal("underlying transport error lost")
	}
}
