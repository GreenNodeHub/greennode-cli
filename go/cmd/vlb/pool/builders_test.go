package pool

import (
	"reflect"
	"strings"
	"testing"
)

// completeTCPMonitor carries every field the live API requires; the spec marks
// only healthCheckProtocol as required, but the rest are enforced in practice.
const completeTCPMonitor = "healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3"

func TestBuildCreateBody(t *testing.T) {
	t.Run("health monitor and members are coerced from shorthand", func(t *testing.T) {
		got, err := buildCreateBody("web", "HTTP", "ROUND_ROBIN", true, false,
			"healthCheckProtocol=HTTP,healthCheckPath=/healthz,healthCheckMethod=GET,successCode=200,"+
				"httpVersion=1.0,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3",
			[]string{"ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false",
				"ipAddress=10.0.0.5,port=8080,monitorPort=8080"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hm := got["healthMonitor"].(map[string]interface{})
		if hm["timeout"] != 5 || hm["interval"] != 30 {
			t.Errorf("health monitor ints not coerced: %#v", hm)
		}
		members := got["members"].([]map[string]interface{})
		if len(members) != 2 {
			t.Fatalf("members length = %d, want 2", len(members))
		}
		if members[0]["port"] != 80 || members[0]["backup"] != false {
			t.Errorf("member[0] not coerced: %#v", members[0])
		}
		if got["stickiness"] != true || got["tlsEncryption"] != false {
			t.Errorf("booleans not carried through: %v", got)
		}
	})

	t.Run("members are omitted when none are given", func(t *testing.T) {
		got, _ := buildCreateBody("web", "TCP", "SOURCE_IP", false, false, completeTCPMonitor, nil)
		if _, ok := got["members"]; ok {
			t.Error("members should be absent when no --member was passed")
		}
	})

	t.Run("an empty health monitor is rejected", func(t *testing.T) {
		if _, err := buildCreateBody("web", "TCP", "SOURCE_IP", false, false, "", nil); err == nil {
			t.Error("expected an error for an empty --health-monitor")
		}
	})

	t.Run("a malformed flag is reported before any semantic complaint", func(t *testing.T) {
		_, err := buildCreateBody("web", "TCP", "SOURCE_IP", false, false,
			completeTCPMonitor, []string{"10.0.0.4:80"})
		if err == nil || !strings.Contains(err.Error(), "expected key=value") {
			t.Errorf("error = %v, want the shorthand parse error", err)
		}
	})
}

// currentPool is a trimmed PoolDto as returned by GET.
func currentPool() map[string]interface{} {
	return map[string]interface{}{
		"uuid":              "pool-1",
		"loadBalanceMethod": "ROUND_ROBIN",
		"stickiness":        true,
		"tlsEncryption":     false,
	}
}

// currentMonitor is the HMonitorDto from GET /pools/{id}/healthMonitor. It is a
// separate fixture because the pool payload does not include it — the live API
// returns only scalars there, which is what broke the first version of
// buildUpdateBody.
func currentMonitor() map[string]interface{} {
	return map[string]interface{}{
		"uuid":                "hm-1",
		"healthCheckProtocol": "HTTP",
		"healthCheckPath":     "/old",
		"healthCheckMethod":   "GET",
		"successCode":         "200",
		"httpVersion":         "1.0",
		"domainName":          "",
		"healthyThreshold":    float64(3),
		"unhealthyThreshold":  float64(3),
		"timeout":             float64(5),
		"interval":            float64(30),
		"progressStatus":      "DONE",
		"displayStatus":       "HEALTHY",
	}
}

func TestBuildUpdateBody(t *testing.T) {
	t.Run("unset fields are carried over from the current pool", func(t *testing.T) {
		changed := map[string]bool{"stickiness": true}
		got, err := buildUpdateBody(currentPool(), currentMonitor(), changed, "", false, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["algorithm"] != "ROUND_ROBIN" {
			t.Errorf("algorithm = %v, want the current ROUND_ROBIN", got["algorithm"])
		}
		if got["stickiness"] != false {
			t.Errorf("stickiness = %v, want the user's false", got["stickiness"])
		}
		if got["tlsEncryption"] != false {
			t.Errorf("tlsEncryption = %v, want the current false", got["tlsEncryption"])
		}
	})

	t.Run("the health monitor keeps its required thresholds and drops read-only fields", func(t *testing.T) {
		changed := map[string]bool{"health-monitor": true}
		got, err := buildUpdateBody(currentPool(), currentMonitor(), changed, "", false, false, "timeout=10,healthCheckPath=/new")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hm := got["healthMonitor"].(map[string]interface{})
		if hm["timeout"] != 10 {
			t.Errorf("timeout = %#v, want the override 10", hm["timeout"])
		}
		if hm["healthCheckPath"] != "/new" {
			t.Errorf("healthCheckPath = %v, want /new", hm["healthCheckPath"])
		}
		for _, required := range []string{"healthyThreshold", "unhealthyThreshold", "interval"} {
			if _, ok := hm[required]; !ok {
				t.Errorf("required field %q was dropped; the API rejects a partial monitor", required)
			}
		}
		for _, readOnly := range []string{"uuid", "progressStatus", "displayStatus", "healthCheckProtocol"} {
			if _, ok := hm[readOnly]; ok {
				t.Errorf("read-only field %q must not be echoed back", readOnly)
			}
		}
	})

	t.Run("an explicit algorithm wins over the current one", func(t *testing.T) {
		got, _ := buildUpdateBody(currentPool(), currentMonitor(), map[string]bool{"algorithm": true}, "SOURCE_IP", false, false, "")
		if got["algorithm"] != "SOURCE_IP" {
			t.Errorf("algorithm = %v, want SOURCE_IP", got["algorithm"])
		}
	})

	t.Run("an unknown current algorithm is an actionable error", func(t *testing.T) {
		if _, err := buildUpdateBody(map[string]interface{}{}, currentMonitor(), map[string]bool{"stickiness": true}, "", true, false, ""); err == nil {
			t.Error("expected an error when the algorithm cannot be determined")
		}
	})

	t.Run("malformed health-monitor shorthand is rejected", func(t *testing.T) {
		if _, err := buildUpdateBody(currentPool(), currentMonitor(), map[string]bool{"health-monitor": true}, "", false, false, "timeout=soon"); err == nil {
			t.Error("expected an error for a non-integer timeout")
		}
	})
}

func TestUpdatableHealthMonitorFieldsExcludeProtocol(t *testing.T) {
	for _, f := range updatableHealthMonitorFields {
		if f == "healthCheckProtocol" {
			t.Error("healthCheckProtocol is fixed at creation and must not be sent on update")
		}
	}
	want := []string{"healthCheckPath", "healthCheckMethod", "healthyThreshold", "unhealthyThreshold",
		"timeout", "interval", "successCode", "domainName", "httpVersion"}
	if !reflect.DeepEqual(updatableHealthMonitorFields, want) {
		t.Errorf("updatableHealthMonitorFields = %v, want %v", updatableHealthMonitorFields, want)
	}
}

// These rules are absent from openapi-vlb.json; they were found by exercising
// the live API, where breaking them yields either an HTTP 500 or a 400 naming a
// field the user never supplied. Validating client-side turns those into one
// actionable message.
func TestValidateHealthMonitor(t *testing.T) {
	full := func(over map[string]interface{}) map[string]interface{} {
		hm := map[string]interface{}{
			"healthCheckProtocol": "TCP",
			"timeout":             5,
			"interval":            30,
			"healthyThreshold":    3,
			"unhealthyThreshold":  3,
		}
		for k, v := range over {
			if v == nil {
				delete(hm, k)
			} else {
				hm[k] = v
			}
		}
		return hm
	}
	fullHTTP := func(over map[string]interface{}) map[string]interface{} {
		return full(mergeMaps(map[string]interface{}{
			"healthCheckProtocol": "HTTP",
			"healthCheckPath":     "/",
			"healthCheckMethod":   "GET",
			"successCode":         "200",
			"httpVersion":         "1.0",
		}, over))
	}

	tests := []struct {
		name    string
		hm      map[string]interface{}
		wantErr string
	}{
		{name: "complete TCP monitor", hm: full(nil)},
		{name: "complete HTTP monitor", hm: fullHTTP(nil)},
		{
			name:    "missing timeout (the API 500s on this)",
			hm:      full(map[string]interface{}{"timeout": nil}),
			wantErr: "timeout",
		},
		{
			name:    "bare protocol only (the API 500s on this)",
			hm:      map[string]interface{}{"healthCheckProtocol": "TCP"},
			wantErr: "timeout, interval, healthyThreshold, unhealthyThreshold",
		},
		{
			name:    "HTTP without its extra fields",
			hm:      full(map[string]interface{}{"healthCheckProtocol": "HTTP"}),
			wantErr: "healthCheckPath, healthCheckMethod, successCode, httpVersion",
		},
		{
			name:    "HTTPS needs the same extras as HTTP",
			hm:      full(map[string]interface{}{"healthCheckProtocol": "HTTPS"}),
			wantErr: "healthCheckPath",
		},
		{
			name:    "httpVersion 1.1 requires domainName",
			hm:      fullHTTP(map[string]interface{}{"httpVersion": "1.1"}),
			wantErr: "domainName is required when httpVersion is 1.1",
		},
		{
			name: "httpVersion 1.1 with domainName is fine",
			hm:   fullHTTP(map[string]interface{}{"httpVersion": "1.1", "domainName": "example.com"}),
		},
		{
			name:    "no protocol at all",
			hm:      full(map[string]interface{}{"healthCheckProtocol": nil}),
			wantErr: "healthCheckProtocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHealthMonitor(tt.hm)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func mergeMaps(base, over map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func TestValidateMembers(t *testing.T) {
	ok := map[string]interface{}{"ipAddress": "10.0.0.4", "port": 80, "monitorPort": 80}

	tests := []struct {
		name    string
		members []map[string]interface{}
		wantErr string
	}{
		{name: "no members is fine — a pool may start empty"},
		{name: "complete member", members: []map[string]interface{}{ok}},
		{
			name:    "missing monitorPort (the API 400s with a confusing message)",
			members: []map[string]interface{}{{"ipAddress": "10.0.0.4", "port": 80}},
			wantErr: "monitorPort is required",
		},
		{
			name:    "the error names the offending member",
			members: []map[string]interface{}{ok, {"ipAddress": "10.0.0.5", "port": 80}},
			wantErr: "--member[1]",
		},
		{
			name:    "missing ipAddress",
			members: []map[string]interface{}{{"port": 80, "monitorPort": 80}},
			wantErr: "ipAddress is required",
		},
		{
			name:    "missing port",
			members: []map[string]interface{}{{"ipAddress": "10.0.0.4", "monitorPort": 80}},
			wantErr: "port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMembers(tt.members)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// The first version of buildUpdateBody seeded the monitor from the pool payload,
// which the live API does not include — every pool update then failed with
// "If healthCheckProtocol field is HTTP, following fields must be specified".
func TestBuildUpdateBodyAlwaysSendsACompleteMonitor(t *testing.T) {
	t.Run("the monitor comes from its own endpoint, not the pool payload", func(t *testing.T) {
		got, err := buildUpdateBody(currentPool(), currentMonitor(),
			map[string]bool{"algorithm": true}, "LEAST_CONNECTIONS", false, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hm, ok := got["healthMonitor"].(map[string]interface{})
		if !ok {
			t.Fatal("healthMonitor must always be sent; the API rejects an update without it")
		}
		// The HTTP-protocol fields the backend demands must all be carried over.
		for _, required := range []string{"healthCheckPath", "healthCheckMethod", "successCode",
			"httpVersion", "timeout", "interval", "healthyThreshold", "unhealthyThreshold"} {
			if _, ok := hm[required]; !ok {
				t.Errorf("healthMonitor is missing %q", required)
			}
		}
	})

	t.Run("an empty domainName is not echoed back", func(t *testing.T) {
		got, _ := buildUpdateBody(currentPool(), currentMonitor(),
			map[string]bool{"algorithm": true}, "SOURCE_IP", false, false, "")
		hm := got["healthMonitor"].(map[string]interface{})
		if _, ok := hm["domainName"]; ok {
			t.Error("an empty domainName should be dropped, not sent")
		}
	})

	t.Run("an unreadable monitor is an actionable error, not a silent 400", func(t *testing.T) {
		_, err := buildUpdateBody(currentPool(), map[string]interface{}{},
			map[string]bool{"algorithm": true}, "SOURCE_IP", false, false, "")
		if err == nil || !strings.Contains(err.Error(), "current health monitor") {
			t.Errorf("error = %v, want it to explain that the monitor could not be read", err)
		}
	})
}
