package vks

import (
	"errors"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/client"
)

func TestEvaluateActive(t *testing.T) {
	forbidden := &client.APIError{StatusCode: 403, Body: ""}
	notFound := &client.APIError{StatusCode: 404, Body: ""}
	serverErr := &client.APIError{StatusCode: 500, Body: ""}

	cases := []struct {
		name               string
		result             interface{}
		err                error
		wantDone, wantFail bool
		wantFatal          bool
	}{
		{"active", map[string]interface{}{"status": "ACTIVE"}, nil, true, false, false},
		{"creating", map[string]interface{}{"status": "CREATING"}, nil, false, false, false},
		{"error", map[string]interface{}{"status": "ERROR"}, nil, false, true, false},
		{"failed", map[string]interface{}{"status": "FAILED"}, nil, false, true, false},
		{"transient err", nil, errors.New("boom"), false, false, false},
		{"transient 500", nil, serverErr, false, false, false},
		{"forbidden 403 fatal", nil, forbidden, false, false, true},
		{"not found 404 fatal", nil, notFound, false, false, true},
	}
	for _, tc := range cases {
		done, failed, _, fatal := evaluateActive(tc.result, tc.err)
		if done != tc.wantDone || failed != tc.wantFail || (fatal != nil) != tc.wantFatal {
			t.Errorf("%s: done=%v failed=%v fatal=%v, want done=%v failed=%v fatal=%v",
				tc.name, done, failed, fatal != nil, tc.wantDone, tc.wantFail, tc.wantFatal)
		}
	}
}

func TestEvaluateDeleted(t *testing.T) {
	notFound := &client.APIError{StatusCode: 404, Body: ""}
	forbidden := &client.APIError{StatusCode: 403, Body: ""}
	serverErr := &client.APIError{StatusCode: 500, Body: ""}

	cases := []struct {
		name               string
		result             interface{}
		err                error
		wantDone, wantFail bool
		wantFatal          bool
	}{
		{"gone 404", nil, notFound, true, false, false},
		{"still deleting", map[string]interface{}{"status": "DELETING"}, nil, false, false, false},
		{"came back active", map[string]interface{}{"status": "ACTIVE"}, nil, false, true, false},
		{"non-404 err transient", nil, serverErr, false, false, false},
		{"plain transient err", nil, errors.New("boom"), false, false, false},
		{"forbidden 403 fatal", nil, forbidden, false, false, true},
	}
	for _, tc := range cases {
		done, failed, _, fatal := evaluateDeleted(tc.result, tc.err)
		if done != tc.wantDone || failed != tc.wantFail || (fatal != nil) != tc.wantFatal {
			t.Errorf("%s: done=%v failed=%v fatal=%v, want done=%v failed=%v fatal=%v",
				tc.name, done, failed, fatal != nil, tc.wantDone, tc.wantFail, tc.wantFatal)
		}
	}
}
