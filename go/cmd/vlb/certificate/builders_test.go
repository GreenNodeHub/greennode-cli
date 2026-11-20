package certificate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildCreateBody(t *testing.T) {
	t.Run("a TLS/SSL certificate carries its key and omits blank optionals", func(t *testing.T) {
		got, err := buildCreateBody("web-cert", typeTLSSSL, "CERT-PEM", "KEY-PEM", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["privateKey"] != "KEY-PEM" || got["certificate"] != "CERT-PEM" {
			t.Errorf("body = %v", got)
		}
		for _, key := range []string{"certificateChain", "passphrase"} {
			if _, ok := got[key]; ok {
				t.Errorf("blank %q should be omitted, not sent empty", key)
			}
		}
	})

	t.Run("a CA certificate needs no private key", func(t *testing.T) {
		if _, err := buildCreateBody("root-ca", typeCA, "CERT-PEM", "", "", ""); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		for _, tc := range []struct {
			name, certType, cert, key string
		}{
			{"unknown type", "SOMETHING", "CERT", "KEY"},
			{"missing certificate", typeCA, "   ", ""},
			{"TLS/SSL without a private key", typeTLSSSL, "CERT", "  "},
		} {
			if _, err := buildCreateBody("c", tc.certType, tc.cert, tc.key, "", ""); err == nil {
				t.Errorf("%s: expected a validation error", tc.name)
			}
		}
	})
}

func TestReadMaterial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(path, []byte("FILE-PEM"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := readMaterial("certificate", "", path); err != nil || got != "FILE-PEM" {
		t.Errorf("readMaterial(file) = %q, %v", got, err)
	}
	if got, err := readMaterial("certificate", "INLINE", ""); err != nil || got != "INLINE" {
		t.Errorf("readMaterial(inline) = %q, %v", got, err)
	}
	_, err := readMaterial("certificate", "", filepath.Join(dir, "missing.pem"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !strings.Contains(err.Error(), "--certificate-file") {
		t.Errorf("error should name the flag, got: %v", err)
	}
}

func TestRedactedPreviewHidesSecrets(t *testing.T) {
	body := map[string]interface{}{
		"name":             "web-cert",
		"type":             typeTLSSSL,
		"certificate":      "-----BEGIN CERTIFICATE-----abc",
		"certificateChain": "-----BEGIN CERTIFICATE-----chain",
		"privateKey":       "-----BEGIN PRIVATE KEY-----supersecret",
		"passphrase":       "hunter2",
	}
	preview := redactedPreview(body)

	if preview["privateKey"] != "<redacted>" || preview["passphrase"] != "<redacted>" {
		t.Errorf("secrets leaked into the dry-run preview: %v", preview)
	}
	for _, key := range []string{"certificate", "certificateChain"} {
		s, _ := preview[key].(string)
		if !strings.HasPrefix(s, "<") || !strings.HasSuffix(s, "bytes of PEM>") {
			t.Errorf("%s = %q, want a size summary rather than the PEM body", key, s)
		}
	}
	if preview["name"] != "web-cert" || preview["type"] != typeTLSSSL {
		t.Errorf("non-secret fields should pass through: %v", preview)
	}
	// The original body must not be mutated — it is what gets POSTed.
	if body["privateKey"] != "-----BEGIN PRIVATE KEY-----supersecret" {
		t.Error("redactedPreview must not mutate the request body")
	}
}

// TestListDoesNotOfferPaginationFlags mirrors the loadbalancer decision: the
// spec declares page/size on GET /cas with defaults 1/10, but the live API
// ignores them and returns every certificate.
func TestListDoesNotOfferPaginationFlags(t *testing.T) {
	for _, name := range []string{"page", "page-size"} {
		if listCmd.Flags().Lookup(name) != nil {
			t.Errorf("`certificate list` declares --%s, but the API ignores it", name)
		}
	}
	if listCmd.Flags().Lookup("name") == nil {
		t.Error("`certificate list` should offer --name (the API honours it)")
	}
}

// The list table carries no expiry column: expiredAt is null in practice, and
// notAfter is epoch milliseconds that the table layer cannot format. Expiry is
// surfaced by `certificate get` and by the delete preview instead.
func TestCertificateTableHasNoExpiryColumn(t *testing.T) {
	for _, col := range tableColumns {
		if col == "expiredAt" {
			t.Error("expiredAt is null in practice; it renders a blank column that hides expired certificates")
		}
		if col == "notAfter" {
			t.Error("notAfter is epoch milliseconds and the table layer cannot format it")
		}
	}
}

func TestExpiryOfPrefersNotAfterAndFlagsExpiry(t *testing.T) {
	past := float64(time.Now().Add(-48 * time.Hour).UnixMilli())
	got := expiryOf(map[string]interface{}{"notAfter": past, "expiredAt": nil})
	if !strings.Contains(got, "EXPIRED") {
		t.Errorf("expiryOf = %q, want it to flag an expired certificate", got)
	}

	future := float64(time.Now().Add(48 * time.Hour).UnixMilli())
	got = expiryOf(map[string]interface{}{"notAfter": future})
	if strings.Contains(got, "EXPIRED") {
		t.Errorf("expiryOf = %q, want no EXPIRED marker for a valid certificate", got)
	}

	if got := expiryOf(map[string]interface{}{}); got != "unknown" {
		t.Errorf("expiryOf with neither field = %q, want \"unknown\"", got)
	}
}
