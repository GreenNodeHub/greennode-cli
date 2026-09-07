package vdbclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/auth"
	"github.com/greennodehub/greennode-cli/internal/client"
)

// testClient builds a vdb client pointed at a test server, with a token already in
// hand so no IAM round trip happens.
func testClient(endpoint string) *Client {
	tm := auth.NewMachineTokenProvider("id", "secret", "")
	tm.SetToken("test-token", time.Now().Add(1*time.Hour))
	return &Client{GreennodeClient: client.NewGreennodeClient(endpoint, tm, 5*time.Second, 5*time.Second, false, false)}
}

// TestErrorSurfacesTheUsefulMessage is the regression test for a real complaint:
// vdb answers a bad request with a machine code in "message" and the sentence that
// explains the problem in "errors[].message". The shared client stops at the first
// recognised field, so the user used to see only "in_valid" and had to re-run with
// --debug to learn what was wrong.
func TestErrorSurfacesTheUsefulMessage(t *testing.T) {
	body := `{"code":400,"message":"in_valid","errors":[{"action":null,` +
		`"fieldError":"invalid","message":"The field pageSize of the request must be greater than 0",` +
		`"request":null,"typeError":"pageSize"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	_, err := testClient(srv.URL).Get("/vdb-relational/v1/database-instances", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 400")
	}

	message := err.Error()
	for _, want := range []string{
		"400",                    // the status
		"in_valid",               // the machine code, kept for support
		"must be greater than 0", // the part the user needs
		"pageSize",               // the offending field
	} {
		if !strings.Contains(message, want) {
			t.Errorf("error %q does not mention %q", message, want)
		}
	}

	// The shared APIError must stay reachable so callers can branch on the status.
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As could not reach *client.APIError")
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Body != body {
		t.Error("APIError.Body no longer carries the raw response")
	}
}

func TestEnrichLeavesOtherErrorsAlone(t *testing.T) {
	// Not an API error at all: pass through untouched.
	plain := fmt.Errorf("dial tcp: connection refused")
	if got := enrich(plain, nil); got != plain {
		t.Errorf("enrich rewrote a non-API error: %v", got)
	}
	if enrich(nil, nil) != nil {
		t.Error("enrich(nil) should stay nil")
	}

	// An API error whose body has nothing to add must not be replaced, so its
	// message keeps whatever the shared formatter produced. These use 400, not 500:
	// the shared client retries 5xx three times with backoff, which would make this
	// test take half a minute.
	for _, body := range []string{
		`{"code":400,"message":"internal"}`,      // no errors array
		`{"code":400,"errors":[]}`,               // empty errors array
		`{"code":400,"errors":[{"message":""}]}`, // blank detail
		`<html>400 Bad Request</html>`,           // not JSON at all
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(body))
		}))

		_, err := testClient(srv.URL).Get("/x", nil)
		srv.Close()

		if err == nil {
			t.Fatalf("body %q: expected an error", body)
		}
		var enriched *Error
		if errors.As(err, &enriched) {
			t.Errorf("body %q was wrapped as enriched although it carries no details: %v", body, err)
		}
	}
}

// TestErrorJoinsSeveralDetails: a validation failure can name more than one field.
func TestErrorJoinsSeveralDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":400,"message":"in_valid","errors":[` +
			`{"message":"first problem","typeError":"fieldA"},` +
			`{"message":"second problem","typeError":"fieldB"}]}`))
	}))
	defer srv.Close()

	_, err := testClient(srv.URL).Get("/x", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "first problem") || !strings.Contains(err.Error(), "second problem") {
		t.Errorf("error %q dropped one of the details", err)
	}
}

// TestErrorRedactsSecretsFromTheMessage guards a live leak: on a rejected password the
// API repeats it verbatim in the error detail ("Redis password [...] contains invalid
// character"), which would otherwise be printed to the terminal.
func TestErrorRedactsSecretsFromTheMessage(t *testing.T) {
	const secret = "SuperSecret2026abcd"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":400,"message":"in_valid","errors":[{"typeError":"invalid",` +
			`"message":"Redis password [` + secret + `] contains invalid character"}]}`))
	}))
	defer srv.Close()

	// Nested exactly as a restore body nests it, to prove the walk reaches inside.
	body := map[string]interface{}{
		"databaseInstances": []interface{}{
			map[string]interface{}{"config": map[string]interface{}{"redisPassword": secret}},
		},
	}

	_, err := testClient(srv.URL).Put("/x", body)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the password reached the error message: %q", err)
	}
	// The rest of the sentence must survive, or the user cannot tell what was wrong.
	for _, want := range []string{"contains invalid character", "***"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, must contain %q", err, want)
		}
	}

	// A request with no secret is unaffected, and stays the same error value.
	if _, err := testClient(srv.URL).Get("/x", nil); !strings.Contains(err.Error(), secret) {
		t.Errorf("a message with no secret in the request must be left intact: %q", err)
	}
}
