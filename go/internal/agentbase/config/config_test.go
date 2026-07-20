package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// withTempCwd changes the working directory to a temporary directory for the
// duration of the test, then restores the original directory.
func withTempCwd(t *testing.T, f func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(orig)
	}()
	f()
}

func writeConfig(t *testing.T, data file) {
	t.Helper()
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, ".greennode.json")
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}

func TestDefaultEnvIsProd(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_ENV")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvProd {
			t.Errorf("expected prod, got %s", cfg.Env)
		}
	})
}

func TestEnvVarOverridesConfig(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{Env: "prod"})
		t.Setenv("GREENNODE_ENV", "dev")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvDev {
			t.Errorf("expected dev, got %s", cfg.Env)
		}
	})
}

func TestConfigFileEnv(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_ENV")
		writeConfig(t, file{Env: "dev"})
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvDev {
			t.Errorf("expected dev, got %s", cfg.Env)
		}
	})
}

func TestCredentialsFromEnvVar(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_ENV")
		t.Setenv("GREENNODE_CLIENT_ID", "env-id")
		t.Setenv("GREENNODE_CLIENT_SECRET", "env-secret")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ClientID != "env-id" {
			t.Errorf("expected env-id, got %s", cfg.ClientID)
		}
		if cfg.ClientSecret != "env-secret" {
			t.Errorf("expected env-secret, got %s", cfg.ClientSecret)
		}
	})
}

func TestCredentialsFromFile(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_CLIENT_ID")
		os.Unsetenv("GREENNODE_CLIENT_SECRET")
		writeConfig(t, file{ClientID: "file-id", ClientSecret: "file-secret"})
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ClientID != "file-id" {
			t.Errorf("expected file-id, got %s", cfg.ClientID)
		}
	})
}

func TestEnvVarOverridesFileCredentials(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{ClientID: "file-id", ClientSecret: "file-secret"})
		t.Setenv("GREENNODE_CLIENT_ID", "env-id")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ClientID != "env-id" {
			t.Errorf("env var should override file: expected env-id, got %s", cfg.ClientID)
		}
	})
}

func TestRequireCredentials(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_CLIENT_ID")
		os.Unsetenv("GREENNODE_CLIENT_SECRET")
		cfg, _ := Load()
		if err := cfg.RequireCredentials(); err == nil {
			t.Error("expected error for missing credentials")
		}
	})
}

func TestRequireCredentialsSuccess(t *testing.T) {
	withTempCwd(t, func() {
		t.Setenv("GREENNODE_CLIENT_ID", "id")
		t.Setenv("GREENNODE_CLIENT_SECRET", "secret")
		cfg, _ := Load()
		if err := cfg.RequireCredentials(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestSetEnvWritesFile(t *testing.T) {
	withTempCwd(t, func() {
		if err := SetEnv(EnvDev); err != nil {
			t.Fatal(err)
		}
		f, err := readFile()
		if err != nil {
			t.Fatal(err)
		}
		if f.Env != "dev" {
			t.Errorf("expected dev, got %s", f.Env)
		}
	})
}

func TestSetEnvInvalid(t *testing.T) {
	withTempCwd(t, func() {
		if err := SetEnv("staging"); err == nil {
			t.Error("expected error for invalid env")
		}
	})
}

func TestSaveCredentials(t *testing.T) {
	withTempCwd(t, func() {
		if err := SaveCredentials("my-id", "my-secret"); err != nil {
			t.Fatal(err)
		}
		f, _ := readFile()
		if f.ClientID != "my-id" || f.ClientSecret != "my-secret" {
			t.Errorf("credentials not saved: id=%s secret=%s", f.ClientID, f.ClientSecret)
		}
	})
}

func TestClearCredentials(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{ClientID: "id", ClientSecret: "secret", AgentIdentity: "identity"})
		if err := ClearCredentials(); err != nil {
			t.Fatal(err)
		}
		f, _ := readFile()
		if f.ClientID != "" || f.ClientSecret != "" || f.AgentIdentity != "" {
			t.Error("credentials not cleared")
		}
	})
}

func TestDevEndpoints(t *testing.T) {
	eps := GetEndpoints(EnvDev)
	if eps.Identity == "" || eps.Runtime == "" || eps.Memory == "" || eps.OAuth2Token == "" {
		t.Error("dev endpoints should not be empty")
	}
	if eps.Identity == GetEndpoints(EnvProd).Identity {
		t.Error("dev and prod identity endpoints should differ")
	}
}

func TestProdEndpoints(t *testing.T) {
	eps := GetEndpoints(EnvProd)
	if eps.Identity == "" || eps.Runtime == "" {
		t.Error("prod endpoints should not be empty")
	}
}

func TestLoadResolvesDevEndpoints(t *testing.T) {
	withTempCwd(t, func() {
		t.Setenv("GREENNODE_ENV", "dev")
		cfg, _ := Load()
		devEps := GetEndpoints(EnvDev)
		if cfg.Endpoints.Identity != devEps.Identity {
			t.Errorf("expected dev identity endpoint, got %s", cfg.Endpoints.Identity)
		}
	})
}

func TestMustLoad(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_ENV")
		cfg := MustLoad()
		if cfg == nil {
			t.Fatal("expected non-nil config from MustLoad")
		}
		if cfg.Env != EnvProd {
			t.Errorf("expected prod, got %s", cfg.Env)
		}
	})
}

