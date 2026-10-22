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

// WriteCredentials selects machine auth and clears login tokens, preserving iam_env.
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
	section.Key("auth_mode").SetValue("machine")
	// Clear inactive login tokens; preserve iam_env.
	section.DeleteKey("refresh_token")
	section.DeleteKey("token_expires_at")

	return w.save(cfg, filePath)
}

// WriteConfig saves region, output, and project; empty project explicitly clears it.
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

// loginTokenKeys stores refresh credentials and context, never access tokens.
var loginTokenKeys = []string{"refresh_token", "token_expires_at", "auth_mode", "iam_env"}

// loginTokenKeysLegacy clears obsolete keys on logout.
var loginTokenKeysLegacy = []string{"login_client_id"}

// WriteLoginToken preserves other profile keys; empty tokens never overwrite existing ones.
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

// ClearLoginToken removes login keys idempotently, preserving machine credentials.
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

// WriteAgentIdentity updates one profile key; empty explicitly clears it.
func (w *ConfigFileWriter) WriteAgentIdentity(profile, name string) error {
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
	section.Key("agent_identity").SetValue(name)

	return w.save(cfg, filePath)
}

// WriteIamEnv updates the profile environment; callers validate token compatibility.
func (w *ConfigFileWriter) WriteIamEnv(profile, env string) error {
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
	section.Key("iam_env").SetValue(env)

	return w.save(cfg, filePath)
}

func (w *ConfigFileWriter) loadOrCreate(filePath string) (*ini.File, error) {
	if _, err := os.Stat(filePath); err == nil {
		return ini.Load(filePath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return ini.Empty(), nil
}

// save atomically replaces INI files with mode 0600.
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
