package loadbalancer

import (
	"errors"
	"fmt"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/client"
)

// lbResponse builds a GET response in the shape VLB returns: the resource under
// a "data" envelope, with the state in displayStatus (not "status").
func lbResponse(status string) interface{} {
	return map[string]interface{}{"data": map[string]interface{}{
		"uuid": "lb-1", "name": "test", "displayStatus": status, "progressStatus": status,
	}}
}

func apiErr(code int) error {
	return fmt.Errorf("wrapped: %w", &client.APIError{StatusCode: code})
}

func TestEvaluateActive(t *testing.T) {
	tests := []struct {
		name       string
		result     interface{}
		err        error
		wantDone   bool
		wantFailed bool
		wantFatal  bool
		wantStatus string
	}{
		{name: "ACTIVE finishes the wait", result: lbResponse("ACTIVE"), wantDone: true, wantStatus: "ACTIVE"},
		{name: "CREATING keeps polling", result: lbResponse("CREATING"), wantStatus: "CREATING"},
		{name: "UPDATING keeps polling", result: lbResponse("UPDATING"), wantStatus: "UPDATING"},
		{name: "DELETING keeps polling", result: lbResponse("DELETING"), wantStatus: "DELETING"},
		{name: "ERROR is terminal", result: lbResponse("ERROR"), wantFailed: true, wantStatus: "ERROR"},
		{name: "FAILED is terminal", result: lbResponse("FAILED"), wantFailed: true, wantStatus: "FAILED"},
		{name: "an unknown status keeps polling", result: lbResponse("REBALANCING"), wantStatus: "REBALANCING"},
		{name: "404 aborts — waiting cannot conjure it", err: apiErr(404), wantFatal: true},
		{name: "403 aborts", err: apiErr(403), wantFatal: true},
		{name: "400 aborts", err: apiErr(400), wantFatal: true},
		{name: "500 is transient, keep polling", err: apiErr(500)},
		{name: "a network error is transient", err: errors.New("connection reset")},
		{name: "an unparseable body keeps polling", result: "not an object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, failed, status, fatal := evaluateActive(tt.result, tt.err)
			checkWait(t, done, failed, status, fatal, tt.wantDone, tt.wantFailed, tt.wantStatus, tt.wantFatal)
		})
	}
}

func TestEvaluateDeleted(t *testing.T) {
	tests := []struct {
		name       string
		result     interface{}
		err        error
		wantDone   bool
		wantFailed bool
		wantFatal  bool
		wantStatus string
	}{
		{name: "404 means gone", err: apiErr(404), wantDone: true, wantStatus: "DELETED"},
		{name: "DELETING keeps polling", result: lbResponse("DELETING"), wantStatus: "DELETING"},
		{name: "ACTIVE before DELETING keeps polling", result: lbResponse("ACTIVE"), wantStatus: "ACTIVE"},
		{name: "ERROR is terminal", result: lbResponse("ERROR"), wantFailed: true, wantStatus: "ERROR"},
		{name: "403 aborts", err: apiErr(403), wantFatal: true},
		{name: "500 is transient", err: apiErr(500)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, failed, status, fatal := newDeletedEvaluator()(tt.result, tt.err)
			checkWait(t, done, failed, status, fatal, tt.wantDone, tt.wantFailed, tt.wantStatus, tt.wantFatal)
		})
	}
}

// The first poll after a delete can still see ACTIVE; only ACTIVE after
// DELETING means the delete was rolled back.
func TestEvaluateDeletedFailsOnActiveOnlyAfterDeleting(t *testing.T) {
	eval := newDeletedEvaluator()
	steps := []struct {
		status     string
		wantFailed bool
	}{
		{"ACTIVE", false},
		{"DELETING", false},
		{"ACTIVE", true},
	}
	for i, step := range steps {
		_, failed, _, _ := eval(lbResponse(step.status), nil)
		if failed != step.wantFailed {
			t.Errorf("poll %d (%s): failed = %v, want %v", i+1, step.status, failed, step.wantFailed)
		}
	}
}

func checkWait(t *testing.T, done, failed bool, status string, fatal error,
	wantDone, wantFailed bool, wantStatus string, wantFatal bool) {
	t.Helper()
	if done != wantDone {
		t.Errorf("done = %v, want %v", done, wantDone)
	}
	if failed != wantFailed {
		t.Errorf("failed = %v, want %v", failed, wantFailed)
	}
	if (fatal != nil) != wantFatal {
		t.Errorf("fatal = %v, want fatal=%v", fatal, wantFatal)
	}
	if wantStatus != "" && status != wantStatus {
		t.Errorf("status = %q, want %q", status, wantStatus)
	}
}

// TestStatusOfReadsTheVLBShape guards the two ways VLB differs from VKS here:
// the data envelope and the displayStatus field name.
func TestStatusOfReadsTheVLBShape(t *testing.T) {
	if got := statusOf(lbResponse("ACTIVE")); got != "ACTIVE" {
		t.Errorf("statusOf(envelope) = %q", got)
	}
	bare := map[string]interface{}{"displayStatus": "ACTIVE"}
	if got := statusOf(bare); got != "ACTIVE" {
		t.Errorf("statusOf(bare object) = %q", got)
	}
	// "status" is the VKS field name and must not be picked up by accident.
	if got := statusOf(map[string]interface{}{"data": map[string]interface{}{"status": "ACTIVE"}}); got != "" {
		t.Errorf("statusOf should read displayStatus, not status; got %q", got)
	}
	if got := statusOf("nonsense"); got != "" {
		t.Errorf("statusOf(non-object) = %q, want empty", got)
	}
}

func TestWaitCommandsAreRegistered(t *testing.T) {
	found := map[string]bool{}
	for _, c := range LoadBalancerCmd.Commands() {
		found[c.Name()] = true
		if c.Name() == "wait-active" || c.Name() == "wait-deleted" {
			for _, flag := range []string{"load-balancer-id", "delay", "max-attempts"} {
				if c.Flags().Lookup(flag) == nil {
					t.Errorf("%s is missing --%s", c.Name(), flag)
				}
			}
		}
	}
	for _, name := range []string{"wait-active", "wait-deleted"} {
		if !found[name] {
			t.Errorf("`loadbalancer %s` is not registered", name)
		}
	}
}
