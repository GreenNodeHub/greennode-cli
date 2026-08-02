package agentbase

import (
	"testing"

	crpkg "github.com/vngcloud/greennode-cli/internal/agentbase/cr"
)

// TestAgentbaseCmd_HasCRSubtree verifies the cr command tree mirrors the
// container-registry contract: a repository (get) with nested image + artifact
// (list/delete each, deletes keyed by query params), and a registry-credential
// (get/reset-secret). There is no async FSM and no create (resources are
// auto-provisioned on first read), so neither 'wait' nor 'create' may appear.
func TestAgentbaseCmd_HasCRSubtree(t *testing.T) {
	crRoot, _, err := AgentbaseCmd.Find([]string{"cr"})
	if err != nil {
		t.Fatalf("agentbase has no 'cr' subcommand: %v", err)
	}
	assertSubCommands(t, crRoot, "repository", "registry-credential")
	assertNoSubCommands(t, crRoot, "wait", "create")

	repoCmd, _, err := crRoot.Find([]string{"repository"})
	if err != nil {
		t.Fatalf("cr has no 'repository' subcommand: %v", err)
	}
	assertSubCommands(t, repoCmd, "get", "image", "artifact")
	assertNoSubCommands(t, repoCmd, "create", "wait")

	imageCmd, _, err := repoCmd.Find([]string{"image"})
	if err != nil {
		t.Fatalf("cr repository has no 'image' subcommand: %v", err)
	}
	assertSubCommands(t, imageCmd, "list", "delete")
	assertNoSubCommands(t, imageCmd, "get", "update")

	artifactCmd, _, err := repoCmd.Find([]string{"artifact"})
	if err != nil {
		t.Fatalf("cr repository has no 'artifact' subcommand: %v", err)
	}
	assertSubCommands(t, artifactCmd, "list", "delete")

	credCmd, _, err := crRoot.Find([]string{"registry-credential"})
	if err != nil {
		t.Fatalf("cr has no 'registry-credential' subcommand: %v", err)
	}
	assertSubCommands(t, credCmd, "get", "reset-secret")
	assertNoSubCommands(t, credCmd, "create", "delete")
}

// TestCRImageDeleteCmd_HasImageNameFlag confirms the delete target rides a
// flag (mapped to the ?imageName= query param), not a positional — the contract
// identifies images by query, not path segment.
func TestCRImageDeleteCmd_HasImageNameFlag(t *testing.T) {
	if flag := crImageDeleteCmd.Flags().Lookup("image-name"); flag == nil {
		t.Fatal("cr repository image delete must define --image-name")
	}
}

func TestCRArtifactDeleteCmd_HasRequiredFlags(t *testing.T) {
	for _, f := range []string{"image-name", "digest"} {
		if flag := crArtifactDeleteCmd.Flags().Lookup(f); flag == nil {
			t.Errorf("cr repository artifact delete must define --%s", f)
		}
	}
}

// TestMaskSecret confirms the registry-credential secret is masked to last-4
// (the table posture); the JSON path reveals it via the raw struct.
func TestMaskSecret(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abcdef123456", "********3456"},
		{"abc", "****"},
		{"", "-"},
		{"1234", "****"}, // len <= 4 fully masked
	}
	for _, c := range cases {
		if got := maskSecret(c.in); got != c.want {
			t.Errorf("maskSecret(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2.0 KiB"},
		{5 * 1024 * 1024, "5.0 MiB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestTagsToStr(t *testing.T) {
	if got := tagsToStr(nil); got != "-" {
		t.Errorf("nil tags should be '-', got %q", got)
	}
	got := tagsToStr([]crpkg.Tag{{Name: "v1"}, {Name: "v2"}})
	if got != "v1,v2" {
		t.Errorf("tags mismatch: %q", got)
	}
}
