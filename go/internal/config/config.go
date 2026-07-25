package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/ini.v1"
)

// REGIONS maps region names to service endpoints.
var REGIONS = map[string]map[string]string{
	"HCM-3": {
		"vks_endpoint":     "https://vks.api.vngcloud.vn",
		"vserver_endpoint": "https://hcm-3.api.vngcloud.vn/vserver/vserver-gateway",
	},
	"HAN": {
		"vks_endpoint":     "https://vks-han-1.api.vngcloud.vn",
		"vserver_endpoint": "https://han-1.api.vngcloud.vn/vserver/vserver-gateway",
	},
}

// Config holds the resolved CLI configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	Region       string
	Output       string
	Profile      string
	ProjectID    string
	Regions      map[string]map[string]string
}

// DefaultConfigDir returns ~/.greennode. This is where config is written.
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".greennode")
}

// legacyConfigDir returns the pre-rename config directory (~/.greenode), kept as
// a read-only fallback so users who configured the CLI before the rename keep
// loading without re-running `grn configure`.
func legacyConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".greenode")
}

// effectiveConfigDir returns the directory to READ config from: the preferred
// ~/.greennode if it exists, otherwise the legacy ~/.greenode if present. Writes
// always target DefaultConfigDir; this helper only affects reads, so a legacy
// install transparently keeps working until the user re-runs `grn configure`,
// which migrates them onto ~/.greennode.
func effectiveConfigDir() string {
	dir := DefaultConfigDir()
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	if legacy := legacyConfigDir(); legacy != dir {
		if _, err := os.Stat(legacy); err == nil {
			return legacy
		}
	}
	return dir
}

// LoadConfig loads configuration for the given profile.
// Resolution order: env vars -> config files.
func LoadConfig(profile string) (*Config, error) {
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	configDir := effectiveConfigDir()
	cfg := &Config{
		Profile: profile,
		Regions: REGIONS,
	}

	// A profile "exists" if it has a section in the credentials file, the config
	// file, or credentials are supplied via env vars. credentials and config are
	// read independently so a profile created by `configure set region` (config
	// file only, no credentials yet) still loads instead of erroring.
	foundProfile := false
	anyFileExists := false

	// Load credentials — env vars override file
	if v := os.Getenv("GRN_ACCESS_KEY_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv("GRN_SECRET_ACCESS_KEY"); v != "" {
		cfg.ClientSecret = v
	}
	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		foundProfile = true
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		credsFile := filepath.Join(configDir, "credentials")
		if _, err := os.Stat(credsFile); err == nil {
			anyFileExists = true
			iniCreds, err := ini.Load(credsFile)
			if err != nil {
				return nil, fmt.Errorf("failed to parse credentials file: %w", err)
			}
			// Missing section is not fatal — the profile may live in the config
			// file only. Just skip credentials for this profile.
			if section, err := iniCreds.GetSection(profile); err == nil {
				foundProfile = true
				if cfg.ClientID == "" {
					cfg.ClientID = section.Key("client_id").String()
				}
				if cfg.ClientSecret == "" {
					cfg.ClientSecret = section.Key("client_secret").String()
				}
			}
		}
	}

	// Load config file (independent of credentials)
	configFile := filepath.Join(configDir, "config")
	if _, err := os.Stat(configFile); err == nil {
		anyFileExists = true
		iniCfg, err := ini.Load(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		sectionName := profile
		if profile != "default" {
			sectionName = "profile " + profile
		}

		section, serr := iniCfg.GetSection(sectionName)
		if serr != nil && profile == "default" {
			// Try the root DEFAULT section for the default profile.
			if root := iniCfg.Section(ini.DefaultSection); len(root.Keys()) > 0 {
				section, serr = root, nil
			}
		}
		if serr == nil && section != nil {
			foundProfile = true
			if v := section.Key("region").String(); v != "" {
				cfg.Region = v
			}
			if v := section.Key("output").String(); v != "" {
				cfg.Output = v
			}
			if v := section.Key("project_id").String(); v != "" {
				cfg.ProjectID = v
			}
		}
	}

	// Config files exist but the profile is in neither — report it like
	// `aws configure` does ("profile could not be found") so reads (get/list)
	// and API clients fail clearly instead of acting on empty config.
	if anyFileExists && !foundProfile {
		return nil, fmt.Errorf("profile '%s' does not exist (run 'grn configure --profile %s' to create it)", profile, profile)
	}

	// Env var overrides for region
	if v := os.Getenv("GRN_DEFAULT_REGION"); v != "" {
		cfg.Region = v
	}

	// Env var override for project_id
	if v := os.Getenv("GRN_DEFAULT_PROJECT_ID"); v != "" {
		cfg.ProjectID = v
	}

	// Default output
	if cfg.Output == "" {
		cfg.Output = "json"
	}

	return cfg, nil
}

// RegionNames returns the configured region names (keys of REGIONS), sorted.
func RegionNames() []string {
	names := make([]string, 0, len(REGIONS))
	for name := range REGIONS {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ProfileNames returns profile (section) names from the credentials file, sorted.
// Returns nil on any error so completion stays silent.
func ProfileNames() []string {
	path := filepath.Join(effectiveConfigDir(), "credentials")
	f, err := ini.Load(path)
	if err != nil {
		return nil
	}
	var names []string
	for _, s := range f.Sections() {
		name := s.Name()
		if name == ini.DefaultSection {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetEndpoint returns the service endpoint for the configured region.
func (c *Config) GetEndpoint(serviceName string) (string, error) {
	if c.Region == "" {
		return "", fmt.Errorf("region is not configured. Use 'grn configure' or the --region flag")
	}
	regionConfig, ok := c.Regions[c.Region]
	if !ok {
		return "", fmt.Errorf("invalid region: %s", c.Region)
	}
	endpointKey := serviceName + "_endpoint"
	endpoint, ok := regionConfig[endpointKey]
	if !ok {
		return "", fmt.Errorf("endpoint not found for service '%s' in region '%s'", serviceName, c.Region)
	}
	return endpoint, nil
}

// MaskCredential masks a credential string showing only last 4 chars.
func MaskCredential(value string) string {
	if len(value) <= 4 {
		return value
	}
	return "****************" + value[len(value)-4:]
}
