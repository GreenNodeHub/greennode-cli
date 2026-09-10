package configure

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/config"
	loginpkg "github.com/greennodehub/greennode-cli/internal/login"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setCmd = &cobra.Command{
	Use:   "set <key> [value]",
	Short: "Set a configuration value",
	Long:  "Set a value. Omit client_secret to enter it without terminal echo.",
	Args:  validateSetArgs,
	RunE:  runSet,
}

func validateSetArgs(_ *cobra.Command, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
	}
	if len(args) == 1 && args[0] != "client_secret" {
		return fmt.Errorf("a value is required for %s", args[0])
	}
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := ""
	if len(args) == 2 {
		value = args[1]
	} else {
		var err error
		value, err = promptForClientSecret()
		if err != nil {
			return err
		}
	}
	profile := cmd.Flag("profile").Value.String()
	if profile == "" {
		profile = os.Getenv("GRN_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	writer := config.NewConfigFileWriter()

	// Preserve persisted fields without saving environment overrides.
	cfg, err := config.LoadProfileFiles(profile)
	if errors.Is(err, config.ErrProfileNotFound) {
		cfg = &config.Config{}
	} else if err != nil {
		return err
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

func promptForClientSecret() (string, error) {
	if cli.IsNonInteractive() {
		return "", fmt.Errorf("client_secret value is required in non-interactive mode")
	}
	fmt.Fprint(os.Stdout, "Client Secret: ")
	var value string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		secret, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stdout)
		if err != nil {
			return "", fmt.Errorf("read client secret: %w", err)
		}
		value = string(secret)
	} else {
		secret, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read client secret: %w", err)
		}
		value = secret
	}
	value = strings.TrimRight(value, "\r\n")
	if value == "" {
		return "", fmt.Errorf("client secret cannot be empty")
	}
	return value, nil
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
