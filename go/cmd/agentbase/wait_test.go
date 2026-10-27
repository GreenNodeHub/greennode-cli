package agentbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientpkg "github.com/greennodehub/greennode-cli/internal/agentbase/client"
	gatewaypkg "github.com/greennodehub/greennode-cli/internal/agentbase/gateway"
	runtimepkg "github.com/greennodehub/greennode-cli/internal/agentbase/runtime"
)

// fakeWaitTokenProvider hands out a static bearer token so the poll tests can
// hit an httptest server through the real agentbase client stack (the same seam
// runtimepkg/gatewaypkg use in their own package tests).
type fakeWaitTokenProvider struct{}

func (fakeWaitTokenProvider) GetToken() (string, error)     { return "test-token", nil }
func (fakeWaitTokenProvider) RefreshToken() (string, error) { return "test-token", nil }

// newWaitServer spins up an httptest server backed by handler and closes it
// with the test.
func newWaitServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// serve404 replies with the error body the API returns once a delete has
// removed the record.
func serve404(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, `{"error":{"code":"not_found","message":"not found"}}`)
}

// serveJSON marshals v as a 200 response.
func serveJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- runtime wait polling ---------------------------------------------------

// TestPollRuntimeToTerminal_NotFoundCountsAsDeleted covers the fast delete:
// the record is already gone by the first poll, so the wait never sees a
// runtime. With notFoundIsDeleted the 404 maps onto the DELETED terminal
// success and the caller gets a minimal {id, DELETED} snapshot.
func TestPollRuntimeToTerminal_NotFoundCountsAsDeleted(t *testing.T) {
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) { serve404(w) })
	client := runtimepkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	rt, err := pollRuntimeToTerminal(context.Background(), client, "rt-1", time.Millisecond, true)
	if err != nil {
		t.Fatalf("pollRuntimeToTerminal: %v", err)
	}
	if rt == nil || rt.ID != "rt-1" || rt.Status != "DELETED" {
		t.Errorf("rt = %+v, want minimal {id: rt-1, status: DELETED} snapshot", rt)
	}
}

// TestPollRuntimeToTerminal_NotFoundStillErrorsWhenNotWaitingForDelete keeps
// the 404 an error for the ACTIVE flows (deploy up): a runtime vanishing
// mid-converge is not a success there.
func TestPollRuntimeToTerminal_NotFoundStillErrorsWhenNotWaitingForDelete(t *testing.T) {
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) { serve404(w) })
	client := runtimepkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	_, err := pollRuntimeToTerminal(context.Background(), client, "rt-1", time.Millisecond, false)
	if err == nil {
		t.Fatal("expected the 404 to surface as an error when not waiting for a delete")
	}
	var apiErr *clientpkg.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("err = %v, want *client.APIError with status 404", err)
	}
}

// TestPollRuntimeToTerminal_DeletingThenNotFoundReturnsLastSeen covers the
// slow delete: the runtime is seen in DELETING, then disappears. The 404 counts
// as DELETED and the caller gets the last-seen runtime, not a synthetic one.
func TestPollRuntimeToTerminal_DeletingThenNotFoundReturnsLastSeen(t *testing.T) {
	var calls int32
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			serveJSON(w, map[string]string{"id": "rt-1", "status": "DELETING"})
			return
		}
		serve404(w)
	})
	client := runtimepkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	rt, err := pollRuntimeToTerminal(context.Background(), client, "rt-1", time.Millisecond, true)
	if err != nil {
		t.Fatalf("pollRuntimeToTerminal: %v", err)
	}
	if rt == nil || rt.ID != "rt-1" || rt.Status != "DELETING" {
		t.Errorf("rt = %+v, want the last-seen DELETING runtime", rt)
	}
}

