package configure

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run:   runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	key := args[0]
	profile := cmd.Flag("profile").Value.String()
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	cfg, err := config.LoadConfig(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var value string
	switch key {
	case "client_id":
		value = config.MaskCredential(cfg.ClientID)
	case "client_secret":
		value = config.MaskCredential(cfg.ClientSecret)
	case "region":
		value = cfg.Region
	case "output":
		value = cfg.Output
	case "profile":
		value = cfg.Profile
	case "project_id":
		value = cfg.ProjectID
	default:
		fmt.Fprintf(os.Stderr, "Unknown configuration key: %s\n", key)
		os.Exit(1)
	}

	if value == "" {
		value = "<not set>"
	}
	fmt.Println(value)
}
