package agentbase

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	runtimepkg "github.com/greennodehub/greennode-cli/internal/agentbase/runtime"
)

// TestAgentbaseCmd_HasRuntimeSubtree verifies the runtime lifecycle group and
// its CRUD + wait + generate leaves are mounted under `grn agentbase`.
func TestAgentbaseCmd_HasRuntimeSubtree(t *testing.T) {
	rtCmd, _, err := AgentbaseCmd.Find([]string{"runtime"})
	if err != nil {
		t.Fatalf("agentbase has no 'runtime' subcommand: %v", err)
	}
	for _, want := range []string{"create", "generate", "list", "get", "update", "delete", "wait"} {
		if _, _, err := rtCmd.Find([]string{want}); err != nil {
			t.Errorf("runtime missing subcommand %q: %v", want, err)
		}
	}
}

// TestRuntimeGenerate_RoundTripsThroughLoadRuntimeSpec proves the template that
// `generate` prints is itself a valid, loadable create spec.
func TestRuntimeGenerate_RoundTripsThroughLoadRuntimeSpec(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := runtimeGenerateCmd.RunE(runtimeGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	req, err := loadRuntimeSpec([]byte(template), true)
	if err != nil {
		t.Fatalf("generated template is not a valid spec: %v", err)
	}
	// The template's flavorId is the literal placeholder "fill-flavor-id"; the
	// round-trip must decode it verbatim (proving camelCase binding), not drop it.
	if req.Name != "my-runtime" || req.ImageURL != "registry.example.com/my-agent:latest" || req.FlavorID != "fill-flavor-id" {
		t.Errorf("round-trip decoded unexpected spec: %+v", req)
	}
}

// TestRuntimeGenerate_PrintsJSONSkeleton verifies -o json prints a JSON skeleton.
func TestRuntimeGenerate_PrintsJSONSkeleton(t *testing.T) {
	output.SetFormat(output.FormatJSON)
	defer output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := runtimeGenerateCmd.RunE(runtimeGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	if !strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Errorf("expected JSON skeleton; got:\n%s", got)
	}
}

// TestLoadRuntimeSpec_NestedCamelCaseDecodes exercises the YAML→map→JSON→struct
// bridge on the runtime spec's nested camelCase keys (imageAuth, command/args,
// environmentVariables map, autoscaling) — the load-bearing test for the bridge
// trick since yaml.v3 ignores json tags.
func TestLoadRuntimeSpec_NestedCamelCaseDecodes(t *testing.T) {
	spec := []byte(`
name: nested-rt
description: a runtime
imageUrl: registry.example.com/agent:latest
imageAuth:
  enabled: true
  username: ci
  password: s3cret
command:
  - python
  - -m
  - svc
args:
  - --port
  - "8080"
environmentVariables:
  LOG_LEVEL: info
  FEATURE_X: "true"
flavorId: f1
autoscaling:
  maxReplicas: 3
  cpuUtilization: 65
  memoryUtilization: 75
`)
	req, err := loadRuntimeSpec(spec, true)
	if err != nil {
		t.Fatalf("loadRuntimeSpec: %v", err)
	}
	if req.Name != "nested-rt" || req.ImageURL != "registry.example.com/agent:latest" || req.FlavorID != "f1" {
		t.Errorf("top-level mismatch: %+v", req)
	}
	if req.ImageAuth == nil || !req.ImageAuth.Enabled || req.ImageAuth.Username != "ci" || req.ImageAuth.Password != "s3cret" {
		t.Errorf("imageAuth mismatch: %+v", req.ImageAuth)
	}
	if len(req.Command) != 3 || req.Command[1] != "-m" {
		t.Errorf("command mismatch: %+v", req.Command)
	}
	if len(req.Args) != 2 || req.Args[1] != "8080" {
		t.Errorf("args mismatch: %+v", req.Args)
	}
	if len(req.EnvironmentVariables) != 2 || req.EnvironmentVariables["LOG_LEVEL"] != "info" {
		t.Errorf("env mismatch: %+v", req.EnvironmentVariables)
	}
	if req.Autoscaling.MaxReplicas != 3 ||
		req.Autoscaling.CPUUtilization != 65 || req.Autoscaling.MemoryUtilization != 75 {
		t.Errorf("autoscaling mismatch: %+v", req.Autoscaling)
	}
}

// TestLoadRuntimeSpec_MissingRequiredFails verifies the required-field gate:
// imageUrl/flavorId always; name additionally when requireName (create).
func TestLoadRuntimeSpec_MissingRequiredFails(t *testing.T) {
	if _, err := loadRuntimeSpec([]byte(`name: only-name`), true); err == nil {
		t.Fatal("expected error for spec missing required fields")
	}
	// update path: name omitted is fine, but imageUrl/flavorId are still required
	if _, err := loadRuntimeSpec([]byte(`name: only-name`), false); err == nil {
		t.Fatal("expected error for update spec missing imageUrl/flavorId")
	}
	if _, err := loadRuntimeSpec([]byte("imageUrl: img\nflavorId: f1"), false); err != nil {
		t.Fatalf("update spec without name should load: %v", err)
	}
}

// TestParseEnvVars covers the KEY=VALUE flag parser.
func TestParseEnvVars(t *testing.T) {
	got, err := parseEnvVars([]string{"LOG_LEVEL=info", "TOKEN=abc=def", "EMPTY="})
	if err != nil {
		t.Fatalf("parseEnvVars: %v", err)
	}
	if got["LOG_LEVEL"] != "info" || got["TOKEN"] != "abc=def" || got["EMPTY"] != "" {
		t.Errorf("unexpected: %+v", got)
	}
	if _, err := parseEnvVars([]string{"NO_EQUALS"}); err == nil {
		t.Error("expected error for entry without '='")
	}
	if _, err := parseEnvVars([]string{"=NOKEY"}); err == nil {
		t.Error("expected error for entry with empty key")
	}
}

// newSpecCmd builds a fresh command with the shared spec flags, for testing
// readImageAuth / readRuntimeSpec in isolation.
func newSpecCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	addRuntimeSpecFlags(cmd)
	return cmd
}