func TestSaveAgentIdentity(t *testing.T) {
	withTempCwd(t, func() {
		if err := SaveAgentIdentity("my-agent"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.AgentIdentity != "my-agent" {
			t.Errorf("expected my-agent, got %s", cfg.AgentIdentity)
		}
		// Overwrite to verify it replaces correctly.
		if err := SaveAgentIdentity("other-agent"); err != nil {
			t.Fatal(err)
		}
		cfg, _ = Load()
		if cfg.AgentIdentity != "other-agent" {
			t.Errorf("expected other-agent, got %s", cfg.AgentIdentity)
		}
	})
}

func TestLoadWithEnvOverridesDev(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{Env: "prod"})
		t.Setenv("GREENNODE_ENV", "prod")
		cfg, err := LoadWithEnv("dev")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvDev {
			t.Errorf("expected dev, got %s", cfg.Env)
		}
		devEps := GetEndpoints(EnvDev)
		if cfg.Endpoints.Identity != devEps.Identity {
			t.Errorf("expected dev identity endpoint, got %s", cfg.Endpoints.Identity)
		}
	})
}

func TestLoadWithEnvOverridesProd(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{Env: "dev"})
		t.Setenv("GREENNODE_ENV", "dev")
		cfg, err := LoadWithEnv("prod")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvProd {
			t.Errorf("expected prod, got %s", cfg.Env)
		}
		prodEps := GetEndpoints(EnvProd)
		if cfg.Endpoints.Runtime != prodEps.Runtime {
			t.Errorf("expected prod runtime endpoint, got %s", cfg.Endpoints.Runtime)
		}
	})
}

func TestLoadWithEnvInvalidReturnsError(t *testing.T) {
	withTempCwd(t, func() {
		_, err := LoadWithEnv("staging")
		if err == nil {
			t.Fatal("expected error for invalid env override")
		}
	})
}

func TestLoadWithEnvEmptyFallsThrough(t *testing.T) {
	withTempCwd(t, func() {
		os.Unsetenv("GREENNODE_ENV")
		writeConfig(t, file{Env: "dev"})
		cfg, err := LoadWithEnv("")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Env != EnvDev {
			t.Errorf("empty override should fall through to config file: expected dev, got %s", cfg.Env)
		}
	})
}

func TestRegistryFromFile(t *testing.T) {
	withTempCwd(t, func() {
		t.Setenv("GREENNODE_REGISTRY_URL", "")
		t.Setenv("GREENNODE_REGISTRY_USERNAME", "")
		t.Setenv("GREENNODE_REGISTRY_PASSWORD", "")
		writeConfig(t, file{
			RegistryURL:      "https://reg.file",
			RegistryUsername: "file-user",
			RegistryPassword: "file-pass",
		})
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.RegistryURL != "https://reg.file" || cfg.RegistryUsername != "file-user" || cfg.RegistryPassword != "file-pass" {
			t.Fatalf("unexpected registry: %+v", cfg)
		}
	})
}

func TestRegistryEnvOverridesFile(t *testing.T) {
	withTempCwd(t, func() {
		writeConfig(t, file{
			RegistryURL:      "https://from-file",
			RegistryUsername: "from-file-user",
			RegistryPassword: "from-file-pass",
		})
		t.Setenv("GREENNODE_REGISTRY_URL", "https://from-env")
		t.Setenv("GREENNODE_REGISTRY_USERNAME", "")
		t.Setenv("GREENNODE_REGISTRY_PASSWORD", "")
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.RegistryURL != "https://from-env" {
			t.Errorf("expected env URL, got %q", cfg.RegistryURL)
		}
		if cfg.RegistryUsername != "from-file-user" {
			t.Errorf("expected file username when env unset, got %q", cfg.RegistryUsername)
		}
	})
}
