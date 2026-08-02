package agentbase

import (
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
)

// TestAgentbaseCmd_HasDeploySubtree verifies the deploy command tree: generate,
// up, status, destroy. deploy is a composite (no backend), so there is no
// create/list/get leaf — those live on the per-service commands it orchestrates.
func TestAgentbaseCmd_HasDeploySubtree(t *testing.T) {
	dep, _, err := AgentbaseCmd.Find([]string{"deploy"})
	if err != nil {
		t.Fatalf("agentbase has no 'deploy' subcommand: %v", err)
	}
	assertSubCommands(t, dep, "generate", "up", "status", "destroy")
	assertNoSubCommands(t, dep, "create", "list", "get", "delete")
}

// TestDeployUpCmd_HasCoreFlags confirms up carries --file, --no-wait, and the
// --image-auth flag (the cr-integration seam).
func TestDeployUpCmd_HasCoreFlags(t *testing.T) {
	for _, f := range []string{"file", "no-wait", "image-auth", "memory-strategy", "set-current"} {
		if flag := deployUpCmd.Flags().Lookup(f); flag == nil {
			t.Errorf("deploy up must define --%s", f)
		}
	}
}

func TestDeployDestroyCmd_HasPurgeFlag(t *testing.T) {
	if flag := deployDestroyCmd.Flags().Lookup("purge"); flag == nil {
		t.Error("deploy destroy must define --purge")
	}
}

