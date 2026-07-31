package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/ini.v1"
)

// ConfigFileWriter creates/updates INI config files.
type ConfigFileWriter struct {
	configDir string
}

// NewConfigFileWriter creates a new writer targeting the default config directory.
func NewConfigFileWriter() *ConfigFileWriter {
	return &ConfigFileWriter{configDir: DefaultConfigDir()}
}

// ensureDir creates the config directory with proper permissions.
func (w *ConfigFileWriter) ensureDir() error {
	return os.MkdirAll(w.configDir, 0700)
}

// WriteCredentials writes client_id and client_secret for the given profile.
func (w *ConfigFileWriter) WriteCredentials(profile, clientID, clientSecret string) error {
	if err := w.ensureDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(w.configDir, "credentials")
	cfg, err := w.loadOrCreate(filePath)
	if err != nil {
		return err
	}

	section, err := cfg.NewSection(profile)
	if err != nil {
		return fmt.Errorf("failed to create section '%s': %w", profile, err)
	}
	section.Key("client_id").SetValue(clientID)
	section.Key("client_secret").SetValue(clientSecret)

	return w.save(cfg, filePath)
}

// WriteConfig writes region, output, and project_id for the given profile.
// An empty projectID is written as an empty key to explicitly clear any
// previously-saved value.
func (w *ConfigFileWriter) WriteConfig(profile, region, output, projectID string) error {
	if err := w.ensureDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(w.configDir, "config")
	cfg, err := w.loadOrCreate(filePath)
	if err != nil {
		return err
	}

	sectionName := profile
	if profile != "default" {
		sectionName = "profile " + profile
	}

	section, err := cfg.NewSection(sectionName)
	if err != nil {
		return fmt.Errorf("failed to create section '%s': %w", sectionName, err)
	}
	section.Key("region").SetValue(region)
	section.Key("output").SetValue(output)
	section.Key("project_id").SetValue(projectID)

	return w.save(cfg, filePath)
}

// loginTokenKeys are the per-section keys WriteLoginToken writes and
// ClearLoginToken removes. refresh_token is secret-at-rest (0600, masked in
// configure list/get); auth_mode/iam_env are non-secret refresh context;
// token_expires_at is a non-secret RFC3339 timestamp. The OAuth client_id is
// NOT persisted here — it is a public identifier baked into source
// (internal/login's per-env presets) and resolved from iam_env at refresh, so
// storing it in the credentials INI is redundant. ClearLoginToken still deletes
// a legacy login_client_id key if one is present from an older CLI version.
var loginTokenKeys = []string{"refresh_token", "token_expires_at", "auth_mode", "iam_env"}

// loginTokenKeysLegacy lists credential-section keys older CLI versions wrote
// that the current version no longer writes but should still clear on logout
// (so a logout fully removes a prior login). login_client_id was dropped from
// the persisted set because it is a public id already in source.
var loginTokenKeysLegacy = []string{"login_client_id"}

// WriteLoginToken persists a PKCE login result into the per-profile credentials
// INI: it folds the refresh token + non-secret refresh context into the same
// section that may already hold machine client_id/client_secret (auth-only
// merge — one identity file per profile). Mirrors WriteCredentials: it
// loadOrCreate's the file (preserving other keys/sections) and NewSection is
// idempotent (returns the existing section without wiping its keys). An empty
// refreshToken is a no-op so a stale empty value can never erase a prior good
// token; the caller (cmd/login) also skips the call on partial success.
func (w *ConfigFileWriter) WriteLoginToken(profile, refreshToken string, expiresAt time.Time, authMode, iamEnv string) error {
	if refreshToken == "" {
		return nil
	}
	if err := w.ensureDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(w.configDir, "credentials")
	cfg, err := w.loadOrCreate(filePath)
	if err != nil {
		return err
	}

	section, err := cfg.NewSection(profile)
	if err != nil {
		return fmt.Errorf("failed to create section '%s': %w", profile, err)
	}
	section.Key("refresh_token").SetValue(refreshToken)
	section.Key("token_expires_at").SetValue(expiresAt.UTC().Format(time.RFC3339))
	section.Key("auth_mode").SetValue(authMode)
	section.Key("iam_env").SetValue(iamEnv)

	return w.save(cfg, filePath)
}

// ClearLoginToken removes the login keys from a profile's credentials section
// (logout). It leaves machine client_id/client_secret intact and is idempotent:
// a missing file, missing section, or already-cleared section is not an error.
func (w *ConfigFileWriter) ClearLoginToken(profile string) error {
	filePath := filepath.Join(w.configDir, "credentials")
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // no credentials file → nothing to clear
		}
		return fmt.Errorf("failed to stat %s: %w", filePath, err)
	}

	cfg, err := w.loadOrCreate(filePath)
	if err != nil {
		return err
	}
	section, err := cfg.GetSection(profile)
	if err != nil {
		return nil // no section for this profile → nothing to clear
	}
	for _, k := range loginTokenKeys {
		section.DeleteKey(k)
	}
	for _, k := range loginTokenKeysLegacy {
		section.DeleteKey(k)
	}
	return w.save(cfg, filePath)
}

func (w *ConfigFileWriter) loadOrCreate(filePath string) (*ini.File, error) {
	if _, err := os.Stat(filePath); err == nil {
		return ini.Load(filePath)
	}
	return ini.Empty(), nil
}

// save writes the INI atomically: serialize to a same-directory temp file,
// chmod 0600, then rename over the target. Same-dir rename is atomic on POSIX
// and never crosses filesystems, so a crash mid-write cannot truncate the
// existing file — important now that `credentials` holds a refresh_token. The
// rename also re-asserts 0600 on an existing file whose perms may have drifted
// (the old O_TRUNC path could not tighten perms on an existing file).
func (w *ConfigFileWriter) save(cfg *ini.File, filePath string) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".cfg-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}
	if err := tmp.Chmod(0600); err != nil {
		cleanup()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}
	if _, err := cfg.WriteTo(tmp); err != nil {
		cleanup()
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp to %s: %w", filePath, err)
	}
	return nil
}
