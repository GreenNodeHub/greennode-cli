package vks

import (
	"errors"
	"strings"
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

func TestRunWaiterReturnsExitCode255(t *testing.T) {
	tests := []struct {
		name string
		eval evaluator
		want string
	}{
		{
			name: "fatal",
			eval: func(interface{}, error) (bool, bool, string, error) {
				return false, false, "", errors.New("forbidden")
			},
			want: "waiting for fixture",
		},
		{
			name: "terminal status",
			eval: func(interface{}, error) (bool, bool, string, error) {
				return false, true, "FAILED", nil
			},
			want: "reached FAILED",
		},
		{
			name: "timeout",
			eval: func(interface{}, error) (bool, bool, string, error) {
				return false, false, "CREATING", nil
			},
			want: "timed out",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runWaiter("fixture", "done", func() (interface{}, error) { return nil, nil }, tc.eval, 0, 1)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want text %q", err, tc.want)
			}
			var coded interface{ ExitCode() int }
			if !errors.As(err, &coded) || coded.ExitCode() != 255 {
				t.Fatalf("error = %T %v, want exit code 255", err, err)
			}
		})
	}
}