// TestDeployGenerate_RoundTripsThroughLoad proves the YAML template prints a
// valid, loadable manifest (with a memory block + imageAuth:auto).
func TestDeployGenerate_RoundTripsThroughLoad(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := deployGenerateCmd.RunE(deployGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	mfst, err := loadDeployManifest([]byte(template))
	if err != nil {
		t.Fatalf("generated template is not a valid manifest: %v", err)
	}
	if mfst.Name != "my-agent" {
		t.Errorf("round-trip decoded unexpected name: %+v", mfst.Name)
	}
	if !mfst.Runtime.ImageAuth.Auto {
		t.Errorf("template imageAuth should be auto: %+v", mfst.Runtime.ImageAuth)
	}
	if mfst.Memory == nil || len(mfst.Memory.LongTermMemoryStrategies) != 1 {
		t.Errorf("template should carry one memory strategy: %+v", mfst.Memory)
	}
}

// TestDeployGenerate_PrintsJSONSkeleton confirms -o json emits a JSON object.
func TestDeployGenerate_PrintsJSONSkeleton(t *testing.T) {
	output.SetFormat(output.FormatJSON)
	defer output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := deployGenerateCmd.RunE(deployGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	got = trimSpaceNewline(got)
	if len(got) == 0 || got[0] != '{' {
		t.Errorf("expected JSON skeleton; got:\n%s", got)
	}
}

// TestLoadDeployManifest decodes a full manifest (with memory + imageAuth:auto)
// through the YAML->map->JSON->struct bridge.
func TestLoadDeployManifest(t *testing.T) {
	spec := []byte(`
name: my-agent
description: "demo"
identity:
  allowedReturnUrls:
    - https://app.example.com/cb
memory:
  eventExpiryDuration: 3600
  strategies:
    - name: prefs
      type: USER_PREFERENCE
      namespaceTemplate: "/strategies/USER_PREFERENCE/actors/{actorId}"
runtime:
  image: registry.vngcloud.vn/u/my-agent:v1
  imageAuth: auto
  command: [./agent]
  args: [--port, "8080"]
  env: {LOG_LEVEL: info}
  flavorId: agent.small
  autoscaling: {minReplicas: 1, maxReplicas: 3, cpuUtilization: 70, memoryUtilization: 80}
`)
	mfst, err := loadDeployManifest(spec)
	if err != nil {
		t.Fatalf("loadDeployManifest: %v", err)
	}
	if mfst.Name != "my-agent" || mfst.Description != "demo" {
		t.Errorf("top-level mismatch: %+v", mfst)
	}
	if len(mfst.Identity.AllowedReturnURLs) != 1 {
		t.Errorf("identity urls mismatch: %+v", mfst.Identity)
	}
	if mfst.Memory == nil || len(mfst.Memory.LongTermMemoryStrategies) != 1 {
		t.Fatalf("memory mismatch: %+v", mfst.Memory)
	}
	if mfst.Memory.LongTermMemoryStrategies[0].Type != "USER_PREFERENCE" {
		t.Errorf("strategy type mismatch: %+v", mfst.Memory.LongTermMemoryStrategies[0])
	}
	if mfst.Runtime.Image != "registry.vngcloud.vn/u/my-agent:v1" || mfst.Runtime.FlavorID != "agent.small" {
		t.Errorf("runtime mismatch: %+v", mfst.Runtime)
	}
	if !mfst.Runtime.ImageAuth.Auto {
		t.Errorf("imageAuth should be auto: %+v", mfst.Runtime.ImageAuth)
	}
	if mfst.Runtime.Autoscaling.MaxReplicas != 3 {
		t.Errorf("autoscaling mismatch: %+v", mfst.Runtime.Autoscaling)
	}
}

// TestLoadDeployManifest_ImageAuthExplicit verifies the explicit object form
// and that a stateless agent (no memory block) decodes.
func TestLoadDeployManifest_ImageAuthExplicit_Stateless(t *testing.T) {
	spec := []byte(`
name: stateless-agent
runtime:
  image: img
  flavorId: f
  imageAuth: {username: u, password: p}
`)
	mfst, err := loadDeployManifest(spec)
	if err != nil {
		t.Fatalf("loadDeployManifest: %v", err)
	}
	if mfst.Memory != nil {
		t.Errorf("stateless agent should have no memory: %+v", mfst.Memory)
	}
	if mfst.Runtime.ImageAuth.Auto || mfst.Runtime.ImageAuth.Username != "u" || mfst.Runtime.ImageAuth.Password != "p" {
		t.Errorf("explicit imageAuth mismatch: %+v", mfst.Runtime.ImageAuth)
	}
}

func TestLoadDeployManifest_MissingRequiredFails(t *testing.T) {
	cases := map[string]string{
		"missing name":     `runtime: {image: i, flavorId: f}`,
		"missing image":    `name: a` + "\n" + `runtime: {flavorId: f}`,
		"missing flavorId": `name: a` + "\n" + `runtime: {image: i}`,
		"memory no strat":  "name: a\nruntime: {image: i, flavorId: f}\nmemory: {eventExpiryDuration: 60}",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadDeployManifest([]byte(spec)); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

// TestNonNilStrings confirms nil slices/maps are normalized to empty (the
// runtime's @NotNull fields reject JSON null).
func TestNonNilStrings(t *testing.T) {
	if got := nonNilStrings(nil); len(got) != 0 {
		t.Errorf("nil should become empty slice, got %v", got)
	}
	in := []string{"a"}
	if got := nonNilStrings(in); len(got) != 1 || got[0] != "a" {
		t.Errorf("non-nil should pass through, got %v", got)
	}
}

func TestNonNilMap(t *testing.T) {
	if got := nonNilMap(nil); got == nil || len(got) != 0 {
		t.Errorf("nil should become empty map, got %v", got)
	}
	in := map[string]string{"k": "v"}
	if got := nonNilMap(in); got["k"] != "v" {
		t.Errorf("non-nil should pass through, got %v", got)
	}
}

func TestMemoryState(t *testing.T) {
	if memoryState("", true) != "skipped (stateless)" {
		t.Error("skipped mismatch")
	}
	if memoryState("id-1", false) != "present" {
		t.Error("present mismatch")
	}
}

// trimSpaceNewline trims leading/trailing whitespace and newlines.
func trimSpaceNewline(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ' ' || last == '\n' || last == '\r' || last == '\t' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