// TestPollRuntimeToTerminal_ActiveStopsPolling guards the plain success path:
// one ACTIVE response ends the loop without extra polls.
func TestPollRuntimeToTerminal_ActiveStopsPolling(t *testing.T) {
	var calls int32
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		serveJSON(w, map[string]string{"id": "rt-1", "status": "ACTIVE"})
	})
	client := runtimepkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	rt, err := pollRuntimeToTerminal(context.Background(), client, "rt-1", time.Millisecond, true)
	if err != nil {
		t.Fatalf("pollRuntimeToTerminal: %v", err)
	}
	if rt.Status != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", rt.Status)
	}
	if calls != 1 {
		t.Errorf("polls = %d, want 1 (ACTIVE is terminal)", calls)
	}
}

// TestPollRuntimeToTerminal_ErrorStateFails guards the terminal-failure path.
func TestPollRuntimeToTerminal_ErrorStateFails(t *testing.T) {
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		serveJSON(w, map[string]string{"id": "rt-1", "status": "ERROR", "statusReason": "boom"})
	})
	client := runtimepkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	_, err := pollRuntimeToTerminal(context.Background(), client, "rt-1", time.Millisecond, true)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Errorf("err = %v, want a terminal-failure error", err)
	}
}

// --- gateway wait polling ---------------------------------------------------

// TestPollGatewayToTerminal_NotFoundCountsAsDeleted covers the fast delete:
// the gateway is already gone by the first poll, so the wait gets a minimal
// {name, DELETED} snapshot and no error.
func TestPollGatewayToTerminal_NotFoundCountsAsDeleted(t *testing.T) {
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) { serve404(w) })
	client := gatewaypkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	gw, err := pollGatewayToTerminal(context.Background(), client, "gw-1", time.Millisecond)
	if err != nil {
		t.Fatalf("pollGatewayToTerminal: %v", err)
	}
	if gw == nil || gw.Name != "gw-1" || gw.State != "DELETED" {
		t.Errorf("gw = %+v, want minimal {name: gw-1, state: DELETED} snapshot", gw)
	}
}

// TestPollGatewayToTerminal_DeletingThenNotFoundReturnsLastSeen covers the
// observed prod flow: the gateway is seen in DELETING, then disappears
// (~25-31s). The 404 counts as DELETED and the caller gets the last-seen
// gateway.
func TestPollGatewayToTerminal_DeletingThenNotFoundReturnsLastSeen(t *testing.T) {
	var calls int32
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			serveJSON(w, map[string]string{"name": "gw-1", "state": "DELETING"})
			return
		}
		serve404(w)
	})
	client := gatewaypkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	gw, err := pollGatewayToTerminal(context.Background(), client, "gw-1", time.Millisecond)
	if err != nil {
		t.Fatalf("pollGatewayToTerminal: %v", err)
	}
	if gw == nil || gw.Name != "gw-1" || gw.State != "DELETING" {
		t.Errorf("gw = %+v, want the last-seen DELETING gateway", gw)
	}
}

// TestPollGatewayToTerminal_ActiveStopsPolling guards the plain success path.
func TestPollGatewayToTerminal_ActiveStopsPolling(t *testing.T) {
	var calls int32
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		serveJSON(w, map[string]string{"name": "gw-1", "state": "ACTIVE"})
	})
	client := gatewaypkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	gw, err := pollGatewayToTerminal(context.Background(), client, "gw-1", time.Millisecond)
	if err != nil {
		t.Fatalf("pollGatewayToTerminal: %v", err)
	}
	if gw.State != "ACTIVE" {
		t.Errorf("state = %q, want ACTIVE", gw.State)
	}
	if calls != 1 {
		t.Errorf("polls = %d, want 1 (ACTIVE is terminal)", calls)
	}
}

// TestPollGatewayToTerminal_FailureStateErrors guards the terminal-failure
// path (the *_ERROR states keep failing, 404 or not).
func TestPollGatewayToTerminal_FailureStateErrors(t *testing.T) {
	srv := newWaitServer(t, func(w http.ResponseWriter, r *http.Request) {
		serveJSON(w, map[string]string{"name": "gw-1", "state": "CREATE_ERROR"})
	})
	client := gatewaypkg.NewClient(srv.URL, fakeWaitTokenProvider{})
	_, err := pollGatewayToTerminal(context.Background(), client, "gw-1", time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Errorf("err = %v, want a terminal-failure error", err)
	}
}