func mustSet(t *testing.T, cmd *cobra.Command, flag, value string) {
	t.Helper()
	if err := cmd.Flags().Set(flag, value); err != nil {
		t.Fatalf("set %s: %v", flag, err)
	}
}

// TestReadImageAuth covers the three paths: none → {enabled:false} (the API
// requires imageAuth present — nil used to 400 server-side); enabled+creds →
// built; enabled without password → error.
func TestReadImageAuth(t *testing.T) {
	t.Run("none_returns_disabled_default", func(t *testing.T) {
		cmd := newSpecCmd(t)
		got, err := readImageAuth(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || got.Enabled {
			t.Errorf("expected &ImageAuth{Enabled:false}, got %+v", got)
		}
		if got.Username != "" || got.Password != "" || got.UseAgentBaseRegistryCredentials != nil {
			t.Errorf("disabled default must carry no credentials: %+v", got)
		}
	})
	t.Run("enabled_with_creds", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "image-auth-enabled", "true")
		mustSet(t, cmd, "image-auth-username", "u")
		mustSet(t, cmd, "image-auth-password", "p")
		got, err := readImageAuth(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || !got.Enabled || got.Username != "u" || got.Password != "p" {
			t.Errorf("unexpected: %+v", got)
		}
	})
	t.Run("enabled_without_password_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "image-auth-enabled", "true")
		mustSet(t, cmd, "image-auth-username", "u")
		// no password
		if _, err := readImageAuth(cmd); err == nil {
			t.Error("expected error when enabled without password")
		}
	})
}

// TestReadRuntimeSpec_AutoscalingMaxLtOneErrors verifies the max>=1 guard
// (minReplicas is platform-owned; only the ceiling is configurable).
func TestReadRuntimeSpec_AutoscalingMaxLtOneErrors(t *testing.T) {
	cliinput.SetInteractive(false)
	cmd := newSpecCmd(t)
	mustSet(t, cmd, "image-url", "img")
	mustSet(t, cmd, "flavor-id", "f")
	mustSet(t, cmd, "max-replicas", "0") // < 1
	if _, err := readRuntimeSpec(cmd, false); err == nil {
		t.Error("expected error when max-replicas < 1")
	}
}

