package configure

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Report a missing profile like `aws configure list` does, rather than
	// printing empty values. LoadConfig only errors when config files exist but
	// the profile is in neither — a fresh machine still lists unset defaults.
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
	envMap := map[string]string{
		"client_id":     "GRN_ACCESS_KEY_ID",
		"client_secret": "GRN_SECRET_ACCESS_KEY",
	}
	if envVar, ok := envMap[name]; ok {
		if os.Getenv(envVar) != "" {
			return configEntry{name: name, value: config.MaskCredential(value), typ: "env", location: envVar}
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
		"region":     "GRN_DEFAULT_REGION",
		"output":     "GRN_DEFAULT_OUTPUT",
		"project_id": "GRN_DEFAULT_PROJECT_ID",
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
