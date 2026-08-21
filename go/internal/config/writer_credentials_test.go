package config

import (
	"testing"
	"time"
)

// WriteCredentials switches the profile to machine auth mode and drops any prior
// PKCE login token — the symmetric inverse of WriteLoginToken. This is the fix
// for `grn login` then `grn configure`: without it the stale auth_mode=user left
// NewTokenProvider selecting LoginTokenProvider and ignoring the just-configured
// machine creds.
func TestWriteCredentials_SwitchesToMachineModeAndClearsLoginToken(t *testing.T) {
	w := newHomeWriter(t)
	// Start from a logged-in profile (auth_mode=user + refresh token + iam_env).
	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := w.WriteLoginToken("default", "rt-orig", exp, "user", "dev"); err != nil {
		t.Fatalf("seed WriteLoginToken: %v", err)
	}

	// User runs `grn configure` (or `configure set client_id/secret`) — repoints
	// the profile at machine credentials.
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}

	s := loadCredsFile(t).Section("default")
	if got := s.Key("auth_mode").String(); got != "machine" {
		t.Errorf("auth_mode=%q, want machine (configure must switch off user mode)", got)
	}
	if got := s.Key("client_id").String(); got != "cid" {
		t.Errorf("client_id=%q, want cid", got)
	}
	if got := s.Key("client_secret").String(); got != "cs" {
		t.Errorf("client_secret=%q, want cs", got)
	}
	// The prior PKCE login token must be gone — it is not active under machine mode.
	if got := s.Key("refresh_token").String(); got != "" {
		t.Errorf("refresh_token=%q after WriteCredentials, want empty (machine mode drops it)", got)
	}
	if got := s.Key("token_expires_at").String(); got != "" {
		t.Errorf("token_expires_at=%q after WriteCredentials, want empty", got)
	}
	// iam_env is preserved: it selects the environment for both auth modes.
	if got := s.Key("iam_env").String(); got != "dev" {
		t.Errorf("iam_env=%q, want dev (preserved across auth-mode switch)", got)
	}
}

// WriteCredentials on a fresh profile (no prior login) just sets machine mode —
// there is nothing to clear, and it must not error.
func TestWriteCredentials_FreshProfileSetsMachineMode(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	s := loadCredsFile(t).Section("default")
	if got := s.Key("auth_mode").String(); got != "machine" {
		t.Errorf("auth_mode=%q, want machine", got)
	}
	if got := s.Key("client_id").String(); got != "cid" {
		t.Errorf("client_id=%q, want cid", got)
	}
}