// TestReadRuntimeSpec_OK verifies the happy path assembles all fields, including
// the flag-defaulted autoscaling and the parsed env map.
func TestReadRuntimeSpec_OK(t *testing.T) {
	cliinput.SetInteractive(false)
	cmd := newSpecCmd(t)
	mustSet(t, cmd, "image-url", "img:1")
	mustSet(t, cmd, "flavor-id", "f1")
	mustSet(t, cmd, "command", "python")
	mustSet(t, cmd, "env", "K=V")
	s, err := readRuntimeSpec(cmd, false)
	if err != nil {
		t.Fatalf("readRuntimeSpec: %v", err)
	}
	if s.ImageURL != "img:1" || s.FlavorID != "f1" {
		t.Errorf("mismatch: %+v", s)
	}
	if len(s.Command) != 1 || s.Command[0] != "python" {
		t.Errorf("command mismatch: %+v", s.Command)
	}
	if s.Env["K"] != "V" {
		t.Errorf("env mismatch: %+v", s.Env)
	}
	if s.Autoscaling.MaxReplicas != 2 { // flag defaults (minReplicas is platform-owned)
		t.Errorf("autoscaling default mismatch: %+v", s.Autoscaling)
	}
	if s.ImageAuth == nil || s.ImageAuth.Enabled {
		t.Errorf("imageAuth should default to &ImageAuth{Enabled:false}: %+v", s.ImageAuth)
	}
	// Ensure the returned spec round-trips into a request struct.
	_ = runtimepkg.CreateAgentRuntimeRequest{
		ImageURL: s.ImageURL, FlavorID: s.FlavorID, Autoscaling: s.Autoscaling,
	}
}

// TestReadNetworkConfig covers the --network-* flag group: unset → nil (the
// platform default PUBLIC), VPC without vpc/subnet → error, VPC happy path.
func TestReadNetworkConfig(t *testing.T) {
	t.Run("unset_returns_nil", func(t *testing.T) {
		cmd := newSpecCmd(t)
		got, err := readNetworkConfig(cmd)
		if err != nil || got != nil {
			t.Errorf("expected (nil, nil), got (%+v, %v)", got, err)
		}
	})
	t.Run("vpc_without_ids_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "network-mode", "VPC")
		if _, err := readNetworkConfig(cmd); err == nil {
			t.Error("expected error when VPC without --vpc-id/--subnet-id")
		}
	})
	t.Run("vpc_ok", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "network-mode", "VPC")
		mustSet(t, cmd, "vpc-id", "vpc-1")
		mustSet(t, cmd, "subnet-id", "sub-1")
		mustSet(t, cmd, "route-cidrs", "10.0.0.0/8")
		got, err := readNetworkConfig(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Mode != "VPC" || got.VpcID != "vpc-1" || got.SubnetID != "sub-1" || len(got.RouteCidrs) != 1 {
			t.Errorf("unexpected: %+v", got)
		}
	})
	t.Run("bad_mode_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "network-mode", "TUNNEL")
		if _, err := readNetworkConfig(cmd); err == nil {
			t.Error("expected error for bad --network-mode")
		}
	})
}

