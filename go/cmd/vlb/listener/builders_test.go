package listener

import (
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestBuildCreateBody(t *testing.T) {
	t.Run("required fields and timeouts are always sent", func(t *testing.T) {
		got, err := buildCreateBody("web", "HTTP", 80, 50, 50, 5, listenerOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]interface{}{
			"listenerName": "web", "listenerProtocol": "HTTP", "listenerProtocolPort": 80,
			"timeoutClient": 50, "timeoutMember": 50, "timeoutConnection": 5,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v, want %v", got, want)
		}
	})

	t.Run("comma-separated lists and repeatable headers are parsed", func(t *testing.T) {
		got, err := buildCreateBody("web", "HTTPS", 443, 50, 50, 5, listenerOptions{
			DefaultPoolID:    "pool-1",
			CertAuthorities:  "ca-1, ca-2",
			ALPNProtocols:    "h2,http/1.1",
			InsertHeaders:    []string{"headerName=X-Forwarded-For,headerValue=true"},
			TLSSecurityPolcy: "MODERN",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got["certificateAuthorities"], []string{"ca-1", "ca-2"}) {
			t.Errorf("certificateAuthorities = %#v", got["certificateAuthorities"])
		}
		if !reflect.DeepEqual(got["alpnProtocols"], []string{"h2", "http/1.1"}) {
			t.Errorf("alpnProtocols = %#v", got["alpnProtocols"])
		}
		headers := got["insertHeaders"].([]map[string]interface{})
		if len(headers) != 1 || headers[0]["headerName"] != "X-Forwarded-For" {
			t.Errorf("insertHeaders = %#v", headers)
		}
		if got["tlsSecurityPolicy"] != "MODERN" || got["defaultPoolId"] != "pool-1" {
			t.Errorf("optional strings not carried through: %v", got)
		}
	})

	t.Run("blank optional strings are omitted, not sent empty", func(t *testing.T) {
		got, _ := buildCreateBody("web", "HTTP", 80, 50, 50, 5, listenerOptions{})
		for _, key := range []string{"defaultPoolId", "allowedCidrs", "blockedCidrs", "defaultAction",
			"certificateAuthorities", "alpnProtocols", "insertHeaders", "tlsSecurityPolicy"} {
			if _, ok := got[key]; ok {
				t.Errorf("key %q should be absent when its flag is unset", key)
			}
		}
	})

	t.Run("out-of-range ports and timeouts are rejected", func(t *testing.T) {
		for _, tc := range []struct {
			name                             string
			port, client, member, connection int
		}{
			{"port 0", 0, 50, 50, 5},
			{"port 65536", 65536, 50, 50, 5},
			{"timeout-client 0", 80, 0, 50, 5},
			{"timeout-member 3601", 80, 50, 3601, 5},
			{"timeout-connection -1", 80, 50, 50, -1},
		} {
			if _, err := buildCreateBody("web", "HTTP", tc.port, tc.client, tc.member, tc.connection, listenerOptions{}); err == nil {
				t.Errorf("%s: expected a validation error", tc.name)
			}
		}
	})
}

// currentListener is a trimmed ListenerDto as returned by GET.
func currentListener() map[string]interface{} {
	return map[string]interface{}{
		"uuid": "li-1", "name": "web", "protocol": "HTTPS", "protocolPort": float64(443),
		"timeoutClient": float64(50), "timeoutMember": float64(50), "timeoutConnection": float64(5),
		"defaultPoolId": "pool-1", "defaultAction": "accept",
		"clientCertificateAuthentication": "off",
		"certificateAuthorities":          []interface{}{"ca-1"},
		"alpnProtocols":                   []interface{}{"h2"},
		"insertHeaders":                   []interface{}{map[string]interface{}{"headerName": "X-A", "headerValue": "1"}},
		"displayStatus":                   "ACTIVE",
	}
}

// updateFlagSet builds the same flag set update.go registers, with the given
// flags marked as explicitly set.
func updateFlagSet(t *testing.T, set map[string]string) *pflag.FlagSet {
	t.Helper()
	f := pflag.NewFlagSet("update", pflag.ContinueOnError)
	addUpdateFlags(f)
	for name, value := range set {
		if err := f.Set(name, value); err != nil {
			t.Fatalf("Set(%q, %q): %v", name, value, err)
		}
	}
	return f
}

func TestBuildUpdateBody(t *testing.T) {
	t.Run("required timeouts are carried over when not overridden", func(t *testing.T) {
		got, err := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"default-pool-id": "pool-2"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["timeoutClient"] != float64(50) || got["timeoutConnection"] != float64(5) {
			t.Errorf("timeouts not carried over: %v", got)
		}
		if got["defaultPoolId"] != "pool-2" {
			t.Errorf("defaultPoolId = %v, want the override pool-2", got["defaultPoolId"])
		}
	})

	t.Run("clientCertificate is read from its differently-named DTO field", func(t *testing.T) {
		got, _ := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"timeout-client": "60"}))
		if got["clientCertificate"] != "off" {
			t.Errorf("clientCertificate = %v, want the current clientCertificateAuthentication 'off'", got["clientCertificate"])
		}
		if got["timeoutClient"] != 60 {
			t.Errorf("timeoutClient = %#v, want the override 60", got["timeoutClient"])
		}
	})

	t.Run("unset list fields keep their current values", func(t *testing.T) {
		got, _ := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"timeout-member": "70"}))
		if !reflect.DeepEqual(got["certificateAuthorities"], []interface{}{"ca-1"}) {
			t.Errorf("certificateAuthorities = %#v, want the current list", got["certificateAuthorities"])
		}
		if !reflect.DeepEqual(got["alpnProtocols"], []interface{}{"h2"}) {
			t.Errorf("alpnProtocols = %#v, want the current list", got["alpnProtocols"])
		}
	})

	t.Run("an explicitly empty list field clears it", func(t *testing.T) {
		got, _ := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"certificate-authorities": ""}))
		if got["certificateAuthorities"] == nil {
			t.Fatal("certificateAuthorities should be present (as empty), not absent")
		}
		if v, ok := got["certificateAuthorities"].([]string); !ok || len(v) != 0 {
			t.Errorf("certificateAuthorities = %#v, want an empty []string", got["certificateAuthorities"])
		}
	})

	t.Run("timeouts that cannot be determined are an actionable error", func(t *testing.T) {
		if _, err := buildUpdateBody(map[string]interface{}{}, updateFlagSet(t, map[string]string{"default-action": "drop"})); err == nil {
			t.Error("expected an error when the current timeouts are unknown")
		}
	})

	t.Run("an out-of-range timeout override is rejected", func(t *testing.T) {
		if _, err := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"timeout-client": "4000"})); err == nil {
			t.Error("expected an error for a timeout above 3600")
		}
	})
}

