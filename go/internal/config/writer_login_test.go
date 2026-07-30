package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

// newHomeWriter points HOME at a temp dir so ConfigFileWriter (which resolves
// DefaultConfigDir() from HOME at each NewConfigFileWriter call) writes into an
// isolated ~/.greennode. Non-parallel: it mutates process env (HOME).
func newHomeWriter(t *testing.T) *ConfigFileWriter {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return NewConfigFileWriter()
}

func credsPath(t *testing.T) string {
	t.Helper()
	home := os.Getenv("HOME")
	return filepath.Join(home, ".greennode", "credentials")
}

func loadCredsFile(t *testing.T) *ini.File {
	t.Helper()
	f, err := ini.Load(credsPath(t))
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	return f
}

// WriteLoginToken folds the refresh token + non-secret context into the same
// [profile] section, preserving machine client_id/client_secret already there.
func TestWriteLoginToken_PreservesMachineCredentials(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}

	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := w.WriteLoginToken("default", "rt-123", exp, "user", "cid", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}

	s := loadCredsFile(t).Section("default")
	// Machine creds intact.
	if s.Key("client_id").String() != "cid" {
		t.Errorf("client_id=%q, want cid", s.Key("client_id").String())
	}
	if s.Key("client_secret").String() != "cs" {
		t.Errorf("client_secret=%q, want cs", s.Key("client_secret").String())
	}
	// Login keys written.
	if s.Key("refresh_token").String() != "rt-123" {
		t.Errorf("refresh_token=%q, want rt-123", s.Key("refresh_token").String())
	}
	if s.Key("auth_mode").String() != "user" {
		t.Errorf("auth_mode=%q, want user", s.Key("auth_mode").String())
	}
	if s.Key("login_client_id").String() != "cid" {
		t.Errorf("login_client_id=%q, want cid", s.Key("login_client_id").String())
	}
	if s.Key("iam_env").String() != "dev" {
		t.Errorf("iam_env=%q, want dev", s.Key("iam_env").String())
	}
	// token_expires_at is RFC3339 UTC and round-trips.
	got, err := time.Parse(time.RFC3339, s.Key("token_expires_at").String())
	if err != nil {
		t.Errorf("token_expires_at unparseable: %v", err)
	}
	if !got.Equal(exp) {
		t.Errorf("token_expires_at=%v, want %v", got, exp)
	}
}

// An empty refresh token is a no-op so a stale empty value can never erase a
// prior good token. (cmd/login also skips the call on partial success.)
func TestWriteLoginToken_EmptyRefreshTokenIsNoop(t *testing.T) {
	w := newHomeWriter(t)
	// Seed an existing good token.
	if err := w.WriteLoginToken("default", "rt-good", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("seed WriteLoginToken: %v", err)
	}
	// Empty refresh token must not touch the file.
	if err := w.WriteLoginToken("default", "", time.Time{}, "user", "cid", "dev"); err != nil {
		t.Fatalf("empty WriteLoginToken: %v", err)
	}
	if got := loadCredsFile(t).Section("default").Key("refresh_token").String(); got != "rt-good" {
		t.Errorf("empty refresh_token overwrote prior value; got %q, want rt-good", got)
	}
}

// WriteLoginToken on one profile leaves a different profile's section untouched
// (per-profile isolation — the consolidation's reason for existing).
func TestWriteLoginToken_PerProfileIsolation(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteLoginToken("default", "rt-prod", time.Now().UTC(), "user", "cid-prod", "prod"); err != nil {
		t.Fatalf("default WriteLoginToken: %v", err)
	}
	if err := w.WriteLoginToken("dev", "rt-dev", time.Now().UTC(), "user", "cid-dev", "dev"); err != nil {
		t.Fatalf("dev WriteLoginToken: %v", err)
	}

	f := loadCredsFile(t)
	if got := f.Section("default").Key("refresh_token").String(); got != "rt-prod" {
		t.Errorf("default refresh_token=%q, want rt-prod", got)
	}
	if got := f.Section("dev").Key("refresh_token").String(); got != "rt-dev" {
		t.Errorf("dev refresh_token=%q, want rt-dev", got)
	}
}

// ClearLoginToken removes only the login keys, leaving machine creds intact.
func TestClearLoginToken_KeepsMachineCredentials(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	if err := w.WriteLoginToken("default", "rt-123", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}

	if err := w.ClearLoginToken("default"); err != nil {
		t.Fatalf("ClearLoginToken: %v", err)
	}
	s := loadCredsFile(t).Section("default")
	if s.Key("client_id").String() != "cid" {
		t.Errorf("client_id=%q, want cid (logout must keep machine creds)", s.Key("client_id").String())
	}
	if s.Key("client_secret").String() != "cs" {
		t.Errorf("client_secret=%q, want cs", s.Key("client_secret").String())
	}
	for _, k := range loginTokenKeys {
		if v := s.Key(k).String(); v != "" {
			t.Errorf("login key %q=%q after ClearLoginToken, want empty", k, v)
		}
	}
}

