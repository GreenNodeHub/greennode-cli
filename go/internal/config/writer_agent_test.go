package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// WriteAgentIdentity folds the agentbase current-agent selection into the same
// [profile] section, preserving machine creds + login keys already there.
func TestWriteAgentIdentity_PreservesOtherKeys(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	if err := w.WriteLoginToken("default", "rt-123", time.Now().UTC(), "user", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}
	if err := w.WriteAgentIdentity("default", "agent-7"); err != nil {
		t.Fatalf("WriteAgentIdentity: %v", err)
	}

	s := loadCredsFile(t).Section("default")
	// Machine creds + login keys intact.
	if s.Key("client_id").String() != "cid" {
		t.Errorf("client_id=%q, want cid", s.Key("client_id").String())
	}
	if s.Key("refresh_token").String() != "rt-123" {
		t.Errorf("refresh_token=%q, want rt-123", s.Key("refresh_token").String())
	}
	if s.Key("iam_env").String() != "dev" {
		t.Errorf("iam_env=%q, want dev", s.Key("iam_env").String())
	}
	// agent_identity written.
	if got := s.Key("agent_identity").String(); got != "agent-7" {
		t.Errorf("agent_identity=%q, want agent-7", got)
	}
}

// WriteAgentIdentity on one profile leaves a different profile's section
// untouched (per-profile isolation, same contract as WriteLoginToken).
func TestWriteAgentIdentity_PerProfileIsolation(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteAgentIdentity("default", "agent-prod"); err != nil {
		t.Fatalf("default WriteAgentIdentity: %v", err)
	}
	if err := w.WriteAgentIdentity("dev", "agent-dev"); err != nil {
		t.Fatalf("dev WriteAgentIdentity: %v", err)
	}
	f := loadCredsFile(t)
	if got := f.Section("default").Key("agent_identity").String(); got != "agent-prod" {
		t.Errorf("default agent_identity=%q, want agent-prod", got)
	}
	if got := f.Section("dev").Key("agent_identity").String(); got != "agent-dev" {
		t.Errorf("dev agent_identity=%q, want agent-dev", got)
	}
}

// An empty name CLEARS the key (explicit unset), not a no-op — so `agent-id use`
// can distinguish "select none" from "no change".
func TestWriteAgentIdentity_EmptyClearsKey(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteAgentIdentity("default", "agent-1"); err != nil {
		t.Fatalf("seed WriteAgentIdentity: %v", err)
	}
	if err := w.WriteAgentIdentity("default", ""); err != nil {
		t.Fatalf("empty WriteAgentIdentity: %v", err)
	}
	if got := loadCredsFile(t).Section("default").Key("agent_identity").String(); got != "" {
		t.Errorf("agent_identity=%q after empty write, want empty (cleared)", got)
	}
}

// WriteIamEnv overwrites only iam_env, leaving the other login keys intact —
// the machine-mode counterpart to `grn login --iam-env` for `context switch`.
func TestWriteIamEnv_OverwritesOnlyIamEnv(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteLoginToken("default", "rt-123", time.Now().UTC(), "user", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}
	if err := w.WriteIamEnv("default", "prod"); err != nil {
		t.Fatalf("WriteIamEnv: %v", err)
	}
	s := loadCredsFile(t).Section("default")
	if s.Key("iam_env").String() != "prod" {
		t.Errorf("iam_env=%q, want prod", s.Key("iam_env").String())
	}
	// Other login keys must survive the single-key write.
	if s.Key("refresh_token").String() != "rt-123" {
		t.Errorf("refresh_token=%q, want rt-123 (single-key write must not disturb)", s.Key("refresh_token").String())
	}
	if s.Key("auth_mode").String() != "user" {
		t.Errorf("auth_mode=%q, want user", s.Key("auth_mode").String())
	}
}

// WriteIamEnv also works on a machine profile that never had login keys — it
// just sets iam_env alongside client_id/client_secret.
func TestWriteIamEnv_OnMachineProfile(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	if err := w.WriteIamEnv("default", "dev"); err != nil {
		t.Fatalf("WriteIamEnv: %v", err)
	}
	s := loadCredsFile(t).Section("default")
	if s.Key("iam_env").String() != "dev" {
		t.Errorf("iam_env=%q, want dev", s.Key("iam_env").String())
	}
	if s.Key("client_id").String() != "cid" {
		t.Errorf("client_id=%q, want cid", s.Key("client_id").String())
	}
}

// LoadConfig reads agent_identity back into Config from the credentials INI —
// agentbase resolves its current agent from the shared profile, not a separate
// .greennode.json.
func TestLoadConfig_ReadsAgentIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, k := range []string{"GRN_PROFILE", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
		t.Setenv(k, "")
	}
	dir := filepath.Join(home, ".greennode")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "credentials"),
		"[default]\nclient_id = cid\nclient_secret = cs\niam_env = prod\nagent_identity = agent-9\n")

	cfg, err := LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AgentIdentity != "agent-9" {
		t.Errorf("AgentIdentity=%q, want agent-9", cfg.AgentIdentity)
	}
	// iam_env still resolves alongside it.
	if cfg.IamEnv != "prod" {
		t.Errorf("IamEnv=%q, want prod", cfg.IamEnv)
	}
}

// A profile with no agent_identity selected loads fine (empty AgentIdentity is
// the default — agentbase commands handle "no current agent").
func TestLoadConfig_MissingAgentIdentityIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, k := range []string{"GRN_PROFILE", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
		t.Setenv(k, "")
	}
	dir := filepath.Join(home, ".greennode")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "credentials"), "[default]\nclient_id = cid\nclient_secret = cs\n")

	cfg, err := LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AgentIdentity != "" {
		t.Errorf("AgentIdentity=%q, want empty", cfg.AgentIdentity)
	}
}
