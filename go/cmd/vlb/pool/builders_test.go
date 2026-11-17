package pool

import (
	"reflect"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/cli"
)

// completeTCPMonitor carries every field the live API requires; the spec marks
// only healthCheckProtocol as required, but the rest are enforced in practice.
const completeTCPMonitor = "healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3"

// completePingUDPMonitor is the only monitor a UDP pool accepts.
const completePingUDPMonitor = "healthCheckProtocol=PING-UDP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3"

func TestBuildCreateBody(t *testing.T) {
	t.Run("health monitor and members are coerced from shorthand", func(t *testing.T) {
		got, err := buildCreateBody("web", "HTTP", "ROUND_ROBIN", true, false, true, true,
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
		got, _ := buildCreateBody("web", "TCP", "SOURCE_IP", false, false, false, false, completeTCPMonitor, nil)
		if _, ok := got["members"]; ok {
			t.Error("members should be absent when no --member was passed")
		}
	})

	t.Run("an empty health monitor is rejected", func(t *testing.T) {
		if _, err := buildCreateBody("web", "TCP", "SOURCE_IP", false, false, false, false, "", nil); err == nil {
			t.Error("expected an error for an empty --health-monitor")
		}
	})

	t.Run("a malformed flag is reported before any semantic complaint", func(t *testing.T) {
		_, err := buildCreateBody("web", "TCP", "SOURCE_IP", false, false, false, false,
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
		"protocol":          "HTTP",
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

	t.Run("switching to httpVersion 1.0 drops the carried-over domainName", func(t *testing.T) {
		monitor := currentMonitor()
		monitor["httpVersion"] = "1.1"
		monitor["domainName"] = "app.example.com"
		got, err := buildUpdateBody(currentPool(), monitor, map[string]bool{"health-monitor": true}, "", false, false, "httpVersion=1.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hm := got["healthMonitor"].(map[string]interface{})
		if _, ok := hm["domainName"]; ok {
			t.Errorf("domainName = %v, want it removed for httpVersion 1.0", hm["domainName"])
		}
	})

	t.Run("an empty override removes the field", func(t *testing.T) {
		monitor := currentMonitor()
		monitor["httpVersion"] = "1.1"
		monitor["domainName"] = "app.example.com"
		got, err := buildUpdateBody(currentPool(), monitor, map[string]bool{"health-monitor": true}, "", false, false, "httpVersion=1.0,domainName=")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hm := got["healthMonitor"].(map[string]interface{})
		if _, ok := hm["domainName"]; ok {
			t.Errorf("domainName = %#v, want it removed by domainName=", hm["domainName"])
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
		name string
		// poolProtocol drives the UDP/PING-UDP pairing rule; TCP is the neutral
		// default for the cases that are not about pairing.
		poolProtocol string
		hm           map[string]interface{}
		wantErr      string
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
			// validateMethod rejects a nil method for HTTP and HTTPS alike, but
			// only the HTTP branch of validateHealthCheck demands path,
			// successCode and httpVersion. Requiring all four for HTTPS refused
			// requests the API accepts.
			name:    "HTTPS needs only the method",
			hm:      full(map[string]interface{}{"healthCheckProtocol": "HTTPS"}),
			wantErr: "healthCheckMethod",
		},
		{
			name: "HTTPS with just the method is accepted",
			hm:   full(map[string]interface{}{"healthCheckProtocol": "HTTPS", "healthCheckMethod": "GET"}),
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

		// "If healthCheckProtocol field is TCP or PING-UDP, following fields
		// cannot be specified: healthCheckPath, healthCheckMethod, successCode,
		// domainName, httpVersion"
		{
			name:    "TCP monitor rejects an HTTP-only field",
			hm:      full(map[string]interface{}{"healthCheckPath": "/healthz"}),
			wantErr: "healthCheckPath cannot be set when healthCheckProtocol is TCP",
		},
		{
			name: "TCP monitor names every offending field at once",
			hm: full(map[string]interface{}{
				"healthCheckPath": "/", "successCode": "200", "httpVersion": "1.0",
			}),
			wantErr: "healthCheckPath, successCode, httpVersion cannot be set",
		},
		{
			name:         "PING-UDP monitor rejects an HTTP-only field",
			poolProtocol: "UDP",
			hm: full(map[string]interface{}{
				"healthCheckProtocol": "PING-UDP", "domainName": "example.com",
			}),
			wantErr: "domainName cannot be set when healthCheckProtocol is PING-UDP",
		},

		// "Domain name cannot be specified for Http version 1.0 health check
		// protocol"
		{
			name:    "httpVersion 1.0 rejects domainName",
			hm:      fullHTTP(map[string]interface{}{"httpVersion": "1.0", "domainName": "example.com"}),
			wantErr: "domainName cannot be set when httpVersion is 1.0",
		},

		// validateMethod / validateHttpVersion enums
		{
			name:    "an unsupported health check method is rejected",
			hm:      fullHTTP(map[string]interface{}{"healthCheckMethod": "DELETE"}),
			wantErr: "healthCheckMethod must be one of GET, POST, PUT",
		},
		{
			name:    "an unsupported httpVersion is rejected",
			hm:      fullHTTP(map[string]interface{}{"httpVersion": "2.0"}),
			wantErr: "httpVersion must be one of 1.0, 1.1",
		},

		// "Timeout must be less than interval"
		{
			name:    "timeout equal to interval is rejected",
			hm:      full(map[string]interface{}{"timeout": 30, "interval": 30}),
			wantErr: "timeout must be less than interval",
		},
		{
			name:    "timeout greater than interval is rejected",
			hm:      full(map[string]interface{}{"timeout": 60, "interval": 30}),
			wantErr: "timeout must be less than interval",
		},

		// validateHealthCheckProtocol's UDP pairing
		{
			name:         "a UDP pool requires PING-UDP",
			poolProtocol: "UDP",
			hm:           full(nil),
			wantErr:      "healthCheckProtocol must be PING-UDP for a UDP pool",
		},
		{
			name:         "a UDP pool with PING-UDP is accepted",
			poolProtocol: "UDP",
			hm:           full(map[string]interface{}{"healthCheckProtocol": "PING-UDP"}),
		},
		{
			name:         "PING-UDP on a non-UDP pool is rejected",
			poolProtocol: "PROXY",
			hm:           full(map[string]interface{}{"healthCheckProtocol": "PING-UDP"}),
			wantErr:      "PING-UDP is only valid for a UDP pool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolProtocol := tt.poolProtocol
			if poolProtocol == "" {
				poolProtocol = "TCP"
			}
			err := validateHealthMonitor(poolProtocol, tt.hm)
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

// The backend tests stickiness/tlsEncryption with != null
// (CreatePoolValidator.validateNonPoolConfiguration), so false is as fatal as
// true: an HTTP pool must carry both and a non-HTTP pool must carry neither.
// Sending them unconditionally made every pool on a Layer 4 load balancer
// impossible to create, and made every non-HTTP pool impossible to update.
func TestStickinessIsGatedOnThePoolProtocol(t *testing.T) {
	t.Run("create omits both for every non-HTTP protocol", func(t *testing.T) {
		// A UDP pool must pair with a PING-UDP health check, so the monitor has
		// to match the pool protocol here.
		monitors := map[string]string{
			"TCP":   completeTCPMonitor,
			"PROXY": completeTCPMonitor,
			"UDP":   completePingUDPMonitor,
		}
		for _, protocol := range []string{"TCP", "UDP", "PROXY"} {
			got, err := buildCreateBody("web", protocol, "ROUND_ROBIN", false, false, false, false,
				monitors[protocol], nil)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", protocol, err)
			}
			for _, key := range []string{"stickiness", "tlsEncryption"} {
				if _, ok := got[key]; ok {
					t.Errorf("%s pool carries %q; the API rejects it for non-HTTP pools", protocol, key)
				}
			}
		}
	})

	t.Run("create sends both for HTTP, which requires them", func(t *testing.T) {
		got, err := buildCreateBody("web", "HTTP", "ROUND_ROBIN", false, false, false, false,
			completeTCPMonitor, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, key := range []string{"stickiness", "tlsEncryption"} {
			if _, ok := got[key]; !ok {
				t.Errorf("HTTP pool is missing %q; the API requires it", key)
			}
		}
	})

	t.Run("create refuses the flags instead of dropping them silently", func(t *testing.T) {
		for _, tc := range []struct {
			name                            string
			stickinessSet, tlsEncryptionSet bool
			want                            string
		}{
			{"stickiness", true, false, "--stickiness is only valid for an HTTP pool"},
			{"tls encryption", false, true, "--tls-encryption is only valid for an HTTP pool"},
			{"both", true, true, "--stickiness and --tls-encryption are only valid for an HTTP pool"},
		} {
			_, err := buildCreateBody("web", "TCP", "ROUND_ROBIN", true, true,
				tc.stickinessSet, tc.tlsEncryptionSet, completeTCPMonitor, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("%s: error = %v, want it to contain %q", tc.name, err, tc.want)
			}
			if err != nil && !strings.Contains(err.Error(), "TCP") {
				t.Errorf("%s: error %q should name the offending protocol", tc.name, err)
			}
		}
	})

	t.Run("update drops the service's own values on a non-HTTP pool", func(t *testing.T) {
		// GET /pools/{id} reports stickiness and tlsEncryption as false for
		// every pool regardless of protocol. Echoing that back at a TCP pool is
		// what the API refuses, so seeding from current state must not happen.
		current := currentPool()
		current["protocol"] = "TCP"
		current["stickiness"] = false
		current["tlsEncryption"] = false

		got, err := buildUpdateBody(current, currentMonitor(), map[string]bool{"algorithm": true},
			"SOURCE_IP", false, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, key := range []string{"stickiness", "tlsEncryption"} {
			if _, ok := got[key]; ok {
				t.Errorf("TCP pool update carries %q echoed from GET; the API rejects it", key)
			}
		}
	})

	t.Run("update refuses the flags on a non-HTTP pool", func(t *testing.T) {
		current := currentPool()
		current["protocol"] = "PROXY"

		_, err := buildUpdateBody(current, currentMonitor(), map[string]bool{"stickiness": true},
			"", true, false, "")
		if err == nil || !strings.Contains(err.Error(), "only valid for an HTTP pool") {
			t.Fatalf("error = %v, want a refusal naming the HTTP-only restriction", err)
		}
		if !strings.Contains(err.Error(), "PROXY") {
			t.Errorf("error %q should name the pool's protocol", err)
		}
	})

	t.Run("update still carries both on an HTTP pool", func(t *testing.T) {
		got, err := buildUpdateBody(currentPool(), currentMonitor(), map[string]bool{"algorithm": true},
			"SOURCE_IP", false, false, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["stickiness"] != true {
			t.Errorf("stickiness = %v, want the current true carried over", got["stickiness"])
		}
		if _, ok := got["tlsEncryption"]; !ok {
			t.Error("tlsEncryption was dropped from an HTTP pool update")
		}
	})
}

// backup is a boxed @NotNull Boolean on MemberRequest, so omitting it returned
// "members[0].backup: must not be null" — a 400 naming a field the user never
// set. Its default is unambiguous, unlike monitorPort's.
func TestMemberBackupDefaultsToFalse(t *testing.T) {
	got, err := buildCreateBody("web", "TCP", "ROUND_ROBIN", false, false, false, false,
		completeTCPMonitor, []string{"ipAddress=10.0.0.4,port=80,monitorPort=80"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	members := got["members"].([]map[string]interface{})
	if got, ok := members[0]["backup"]; !ok || got != false {
		t.Errorf("backup = %#v (present=%v), want false to be filled in", got, ok)
	}
}

func TestMemberBackupIsNotOverwrittenWhenSet(t *testing.T) {
	got, err := buildCreateBody("web", "TCP", "ROUND_ROBIN", false, false, false, false,
		completeTCPMonitor, []string{"ipAddress=10.0.0.4,port=80,monitorPort=80,backup=true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	members := got["members"].([]map[string]interface{})
	if members[0]["backup"] != true {
		t.Errorf("backup = %#v, want the user's true preserved", members[0]["backup"])
	}
}

// The --health-monitor example in the help must be accepted by the CLI's own
// validation; the old one was missing five required fields.
func TestHelpHealthMonitorExampleIsValid(t *testing.T) {
	hm, err := cli.ParseStructFlagTyped(exampleHealthMonitor, healthMonitorIntFields, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := validateHealthMonitor("TCP", hm); err != nil {
		t.Errorf("help example %q is rejected: %v", exampleHealthMonitor, err)
	}
}
