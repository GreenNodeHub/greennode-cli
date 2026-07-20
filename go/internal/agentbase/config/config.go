// Package config handles environment context resolution and credential management
// for the GreenNode AgentBase CLI.
//
// Resolution priority (first wins):
//  1. --env flag (passed as envOverride to LoadWithEnv)
//  2. Environment variable (e.g. GREENNODE_ENV)
//  3. Field in ./.greennode.json (current working directory)
//  4. Default value (prod)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Env represents a deployment environment.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// Endpoints holds the resolved API base URLs for a given environment.
type Endpoints struct {
	Identity    string
	Runtime     string
	Memory      string
	OAuth2Token string
	Gateway     string
	Cr          string
	Policy      string
}

var endpointsByEnv = map[Env]Endpoints{
	EnvDev: {
		Identity:    "https://agentbase.api-dev.vngcloud.tech/identity",
		Runtime:     "https://pub-iamapis.api-dev.vngcloud.tech/agent-core-runtime",
		Memory:      "https://pub-iamapis.api-dev.vngcloud.tech/agent-core-memory",
		OAuth2Token: "https://pub-iamapis.api-dev.vngcloud.tech/accounts-api/v2/auth/token",
		Gateway:     "https://agentbase.api-dev.vngcloud.tech/gateway",
		Cr:          "https://agentbase.api-dev.vngcloud.tech/cr",
		Policy:      "https://agentbase.api-dev.vngcloud.tech/policy",
	},
	EnvProd: {
		Identity:    "https://agentbase.api.vngcloud.vn/identity",
		Runtime:     "https://agentbase.api.vngcloud.vn/runtime",
		Memory:      "https://agentbase.api.vngcloud.vn/memory",
		OAuth2Token: "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token",
		Gateway:     "https://agentbase.api.vngcloud.vn/gateway",
		Cr:          "https://agentbase.api.vngcloud.vn/cr",
		Policy:      "https://agentbase.api.vngcloud.vn/policy",
	},
}

// file is the on-disk representation of ./.greennode.json.
type file struct {
	Env              string `json:"env,omitempty"`
	ClientID         string `json:"client_id,omitempty"`
	ClientSecret     string `json:"client_secret,omitempty"`
	AgentIdentity    string `json:"agent_identity,omitempty"`
	RegistryURL      string `json:"registry_url,omitempty"`
	RegistryUsername string `json:"registry_username,omitempty"`
	RegistryPassword string `json:"registry_password,omitempty"`
}

// Config holds all resolved configuration values.
type Config struct {
	Env              Env
	ClientID         string
	ClientSecret     string
	AgentIdentity    string
	RegistryURL      string
	RegistryUsername string
	RegistryPassword string
	Endpoints        Endpoints
}

// LoadWithEnv resolves the active configuration, using envOverride as the
// highest-priority source for the environment field when it is non-empty.
// envOverride must be "dev" or "prod" if provided; any other value returns an error.
func LoadWithEnv(envOverride string) (*Config, error) {
	if envOverride != "" && envOverride != string(EnvDev) && envOverride != string(EnvProd) {
		return nil, fmt.Errorf("invalid environment %q: must be 'dev' or 'prod'", envOverride)
	}

	f, _ := readFile()

	var env Env
	if envOverride != "" {
		env = Env(envOverride)
	} else {
		env = resolveEnv(f)
	}

	eps, ok := endpointsByEnv[env]
	if !ok {
		eps = endpointsByEnv[EnvProd]
	}

	return &Config{
		Env:              env,
		ClientID:         resolveString("GREENNODE_CLIENT_ID", f.ClientID),
		ClientSecret:     resolveString("GREENNODE_CLIENT_SECRET", f.ClientSecret),
		AgentIdentity:    resolveString("GREENNODE_AGENT_IDENTITY", f.AgentIdentity),
		RegistryURL:      resolveString("GREENNODE_REGISTRY_URL", f.RegistryURL),
		RegistryUsername: resolveString("GREENNODE_REGISTRY_USERNAME", f.RegistryUsername),
		RegistryPassword: resolveString("GREENNODE_REGISTRY_PASSWORD", f.RegistryPassword),
		Endpoints:        eps,
	}, nil
}

// Load resolves the active configuration using the priority chain.
// To apply a per-invocation environment override, use LoadWithEnv instead.
func Load() (*Config, error) {
	return LoadWithEnv("")
}

// MustLoad loads config and exits on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading config:", err)
		os.Exit(1)
	}
	return cfg
}

// RequireCredentials ensures client_id and client_secret are set.
func (c *Config) RequireCredentials() error {
	if c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf("credentials not set — run 'greennode identity login' to authenticate")
	}
	return nil
}

// SetEnv writes the environment to ./.greennode.json.
func SetEnv(env Env) error {
	if env != EnvDev && env != EnvProd {
		return fmt.Errorf("invalid environment %q: must be 'dev' or 'prod'", env)
	}
	f, _ := readFile()
	f.Env = string(env)
	return writeFile(f)
}

// SaveCredentials writes client_id and client_secret to ./.greennode.json.
func SaveCredentials(clientID, clientSecret string) error {
	f, _ := readFile()
	f.ClientID = clientID
	f.ClientSecret = clientSecret
	return writeFile(f)
}

// SaveAgentIdentity writes agent_identity to ./.greennode.json.
func SaveAgentIdentity(name string) error {
	f, _ := readFile()
	f.AgentIdentity = name
	return writeFile(f)
}

// ClearCredentials removes credentials from ./.greennode.json.
func ClearCredentials() error {
	f, _ := readFile()
	f.ClientID = ""
	f.ClientSecret = ""
	f.AgentIdentity = ""
	return writeFile(f)
}

// GetEndpoints returns the resolved endpoints for a given environment string.
func GetEndpoints(env Env) Endpoints {
	if eps, ok := endpointsByEnv[env]; ok {
		return eps
	}
	return endpointsByEnv[EnvProd]
}

func resolveEnv(f *file) Env {
	if v := os.Getenv("GREENNODE_ENV"); v != "" {
		return Env(v)
	}
	if f.Env != "" {
		return Env(f.Env)
	}
	return EnvProd
}

func resolveString(envKey, fileVal string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fileVal
}

func configFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".greennode.json"), nil
}

func readFile() (*file, error) {
	path, err := configFilePath()
	if err != nil {
		return &file{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &file{}, nil
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return &file{}, err
	}
	return &f, nil
}

func writeFile(f *file) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
