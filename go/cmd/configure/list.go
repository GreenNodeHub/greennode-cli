package configure

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration values",
	Run:   runList,
}

type configEntry struct {
	name     string
	value    string
	typ      string
	location string
}

func runList(cmd *cobra.Command, args []string) {
	profile := cmd.Flag("profile").Value.String()
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	// Report missing profiles; fresh installations show unset defaults.
	cfg, err := config.LoadConfig(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	configDir := config.DefaultConfigDir()
	credsFile := filepath.Join(configDir, "credentials")
	configFile := filepath.Join(configDir, "config")

	entries := []configEntry{
		resolveEntry("profile", profile, "", ""),
		resolveCredEntry("client_id", cfg.ClientID, credsFile),
		resolveCredEntry("client_secret", cfg.ClientSecret, credsFile),
		resolveConfigEntry("region", cfg.Region, configFile),
		resolveConfigEntry("output", cfg.Output, configFile),
		resolveConfigEntry("project_id", cfg.ProjectID, configFile),
		resolveConfigEntry("portal_user_id", cfg.PortalUserID, configFile),
		// Store refresh tokens, never access tokens, in profile credentials.
		resolveCredEntry("refresh_token", cfg.RefreshToken, credsFile),
		resolveCredEntryPlain("auth_mode", cfg.AuthMode, credsFile),
		resolveCredEntryPlain("iam_env", cfg.IamEnv, credsFile),
		resolveCredEntryPlain("token_expires_at", tokenExpiresAtStr(cfg.TokenExpiresAt), credsFile),
		// AgentBase's per-profile current agent.
		resolveCredEntryPlain("agent_identity", cfg.AgentIdentity, credsFile),
	}

	// Print header
	fmt.Printf("%13s %24s %15s    %s\n", "Name", "Value", "Type", "Location")
	fmt.Printf("%13s %24s %15s    %s\n", "----", "-----", "----", "--------")

	for _, e := range entries {
		fmt.Printf("%13s %24s %15s    %s\n", e.name, e.value, e.typ, e.location)
	}
}

func resolveEntry(name, value, typ, location string) configEntry {
	if value == "" {
		return configEntry{name: name, value: "<not set>", typ: "None", location: "None"}
	}
	if typ == "" {
		typ = "None"
	}
	if location == "" {
		location = "None"
	}
	return configEntry{name: name, value: value, typ: typ, location: location}
}

func resolveCredEntry(name, value, credsFile string) configEntry {
	if value == "" {
		return configEntry{name: name, value: "<not set>", typ: "None", location: "None"}
	}

	// Check if value came from env var
	envMap := map[string][]string{
		"client_id":     {"GRN_CLIENT_ID", "GRN_ACCESS_KEY_ID"},
		"client_secret": {"GRN_CLIENT_SECRET", "GRN_SECRET_ACCESS_KEY"},
	}
	if envVars, ok := envMap[name]; ok {
		for _, envVar := range envVars {
			if os.Getenv(envVar) != "" {
				return configEntry{name: name, value: config.MaskCredential(value), typ: "env", location: envVar}
			}
		}
	}

	home, _ := os.UserHomeDir()
	loc := "~" + credsFile[len(home):]
	return configEntry{name: name, value: config.MaskCredential(value), typ: "config-file", location: loc}
}

func resolveConfigEntry(name, value, configFile string) configEntry {
	if value == "" {
		return configEntry{name: name, value: "<not set>", typ: "None", location: "None"}
	}

	// Check if value came from env var
	envMap := map[string]string{
		"region":         "GRN_DEFAULT_REGION",
		"output":         "GRN_DEFAULT_OUTPUT",
		"project_id":     "GRN_DEFAULT_PROJECT_ID",
		"portal_user_id": "GRN_PORTAL_USER_ID",
	}
	if envVar, ok := envMap[name]; ok {
		if os.Getenv(envVar) != "" {
			return configEntry{name: name, value: value, typ: "env", location: envVar}
		}
	}

	home, _ := os.UserHomeDir()
	loc := "~" + configFile[len(home):]
	return configEntry{name: name, value: value, typ: "config-file", location: loc}
}

// resolveCredEntryPlain locates non-secret credential keys without masking.
func resolveCredEntryPlain(name, value, credsFile string) configEntry {
	if value == "" {
		return configEntry{name: name, value: "<not set>", typ: "None", location: "None"}
	}
	home, _ := os.UserHomeDir()
	loc := "~" + credsFile[len(home):]
	return configEntry{name: name, value: value, typ: "config-file", location: loc}
}

// tokenExpiresAtStr returns RFC3339 expiry or empty.
func tokenExpiresAtStr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
