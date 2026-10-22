package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPortalUserIDPreservesExistingProfilesAndLogin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, key := range []string{"GRN_PROFILE", "GRN_CLIENT_ID", "GRN_CLIENT_SECRET", "GRN_DEFAULT_REGION", "GRN_DEFAULT_PROJECT_ID", "GRN_PORTAL_USER_ID"} {
		t.Setenv(key, "")
	}
	w := NewConfigFileWriter()
	if err := w.WriteConfig("existing", "HAN", "table", "project"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteConfig("other", "HCM-3", "text", "other-project"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLoginToken("existing", "old-refresh", time.Now().Add(time.Hour), "user", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteAgentIdentity("existing", "agent"); err != nil {
		t.Fatal(err)
	}
	credPath := filepath.Join(DefaultConfigDir(), "credentials")
	before, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePortalUserID("existing", "2147483647"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteConfig("existing", "HCM-3", "json", "new-project"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig("existing")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PortalUserID != "2147483647" || cfg.RefreshToken != "old-refresh" || cfg.AuthMode != "user" || cfg.IamEnv != "dev" || cfg.AgentIdentity != "agent" {
		t.Fatal("updating config lost existing fields")
	}
	other, err := LoadConfig("other")
	if err != nil || other.ProjectID != "other-project" || other.Output != "text" {
		t.Fatal("updating config lost another profile")
	}
	after, err := os.ReadFile(credPath)
	if err != nil || string(after) != string(before) {
		t.Fatal("portal update rewrote credentials")
	}
	configPath := filepath.Join(DefaultConfigDir(), "config")
	info, err := os.Stat(configPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("config permissions not 0600")
	}
	baseline, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"0", "-1", "2147483648", "user-id", "1.5"} {
		if err := w.WritePortalUserID("existing", value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	unchanged, err := os.ReadFile(configPath)
	if err != nil || string(unchanged) != string(baseline) {
		t.Fatal("invalid value changed config")
	}
	t.Setenv("GRN_PORTAL_USER_ID", "7")
	cfg, err = LoadConfig("existing")
	if err != nil || cfg.PortalUserID != "7" {
		t.Fatal("env override ignored")
	}
	t.Setenv("GRN_PORTAL_USER_ID", "")
	if err := w.WritePortalUserID("existing", ""); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig("existing")
	if err != nil || cfg.PortalUserID != "" {
		t.Fatal("explicit clear ignored")
	}
}

func TestFoundationRegionEndpointsAreServiceSpecific(t *testing.T) {
	for _, tc := range []struct{ region, service, endpoint string }{
		{"HCM-3", "vbackup", "https://hcm-3.api.vngcloud.vn/vbackup-gateway"},
		{"HCM-3", "vlb", "https://hcm-3.api.vngcloud.vn/vserver/vlb-gateway"},
		{"HAN", "vlb", "https://han-1.api.vngcloud.vn/vserver/vlb-gateway"},
		{"HCM-3", "vstorage", "https://hcm03-api.vstorage.vngcloud.vn"},
		{"HAN", "vstorage", "https://han02-api.vstorage.vngcloud.vn"},
		{"HCM-4", "vstorage", "https://hcm04-api.vstorage.vngcloud.vn"},
	} {
		cfg := &Config{Region: tc.region, Regions: REGIONS}
		got, err := cfg.GetEndpoint(tc.service)
		if err != nil || got != tc.endpoint {
			t.Errorf("%s/%s = %q, %v", tc.region, tc.service, got, err)
		}
	}
	for _, region := range []string{"HAN", "HCM-4"} {
		cfg := &Config{Region: region, Regions: REGIONS}
		if _, err := cfg.GetEndpoint("vbackup"); err == nil {
			t.Errorf("invented vbackup endpoint in %s", region)
		}
	}
}

func TestPortalUserIDRejectsMalformedExistingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	w := NewConfigFileWriter()
	if err := w.ensureDir(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(DefaultConfigDir(), "config")
	before := "[unterminated\n"
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.WritePortalUserID("default", "7"); err == nil {
		t.Fatal("invalid INI overwritten")
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.EqualFold(string(data), before) {
		t.Fatal("invalid INI changed")
	}
}

func TestCredentialEnvironmentAliases(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("GRN_ACCESS_KEY_ID", "fixture-legacy-id")
	t.Setenv("GRN_SECRET_ACCESS_KEY", "fixture-legacy-secret")
	cfg, err := LoadConfig("default")
	if err != nil || cfg.ClientID != "fixture-legacy-id" || cfg.ClientSecret != "fixture-legacy-secret" {
		t.Fatalf("legacy credential aliases not loaded: %#v, %v", cfg, err)
	}

	t.Setenv("GRN_CLIENT_ID", "fixture-current-id")
	t.Setenv("GRN_CLIENT_SECRET", "fixture-current-secret")
	cfg, err = LoadConfig("default")
	if err != nil || cfg.ClientID != "fixture-current-id" || cfg.ClientSecret != "fixture-current-secret" {
		t.Fatalf("current credential variables must win: %#v, %v", cfg, err)
	}
}

func TestLoadProfileFilesIgnoresEnvironment(t *testing.T) {
	isolateConfigEnv(t)
	writer := NewConfigFileWriter()
	if err := writer.WriteCredentials("default", "fixture-file-id", "fixture-file-secret"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteConfig("default", "HAN", "json", "fixture-file-project"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRN_CLIENT_ID", "fixture-env-id")
	t.Setenv("GRN_CLIENT_SECRET", "fixture-env-secret")
	t.Setenv("GRN_DEFAULT_PROJECT_ID", "fixture-env-project")

	cfg, err := LoadProfileFiles("default")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "fixture-file-id" || cfg.ClientSecret != "fixture-file-secret" || cfg.ProjectID != "fixture-file-project" {
		t.Fatalf("persisted profile was overlaid by environment: %#v", cfg)
	}
}