// TestReadInboundAuth covers the --inbound-* flag group: unset → nil, NONE/IAM
// happy paths, JWT validation matrix (source required; DISCOVERY needs URL;
// JWKS needs valid inline JSON; jwt flags rejected for non-JWT modes).
func TestReadInboundAuth(t *testing.T) {
	t.Run("unset_returns_nil", func(t *testing.T) {
		cmd := newSpecCmd(t)
		got, err := readInboundAuth(cmd)
		if err != nil || got != nil {
			t.Errorf("expected (nil, nil), got (%+v, %v)", got, err)
		}
	})
	t.Run("none_ok", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "NONE")
		got, err := readInboundAuth(cmd)
		if err != nil || got == nil || got.Mode != "NONE" || got.JWT != nil {
			t.Errorf("unexpected: (%+v, %v)", got, err)
		}
	})
	t.Run("jwt_without_source_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "JWT")
		if _, err := readInboundAuth(cmd); err == nil {
			t.Error("expected error when JWT without --inbound-jwt-source")
		}
	})
	t.Run("jwt_discovery_without_url_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "JWT")
		mustSet(t, cmd, "inbound-jwt-source", "DISCOVERY")
		if _, err := readInboundAuth(cmd); err == nil {
			t.Error("expected error when DISCOVERY without --inbound-jwt-discovery-url")
		}
	})
	t.Run("jwt_discovery_ok", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "JWT")
		mustSet(t, cmd, "inbound-jwt-source", "DISCOVERY")
		mustSet(t, cmd, "inbound-jwt-discovery-url", "https://idp/.well-known/openid-configuration")
		mustSet(t, cmd, "inbound-jwt-audience", "aud1")
		mustSet(t, cmd, "inbound-jwt-principal-claim", "sub")
		got, err := readInboundAuth(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Mode != "JWT" || got.JWT == nil || got.JWT.Source != "DISCOVERY" ||
			got.JWT.DiscoveryURL == "" || len(got.JWT.AllowedAudiences) != 1 || got.JWT.PrincipalClaim != "sub" {
			t.Errorf("unexpected: %+v", got)
		}
	})
	t.Run("jwt_jwks_invalid_json_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "JWT")
		mustSet(t, cmd, "inbound-jwt-source", "JWKS")
		mustSet(t, cmd, "inbound-jwt-jwks", "{not-json")
		if _, err := readInboundAuth(cmd); err == nil {
			t.Error("expected error for invalid JWKS JSON")
		}
	})
	t.Run("jwt_jwks_ok", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "JWT")
		mustSet(t, cmd, "inbound-jwt-source", "JWKS")
		mustSet(t, cmd, "inbound-jwt-jwks", `{"keys":[]}`)
		got, err := readInboundAuth(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got.JWT.JWKS) != `{"keys":[]}` {
			t.Errorf("jwks mismatch: %s", got.JWT.JWKS)
		}
	})
	t.Run("iam_with_jwt_flags_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "inbound-mode", "IAM")
		mustSet(t, cmd, "inbound-jwt-source", "JWKS")
		if _, err := readInboundAuth(cmd); err == nil {
			t.Error("expected error when jwt flags set with --inbound-mode IAM")
		}
	})
}

// TestReadRuntimeSpec_NewFieldsAssemble verifies the new spec fields flow
// through readRuntimeSpec into the spec struct (protocol, allowedCidrs,
// network, inbound) and that the deprecated --min-replicas no longer errors.
func TestReadRuntimeSpec_NewFieldsAssemble(t *testing.T) {
	cliinput.SetInteractive(false)
	cmd := newSpecCmd(t)
	mustSet(t, cmd, "image-url", "img:1")
	mustSet(t, cmd, "flavor-id", "f1")
	mustSet(t, cmd, "protocol", "MCP")
	mustSet(t, cmd, "allowed-cidrs", "10.0.0.0/8")
	mustSet(t, cmd, "network-mode", "VPC")
	mustSet(t, cmd, "vpc-id", "vpc-1")
	mustSet(t, cmd, "subnet-id", "sub-1")
	mustSet(t, cmd, "inbound-mode", "IAM")
	mustSet(t, cmd, "min-replicas", "3") // deprecated: tolerated, ignored
	s, err := readRuntimeSpec(cmd, false)
	if err != nil {
		t.Fatalf("readRuntimeSpec: %v", err)
	}
	if s.Protocol != "MCP" || len(s.AllowedCidrs) != 1 ||
		s.Network == nil || s.Network.Mode != "VPC" ||
		s.Inbound == nil || s.Inbound.Mode != "IAM" {
		t.Errorf("new fields mismatch: %+v", s)
	}
	// The assembled spec must fit the create request (compile-time shape check).
	_ = runtimepkg.CreateAgentRuntimeRequest{
		NetworkConfig: s.Network, AllowedCidrs: s.AllowedCidrs,
		InboundAuth: s.Inbound, Protocol: s.Protocol,
	}
}

