package configure

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/config"
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
	// Login (user) identity. refresh_token is secret-at-rest → masked (a user
	// can confirm a login token is present without seeing it). auth_mode,
	// iam_env and the expiry are non-secret refresh context. The client_id is
	// not stored here — it is resolved from iam_env at refresh.
	case "refresh_token":
		value = config.MaskCredential(cfg.RefreshToken)
	case "auth_mode":
		value = cfg.AuthMode
	case "iam_env":
		value = cfg.IamEnv
	case "token_expires_at":
		if !cfg.TokenExpiresAt.IsZero() {
			value = cfg.TokenExpiresAt.UTC().Format(time.RFC3339)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown configuration key: %s\n", key)
		os.Exit(1)
	}

	if value == "" {
		value = "<not set>"
	}
	fmt.Println(value)
}
