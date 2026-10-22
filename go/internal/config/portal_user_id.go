package config

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// ValidatePositiveInt32 checks the numeric portal-user header contract.
func ValidatePositiveInt32(value string) (int32, error) {
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("value must be a positive 32-bit integer")
	}
	return int32(n), nil
}

// WritePortalUserID preserves other keys; empty explicitly clears the value.
func (w *ConfigFileWriter) WritePortalUserID(profile, value string) error {
	if value != "" {
		if _, err := ValidatePositiveInt32(value); err != nil {
			return fmt.Errorf("portal_user_id: %w", err)
		}
	}
	if err := w.ensureDir(); err != nil {
		return err
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
		return err
	}
	section.Key("portal_user_id").SetValue(value)
	return w.save(cfg, filePath)
}