// TestReadImageAuth_UseRegistryCredentials covers the platform-registry path.
func TestReadImageAuth_UseRegistryCredentials(t *testing.T) {
	cmd := newSpecCmd(t)
	mustSet(t, cmd, "image-auth-use-registry-credentials", "true")
	got, err := readImageAuth(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || !got.Enabled || got.UseAgentBaseRegistryCredentials == nil || !*got.UseAgentBaseRegistryCredentials {
		t.Errorf("unexpected: %+v", got)
	}
}

// TestCreateRequestFromSpec verifies the flag-path create request carries the
// name (34830: the request literal used to omit it, so create-by-flags always
// failed server-side with a name validation error).
func TestCreateRequestFromSpec(t *testing.T) {
	spec := &runtimeSpec{
		ImageURL: "img:1", FlavorID: "f1", Command: []string{"python"},
		Env:         map[string]string{"K": "V"},
		ImageAuth:   &runtimepkg.ImageAuth{Enabled: false},
		Autoscaling: runtimepkg.Autoscaling{MaxReplicas: 2},
	}
	req := createRequestFromSpec("my-rt", spec)
	if req.Name != "my-rt" {
		t.Errorf("name missing from create request: %+v", req)
	}
	if req.ImageURL != "img:1" || req.FlavorID != "f1" || req.Autoscaling.MaxReplicas != 2 {
		t.Errorf("spec fields not mapped: %+v", req)
	}
	if req.ImageAuth == nil || req.ImageAuth.Enabled {
		t.Errorf("imageAuth should be &ImageAuth{Enabled:false}: %+v", req.ImageAuth)
	}
}

// TestReadRuntimeSpec_ForUpdate covers the update mode: no prompting for
// image-url/flavor-id (unset = carry over), explicitly-set-but-empty rejected,
// and the autoscaling max>=1 guard skipped when no autoscaling flag is set
// (the guard runs on the merged request, not the partial spec).
func TestReadRuntimeSpec_ForUpdate(t *testing.T) {
	cliinput.SetInteractive(false)
	t.Run("unset_image_and_flavor_ok", func(t *testing.T) {
		cmd := newSpecCmd(t)
		s, err := readRuntimeSpec(cmd, true)
		if err != nil {
			t.Fatalf("readRuntimeSpec(forUpdate): %v", err)
		}
		if s.ImageURL != "" || s.FlavorID != "" {
			t.Errorf("unset flags must stay empty on update: %+v", s)
		}
	})
	t.Run("explicitly_empty_image_url_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "image-url", "")
		if _, err := readRuntimeSpec(cmd, true); err == nil {
			t.Error("expected error for explicitly-set-but-empty --image-url")
		}
	})
	t.Run("explicitly_empty_flavor_id_errors", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "flavor-id", "")
		if _, err := readRuntimeSpec(cmd, true); err == nil {
			t.Error("expected error for explicitly-set-but-empty --flavor-id")
		}
	})
	t.Run("autoscaling_guard_skipped_when_unset", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "image-url", "img")
		mustSet(t, cmd, "flavor-id", "f")
		if _, err := readRuntimeSpec(cmd, true); err != nil {
			t.Errorf("update mode must not require autoscaling flags: %v", err)
		}
	})
	t.Run("autoscaling_guard_applies_when_set", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "max-replicas", "0")
		if _, err := readRuntimeSpec(cmd, true); err == nil {
			t.Error("expected error when --max-replicas < 1 is explicitly set")
		}
	})
}

// TestReadNetworkConfig_DanglingFlagsError covers vpc-id/subnet-id/route-cidrs
// set without --network-mode: previously silently ignored, now an error.
func TestReadNetworkConfig_DanglingFlagsError(t *testing.T) {
	for _, flag := range []string{"vpc-id", "subnet-id"} {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, flag, "x")
		if _, err := readNetworkConfig(cmd); err == nil {
			t.Errorf("expected error for dangling --%s", flag)
		}
	}
	cmd := newSpecCmd(t)
	mustSet(t, cmd, "route-cidrs", "10.0.0.0/8")
	if _, err := readNetworkConfig(cmd); err == nil {
		t.Error("expected error for dangling --route-cidrs")
	}
}