// ClearLoginToken is idempotent: a missing file, a missing section, and an
// already-cleared section are all non-errors.
func TestClearLoginToken_Idempotent(t *testing.T) {
	w := newHomeWriter(t)
	// Missing file entirely.
	if err := w.ClearLoginToken("default"); err != nil {
		t.Errorf("clear on missing file: %v", err)
	}
	// File exists but no section for the profile.
	if err := w.WriteCredentials("other", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	if err := w.ClearLoginToken("ghost"); err != nil {
		t.Errorf("clear on missing section: %v", err)
	}
	// Section exists but already cleared.
	if err := w.WriteCredentials("default", "cid", "cs"); err != nil {
		t.Fatalf("WriteCredentials default: %v", err)
	}
	if err := w.ClearLoginToken("default"); err != nil {
		t.Fatalf("clear default once: %v", err)
	}
	if err := w.ClearLoginToken("default"); err != nil {
		t.Errorf("clear default twice (already cleared): %v", err)
	}
}

// LoadConfig reads the login keys back into Config from the credentials INI.
func TestLoadConfig_ReadsLoginKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, k := range []string{"GRN_PROFILE", "GRN_ACCESS_KEY_ID", "GRN_SECRET_ACCESS_KEY", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID"} {
		t.Setenv(k, "")
	}
	dir := filepath.Join(home, ".greennode")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	writeFile(t, filepath.Join(dir, "credentials"),
		"[default]\n"+
			"auth_mode = user\n"+
			"refresh_token = rt-xyz\n"+
			"token_expires_at = "+exp.Format(time.RFC3339)+"\n"+
			"login_client_id = cid-login\n"+
			"iam_env = dev\n")

	cfg, err := LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AuthMode != "user" {
		t.Errorf("AuthMode=%q, want user", cfg.AuthMode)
	}
	if cfg.RefreshToken != "rt-xyz" {
		t.Errorf("RefreshToken=%q, want rt-xyz", cfg.RefreshToken)
	}
	if !cfg.TokenExpiresAt.Equal(exp) {
		t.Errorf("TokenExpiresAt=%v, want %v", cfg.TokenExpiresAt, exp)
	}
	if cfg.LoginClientID != "cid-login" {
		t.Errorf("LoginClientID=%q, want cid-login", cfg.LoginClientID)
	}
	if cfg.IamEnv != "dev" {
		t.Errorf("IamEnv=%q, want dev", cfg.IamEnv)
	}
}

// A login-only profile (no machine client_id/client_secret) "exists": LoadConfig
// must not reject it as "profile does not exist".
func TestLoadConfig_LoginOnlyProfileExists(t *testing.T) {
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
		"[default]\nauth_mode = user\nrefresh_token = rt\nlogin_client_id = cid\niam_env = dev\n")

	cfg, err := LoadConfig("default")
	if err != nil {
		t.Fatalf("login-only profile should exist, got error: %v", err)
	}
	if cfg.RefreshToken != "rt" {
		t.Errorf("RefreshToken=%q, want rt", cfg.RefreshToken)
	}
}

// The atomic save writes the credentials file at 0600 (refresh_token is
// secret-at-rest). Regression: the old O_TRUNC path admitted looser perms.
func TestSave_CredentialsFileIs0600(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteLoginToken("default", "rt", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}
	fi, err := os.Stat(credsPath(t))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("credentials perm=%o, want 0600", fi.Mode().Perm())
	}
	// Re-writing an existing file must re-assert 0600 even if perms drifted.
	if err := os.Chmod(credsPath(t), 0644); err != nil {
		t.Fatalf("chmod loosen: %v", err)
	}
	if err := w.WriteLoginToken("default", "rt2", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("rewrite WriteLoginToken: %v", err)
	}
	fi, _ = os.Stat(credsPath(t))
	if fi.Mode().Perm() != 0600 {
		t.Errorf("after rewrite, credentials perm=%o, want 0600 (save must tighten)", fi.Mode().Perm())
	}
}

// The atomic save leaves no leftover temp files in the config dir.
func TestSave_NoLeftoverTempFiles(t *testing.T) {
	w := newHomeWriter(t)
	if err := w.WriteLoginToken("default", "rt", time.Now().UTC(), "user", "cid", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(credsPath(t)))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".cfg-") {
			t.Errorf("leftover temp file %q after save", e.Name())
		}
	}
}
