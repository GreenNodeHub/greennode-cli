package config

import (
	"strings"
	"testing"
)

// vdb is only offered in HCM-3 today. Pin both halves of that: the endpoint
// resolves where the service exists, and fails with a message naming the service
// and region where it does not — so a HAN user gets a real answer instead of a
// confusing auth or 404 error deeper in the stack.
func TestGetEndpointVdb(t *testing.T) {
	cfg := &Config{Region: "HCM-3", Regions: REGIONS}
	got, err := cfg.GetEndpoint("vdb")
	if err != nil {
		t.Fatalf("GetEndpoint(vdb) in HCM-3 returned error: %v", err)
	}
	if got != "https://vdb-gateway.vngcloud.vn" {
		t.Errorf("vdb endpoint in HCM-3 = %q, want https://vdb-gateway.vngcloud.vn", got)
	}

	cfg = &Config{Region: "HAN", Regions: REGIONS}
	_, err = cfg.GetEndpoint("vdb")
	if err == nil {
		t.Fatal("GetEndpoint(vdb) in HAN succeeded, want an error (vdb is not offered there)")
	}
	if !strings.Contains(err.Error(), "vdb") || !strings.Contains(err.Error(), "HAN") {
		t.Errorf("error = %q, want it to name both the service and the region", err)
	}
}
