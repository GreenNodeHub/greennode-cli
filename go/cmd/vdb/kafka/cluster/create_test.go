package cluster

import (
	"strings"
	"testing"
)

// Both authentication modes off is rejected client-side: the gateway refuses the
// combination with a bare "Need to enable at least one authentication mode" that
// names no field (measured live), so the error here names the flags instead.
func TestCreateRequiresAuthenticationMode(t *testing.T) {
	f := createCmd.Flags()
	f.Set("volume-size", "100")   //nolint:errcheck — past the earlier bounds checks
	f.Set("mtls-authen", "false") //nolint:errcheck
	f.Set("sasl-authen", "false") //nolint:errcheck

	err := runCreate(createCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--mtls-authen") ||
		!strings.Contains(err.Error(), "--sasl-authen") {
		t.Fatalf("expected the both-off error to name the flags, got %v", err)
	}
}

// An explicit --vserver-project-id wins without reading the profile, so the flag
// never depends on what the local config happens to contain.
func TestVserverProjectIDExplicitFlag(t *testing.T) {
	createCmd.Flags().Set("vserver-project-id", "pro-123") //nolint:errcheck

	got, err := vserverProjectID(createCmd)
	if err != nil {
		t.Fatalf("vserverProjectID: %v", err)
	}
	if got != "pro-123" {
		t.Errorf("vserverProjectID = %q, want pro-123", got)
	}
}