func TestUpdateFlagNamesCoverEveryBodyFlag(t *testing.T) {
	f := pflag.NewFlagSet("update", pflag.ContinueOnError)
	addUpdateFlags(f)
	declared := map[string]bool{}
	f.VisitAll(func(fl *pflag.Flag) { declared[fl.Name] = true })

	for _, name := range updateFlagNames() {
		if !declared[name] {
			t.Errorf("updateFlagNames() lists %q, which addUpdateFlags does not declare", name)
		}
		delete(declared, name)
	}
	for name := range declared {
		t.Errorf("flag %q is declared but missing from updateFlagNames(); "+
			"setting it alone would fail the 'nothing to update' check", name)
	}
}

// currentHTTPListener mirrors what the live API returns for a plain HTTP
// listener: the TLS fields are present but empty. Echoing them back is what the
// service rejects with "Cannot specify client certificate, certificate
// authorities, default certificate authority, or tls security policy for http
// listener" — which broke every HTTP listener update.
func currentHTTPListener() map[string]interface{} {
	return map[string]interface{}{
		"uuid": "li-2", "name": "http80", "protocol": "HTTP", "protocolPort": float64(80),
		"timeoutClient": float64(60), "timeoutMember": float64(60), "timeoutConnection": float64(5),
		"defaultPoolId": "pool-1", "defaultPoolName": nil, "defaultAction": "accept",
		"alpnProtocols": []interface{}{}, "certificateAuthorities": []interface{}{},
		"clientCertificateAuthentication": nil, "defaultCertificateAuthority": nil,
		"tlsSecurityPolicy": nil,
		"insertHeaders":     []interface{}{map[string]interface{}{"headerName": "X-Forwarded-For", "headerValue": "true"}},
	}
}

