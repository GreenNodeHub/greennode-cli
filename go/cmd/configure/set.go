package configure

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run:   runSet,
}

func runSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]
	profile := cmd.Flag("profile").Value.String()
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	writer := config.NewConfigFileWriter()

	// Load existing config so unrelated fields are preserved on write. For a
	// brand-new profile LoadConfig returns (nil, err); fall back to empty
	// defaults so the value can still be set instead of panicking.
	cfg, err := config.LoadConfig(profile)
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	switch key {
	case "client_id":
		if err := writer.WriteCredentials(profile, value, cfg.ClientSecret); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "client_secret":
		if err := writer.WriteCredentials(profile, cfg.ClientID, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "region":
		if err := writer.WriteConfig(profile, value, cfg.Output, cfg.ProjectID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "output":
		if err := writer.WriteConfig(profile, cfg.Region, value, cfg.ProjectID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "project_id":
		if err := writer.WriteConfig(profile, cfg.Region, cfg.Output, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown configuration key: %s\n", key)
		os.Exit(1)
	}

	fmt.Printf("Set '%s' to '%s' for profile '%s'.\n", key, displaySetValue(key, value), profile)
}

// displaySetValue masks credential values so `configure set` never echoes a
// secret in plaintext (matching how `configure list` masks them). Non-sensitive
// values are shown as-is.
func displaySetValue(key, value string) string {
	switch key {
	case "client_id", "client_secret":
		return config.MaskCredential(value)
	default:
		return value
	}
}
