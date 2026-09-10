package configure

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/config"
	loginpkg "github.com/greennodehub/greennode-cli/internal/login"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runSet,
}

func runSet(cmd *cobra.Command, args []string) error {
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

	// Preserve unrelated fields; new profiles start empty.
	cfg, err := config.LoadConfig(profile)
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	switch key {
	case "client_id":
		if err := writer.WriteCredentials(profile, value, cfg.ClientSecret); err != nil {
			return err
		}
	case "client_secret":
		if err := writer.WriteCredentials(profile, cfg.ClientID, value); err != nil {
			return err
		}
	case "region":
		if err := writer.WriteConfig(profile, value, cfg.Output, cfg.ProjectID); err != nil {
			return err
		}
	case "output":
		if err := writer.WriteConfig(profile, cfg.Region, value, cfg.ProjectID); err != nil {
			return err
		}
	case "project_id":
		if err := writer.WriteConfig(profile, cfg.Region, cfg.Output, value); err != nil {
			return err
		}
	// User tokens bind iam_env; switching requires login. Machine profiles may switch.
	case "iam_env":
		if cfg.AuthMode == "user" {
			return fmt.Errorf("iam_env is bound to the login token on a user profile; re-login with 'grn login --iam-env %s' to switch", value)
		}
		if _, err := loginpkg.TokenURLForEnv(value); err != nil {
			return err
		}
		if err := writer.WriteIamEnv(profile, value); err != nil {
			return err
		}
	// AgentBase's per-profile current agent.
	case "agent_identity":
		if err := writer.WriteAgentIdentity(profile, value); err != nil {
			return err
		}
	case "portal_user_id":
		if err := writer.WritePortalUserID(profile, value); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	fmt.Printf("Set '%s' to '%s' for profile '%s'.\n", key, displaySetValue(key, value), profile)
	return nil
}

// displaySetValue masks credentials before display.
func displaySetValue(key, value string) string {
	switch key {
	case "client_id", "client_secret":
		return config.MaskCredential(value)
	default:
		return value
	}
}