func TestBuildUpdateBodyOmitsTLSFieldsOnPlainHTTP(t *testing.T) {
	tlsKeys := []string{"certificateAuthorities", "alpnProtocols", "tlsSecurityPolicy",
		"defaultCertificateAuthority", "clientCertificate"}

	t.Run("an HTTP listener sends no TLS field at all", func(t *testing.T) {
		got, err := buildUpdateBody(currentHTTPListener(), updateFlagSet(t, map[string]string{"timeout-client": "120"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, k := range tlsKeys {
			if _, ok := got[k]; ok {
				t.Errorf("%q must not be sent for an HTTP listener (the API rejects it)", k)
			}
		}
		if got["timeoutClient"] != 120 {
			t.Errorf("timeoutClient = %#v, want the override 120", got["timeoutClient"])
		}
		if got["defaultPoolId"] != "pool-1" {
			t.Errorf("defaultPoolId should still be carried over, got %#v", got["defaultPoolId"])
		}
	})

	t.Run("an explicit TLS flag is still honoured on HTTP so the API can rule on it", func(t *testing.T) {
		got, _ := buildUpdateBody(currentHTTPListener(), updateFlagSet(t, map[string]string{"tls-security-policy": "MODERN"}))
		if got["tlsSecurityPolicy"] != "MODERN" {
			t.Errorf("an explicitly-set flag should be sent, got %#v", got["tlsSecurityPolicy"])
		}
	})

	t.Run("an HTTPS listener still carries its TLS state", func(t *testing.T) {
		got, _ := buildUpdateBody(currentListener(), updateFlagSet(t, map[string]string{"timeout-client": "120"}))
		if !reflect.DeepEqual(got["certificateAuthorities"], []interface{}{"ca-1"}) {
			t.Errorf("certificateAuthorities = %#v, want the current list preserved", got["certificateAuthorities"])
		}
		if got["clientCertificate"] != "off" {
			t.Errorf("clientCertificate = %#v, want it preserved", got["clientCertificate"])
		}
	})

	t.Run("empty TLS lists are never echoed, even on HTTPS", func(t *testing.T) {
		cur := currentListener()
		cur["certificateAuthorities"] = []interface{}{}
		cur["alpnProtocols"] = []interface{}{}
		got, _ := buildUpdateBody(cur, updateFlagSet(t, map[string]string{"timeout-client": "120"}))
		for _, k := range []string{"certificateAuthorities", "alpnProtocols"} {
			if _, ok := got[k]; ok {
				t.Errorf("an empty %q should be dropped, not sent", k)
			}
		}
	})
}

// The service does populate defaultPoolName when a default pool is attached — an
// earlier comment here claimed it was always null, so the table showed a raw
// UUID instead. Verified live: the service returns the pool's name here.
func TestListenerTableShowsThePoolName(t *testing.T) {
	for _, col := range tableColumns {
		if col == "defaultPoolId" {
			t.Error("table shows defaultPoolId; defaultPoolName is populated and reads better")
		}
	}
	found := false
	for _, col := range tableColumns {
		if col == "defaultPoolName" {
			found = true
		}
	}
	if !found {
		t.Error("table is missing defaultPoolName")
	}
}