// testVersion is a current-version fixture for buildUpdateRequest: every field
// distinct so carry-over vs override is observable.
func testVersion() *runtimepkg.AgentRuntimeVersionDto {
	return &runtimepkg.AgentRuntimeVersionDto{
		Version:              7,
		Description:          "cur-desc",
		ImageURL:             "cur-img:7",
		Command:              []string{"cur-cmd"},
		Args:                 []string{"cur-arg"},
		EnvironmentVariables: map[string]string{"CUR": "1"},
		NetworkConfig: &runtimepkg.AgentRuntimeNetworkConfigEntity{
			Mode: "VPC", VpcID: "cur-vpc", SubnetID: "cur-sub", RouteCidrs: []string{"10.0.0.0/8"},
		},
		AllowedCidrs: []string{"10.0.0.0/8"},
		InboundAuth: &runtimepkg.AgentRuntimeInboundAuthDto{
			Mode: "JWT",
			JWT: &runtimepkg.AgentRuntimeInboundAuthJwtDto{
				Source: "JWKS", JWKS: json.RawMessage(`{"keys": []}`),
				AllowedAudiences: []string{"aud"}, PrincipalClaim: "sub",
			},
		},
		Protocol:    "MCP",
		FlavorID:    "cur-flavor",
		Autoscaling: runtimepkg.Autoscaling{MaxReplicas: 3},
	}
}

// specFromFlags assembles the update-path spec from a flag command.
func specFromFlags(t *testing.T, cmd *cobra.Command) *runtimeSpec {
	t.Helper()
	s, err := readRuntimeSpec(cmd, true)
	if err != nil {
		t.Fatalf("readRuntimeSpec(forUpdate): %v", err)
	}
	return s
}

