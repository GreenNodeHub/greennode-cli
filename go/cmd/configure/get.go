package configure

import (
	"fmt"
	"os"
	"time"

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
	case "portal_user_id":
		value = cfg.PortalUserID
	// Mask stored refresh tokens; show non-secret auth context.
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
	// AgentBase's per-profile current agent.
	case "agent_identity":
		value = cfg.AgentIdentity
	default:
		fmt.Fprintf(os.Stderr, "Unknown configuration key: %s\n", key)
		os.Exit(1)
	}

	if value == "" {
		value = "<not set>"
	}
	fmt.Println(value)
}
