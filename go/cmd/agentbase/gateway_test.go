package agentbase

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
)

// captureStdout swaps os.Stdout for a pipe and returns everything fn wrote.
// Used for `generate`, which writes the template directly to stdout (not via
// cobra's OutOrStdout), so the only way to assert it in-process is to swap fd 1.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

// TestGatewayGenerate_PrintsYAMLTemplate verifies generate (default format)
// emits the commented YAML template to stdout.
func TestGatewayGenerate_PrintsYAMLTemplate(t *testing.T) {
	output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := gatewayGenerateCmd.RunE(gatewayGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	if !strings.Contains(got, "name: my-gateway") || !strings.Contains(got, "networkMode: PUBLIC") {
		t.Errorf("YAML template missing expected keys; got:\n%s", got)
	}
}

// TestGatewayGenerate_PrintsJSONSkeleton verifies -o json prints a JSON skeleton
// that round-trips through loadCreateSpec.
func TestGatewayGenerate_PrintsJSONSkeleton(t *testing.T) {
	output.SetFormat(output.FormatJSON)
	defer output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := gatewayGenerateCmd.RunE(gatewayGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	if !strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Errorf("expected JSON skeleton; got:\n%s", got)
	}
}

// TestGatewayGenerate_RoundTripsThroughLoadCreateSpec proves the template that
// `generate` prints is itself a valid, loadable create spec (the core UX claim:
// generate > file.yaml > edit > create --file file.yaml).
func TestGatewayGenerate_RoundTripsThroughLoadCreateSpec(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := gatewayGenerateCmd.RunE(gatewayGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	req, err := loadCreateSpec([]byte(template))
	if err != nil {
		t.Fatalf("generated template is not a valid spec: %v", err)
	}
	if req.Name != "my-gateway" || req.NetworkMode != "PUBLIC" || req.FlavorID != "fill-flavor-id" {
		t.Errorf("round-trip decoded unexpected spec: %+v", req)
	}
	if req.Replicas != 1 || req.InboundAuth.Mode != "NONE" {
		t.Errorf("round-trip decoded unexpected replicas/inbound: %+v", req)
	}
}

// TestLoadCreateSpec_NestedCamelCaseDecodes exercises the YAML→map→JSON→struct
// bridge on nested camelCase keys. yaml.v3 does NOT honor json tags, so the
// bridge (yaml.Unmarshal into map → json.Marshal → json.Unmarshal into struct)
// is what makes the file's camelCase keys bind correctly. This is the load-bearing
// test for that trick: targets[].outboundAuth.scopes, inboundAuth.jwt.*,
// privateNetwork.publicEndpointEnabled, allowedCidrs, hostAliases all must land
// in the right struct fields.
func TestLoadCreateSpec_NestedCamelCaseDecodes(t *testing.T) {
	spec := []byte(`
name: nested-gw
networkMode: PRIVATE
flavorId: f1
replicas: 2
inboundAuth:
  mode: JWT
  jwt:
    source: DISCOVERY
    discoveryUrl: https://idp.example.com/.well-known/openid-configuration
    allowedAudiences:
      - aud1
    allowedClients:
      - client1
    principalClaim: sub
privateNetwork:
  vpcId: vpc-1
  subnetId: sub-1
  routes:
    - 172.16.0.0/12
  publicEndpointEnabled: true
targets:
  - name: weather
    type: MCP
    endpoint: https://mcp.example.com
    outboundAuth:
      type: OAUTH
      providerSource: CUSTOM
      flow: 2LO
      scopes:
        - read
        - write
allowedCidrs:
  - 10.0.0.0/8
hostAliases:
  - ip: 10.0.0.1
    hostnames:
      - foo.local
      - bar.local
`)
	req, err := loadCreateSpec(spec)
	if err != nil {
		t.Fatalf("loadCreateSpec: %v", err)
	}

	// Top-level sealed fields.
	if req.Name != "nested-gw" || req.NetworkMode != "PRIVATE" || req.FlavorID != "f1" || req.Replicas != 2 {
		t.Errorf("top-level mismatch: %+v", req)
	}

	// Nested inboundAuth.jwt (camelCase bridge).
	if req.InboundAuth.Mode != "JWT" || req.InboundAuth.JWT == nil {
		t.Fatalf("inbound jwt not decoded: %+v", req.InboundAuth)
	}
	jwt := req.InboundAuth.JWT
	if jwt.Source != "DISCOVERY" || jwt.DiscoveryURL == "" || jwt.PrincipalClaim != "sub" {
		t.Errorf("jwt fields mismatch: %+v", jwt)
	}
	if len(jwt.AllowedAudiences) != 1 || jwt.AllowedAudiences[0] != "aud1" {
		t.Errorf("allowedAudiences mismatch: %+v", jwt.AllowedAudiences)
	}

	// privateNetwork (camelCase + bool).
	if req.PrivateNetwork == nil {
		t.Fatal("privateNetwork not decoded")
	}
	pn := req.PrivateNetwork
	if pn.VPCID != "vpc-1" || pn.SubnetID != "sub-1" {
		t.Errorf("privateNetwork vpc/subnet mismatch: %+v", pn)
	}
	if len(pn.Routes) != 1 || pn.Routes[0] != "172.16.0.0/12" {
		t.Errorf("routes mismatch: %+v", pn.Routes)
	}
	if !pn.PublicEndpointEnabled {
		t.Error("publicEndpointEnabled should be true")
	}

	// targets[].outboundAuth (deeply nested camelCase + pointer optional).
	if len(req.Targets) != 1 {
		t.Fatalf("targets mismatch: %+v", req.Targets)
	}
	oa := req.Targets[0].OutboundAuth
	if oa.Type != "OAUTH" || oa.Flow != "2LO" {
		t.Errorf("outboundAuth type/flow mismatch: %+v", oa)
	}
	if oa.ProviderSource == nil || *oa.ProviderSource != "CUSTOM" {
		t.Errorf("providerSource mismatch: %+v", oa.ProviderSource)
	}
	if len(oa.Scopes) != 2 || oa.Scopes[0] != "read" || oa.Scopes[1] != "write" {
		t.Errorf("scopes mismatch: %+v", oa.Scopes)
	}

	// allowedCidrs (nullable slice pointer).
	if req.AllowedCIDRs == nil || len(*req.AllowedCIDRs) != 1 || (*req.AllowedCIDRs)[0] != "10.0.0.0/8" {
		t.Errorf("allowedCidrs mismatch: %+v", req.AllowedCIDRs)
	}

	// hostAliases (nested array).
	if len(req.HostAliases) != 1 || req.HostAliases[0].IP != "10.0.0.1" {
		t.Errorf("hostAliases mismatch: %+v", req.HostAliases)
	}
	if len(req.HostAliases[0].Hostnames) != 2 {
		t.Errorf("hostnames mismatch: %+v", req.HostAliases[0].Hostnames)
	}
}

// TestLoadCreateSpec_MissingRequiredFails verifies the required-field gate
// rejects a spec missing name/networkMode/flavorId.
func TestLoadCreateSpec_MissingRequiredFails(t *testing.T) {
	_, err := loadCreateSpec([]byte(`name: only-name`))
	if err == nil {
		t.Fatal("expected error for spec missing required fields")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected required-field error, got: %v", err)
	}
}

// TestParseHostAliases covers the ip=host1,host2 flag parser.
func TestParseHostAliases(t *testing.T) {
	out, err := parseHostAliases([]string{"10.0.0.1=foo.local,bar.local", "10.0.0.2=solo.local"})
	if err != nil {
		t.Fatalf("parseHostAliases: %v", err)
	}
	if len(out) != 2 || out[0].IP != "10.0.0.1" || len(out[0].Hostnames) != 2 || out[1].Hostnames[0] != "solo.local" {
		t.Errorf("unexpected: %+v", out)
	}
	if _, err := parseHostAliases([]string{"no-ip-prefix"}); err == nil {
		t.Error("expected error for malformed host-alias")
	}
	if _, err := parseHostAliases([]string{"=nohosts"}); err == nil {
		t.Error("expected error for empty ip")
	}
}