// TestBuildUpdateRequest covers the update merge semantics (34836): unset
// flags carry the current version's field over; changed flags replace their
// field wholesale; imageAuth is derived from the current version with the
// password caveat.
func TestBuildUpdateRequest(t *testing.T) {
	t.Run("nothing_set_carries_everything", func(t *testing.T) {
		cmd := newSpecCmd(t)
		cur := testVersion()
		req, carried, err := buildUpdateRequest(specFromFlags(t, cmd), cur, cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.Description != "cur-desc" || req.ImageURL != "cur-img:7" ||
			req.Command[0] != "cur-cmd" || req.Args[0] != "cur-arg" ||
			req.EnvironmentVariables["CUR"] != "1" || req.FlavorID != "cur-flavor" ||
			req.Protocol != "MCP" || req.AllowedCidrs[0] != "10.0.0.0/8" ||
			req.Autoscaling.MaxReplicas != 3 {
			t.Errorf("carry-over mismatch: %+v", req)
		}
		if req.NetworkConfig == nil || req.NetworkConfig.Mode != "VPC" || req.NetworkConfig.VpcID != "cur-vpc" ||
			req.NetworkConfig.SubnetID != "cur-sub" || req.NetworkConfig.RouteCidrs[0] != "10.0.0.0/8" {
			t.Errorf("networkConfig carry-over mismatch: %+v", req.NetworkConfig)
		}
		if req.InboundAuth == nil || req.InboundAuth.Mode != "JWT" || req.InboundAuth.JWT == nil ||
			req.InboundAuth.JWT.Source != "JWKS" || string(req.InboundAuth.JWT.JWKS) != `{"keys": []}` ||
			req.InboundAuth.JWT.AllowedAudiences[0] != "aud" || req.InboundAuth.JWT.PrincipalClaim != "sub" {
			t.Errorf("inboundAuth carry-over mismatch: %+v", req.InboundAuth)
		}
		for _, f := range []string{"description", "imageUrl", "command", "args", "environmentVariables",
			"networkConfig", "allowedCidrs", "inboundAuth", "protocol", "flavorId", "autoscaling", "imageAuth"} {
			if !slices.Contains(carried, f) {
				t.Errorf("carried list missing %s: %v", f, carried)
			}
		}
	})
	t.Run("changed_flag_replaces_field_wholesale", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "env", "NEW=2")
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), testVersion(), cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.EnvironmentVariables["NEW"] != "2" || len(req.EnvironmentVariables) != 1 {
			t.Errorf("env should be replaced wholesale, not merged: %+v", req.EnvironmentVariables)
		}
		if req.ImageURL != "cur-img:7" {
			t.Errorf("unset image-url should carry over: %+v", req)
		}
	})
	t.Run("network_flag_replaces_network", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "network-mode", "VPC")
		mustSet(t, cmd, "vpc-id", "v1")
		mustSet(t, cmd, "subnet-id", "s1")
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), testVersion(), cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.NetworkConfig == nil || req.NetworkConfig.VpcID != "v1" ||
			req.NetworkConfig.SubnetID != "s1" || len(req.NetworkConfig.RouteCidrs) != 0 {
			t.Errorf("networkConfig should be the flag spec: %+v", req.NetworkConfig)
		}
	})
	t.Run("imageauth_password_auth_cannot_carry", func(t *testing.T) {
		cur := testVersion()
		cur.ImageAuth = &runtimepkg.AgentRuntimeImageAuthDto{Enabled: true, Username: "u"}
		cmd := newSpecCmd(t)
		_, _, err := buildUpdateRequest(specFromFlags(t, cmd), cur, cmd.Flags().Changed)
		if err == nil {
			t.Error("expected error carrying username/password image auth")
		}
	})
	t.Run("imageauth_registry_credentials_carries", func(t *testing.T) {
		cur := testVersion()
		cur.ImageAuth = &runtimepkg.AgentRuntimeImageAuthDto{Enabled: true, UseAgentBaseRegistryCredentials: true, Username: "u"}
		cmd := newSpecCmd(t)
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), cur, cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		ia := req.ImageAuth
		if ia == nil || !ia.Enabled || ia.UseAgentBaseRegistryCredentials == nil || !*ia.UseAgentBaseRegistryCredentials ||
			ia.Username != "u" || ia.Password != "" {
			t.Errorf("registry-credentials imageAuth should carry verbatim: %+v", ia)
		}
	})
	t.Run("imageauth_disabled_carries_as_disabled", func(t *testing.T) {
		cur := testVersion() // ImageAuth nil = disabled
		cmd := newSpecCmd(t)
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), cur, cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.ImageAuth == nil || req.ImageAuth.Enabled {
			t.Errorf("nil imageAuth must carry as {enabled:false}: %+v", req.ImageAuth)
		}
	})
	t.Run("imageauth_flag_wins", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "image-auth-enabled", "true")
		mustSet(t, cmd, "image-auth-username", "nu")
		mustSet(t, cmd, "image-auth-password", "np")
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), testVersion(), cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.ImageAuth == nil || !req.ImageAuth.Enabled || req.ImageAuth.Username != "nu" || req.ImageAuth.Password != "np" {
			t.Errorf("image-auth flags should replace imageAuth: %+v", req.ImageAuth)
		}
	})
	t.Run("autoscaling_flag_replaces_autoscaling", func(t *testing.T) {
		cmd := newSpecCmd(t)
		mustSet(t, cmd, "max-replicas", "5")
		req, _, err := buildUpdateRequest(specFromFlags(t, cmd), testVersion(), cmd.Flags().Changed)
		if err != nil {
			t.Fatalf("buildUpdateRequest: %v", err)
		}
		if req.Autoscaling.MaxReplicas != 5 {
			t.Errorf("autoscaling should be the flag spec: %+v", req.Autoscaling)
		}
	})
}

// TestCarriedImageAuth_UnsetVersion returns {enabled:false} (imageAuth is
// required server-side; nil used to 400 the update).
func TestCarriedImageAuth_UnsetVersion(t *testing.T) {
	ia, err := carriedImageAuth(&runtimepkg.AgentRuntimeVersionDto{})
	if err != nil {
		t.Fatalf("carriedImageAuth: %v", err)
	}
	if ia == nil || ia.Enabled || ia.Username != "" {
		t.Errorf("expected &ImageAuth{Enabled:false}, got %+v", ia)
	}
}
