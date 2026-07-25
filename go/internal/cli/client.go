// Package cli holds CLI infrastructure shared across all product services
// (client construction, output formatting, common flag parsing, and the
// service registry). It is service-agnostic: callers pass their service name.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/auth"
	"github.com/vngcloud/greennode-cli/internal/client"
	"github.com/vngcloud/greennode-cli/internal/config"
)

// NewClient builds a GreennodeClient for the given service from the command's
// global flags. The endpoint is resolved per service via config.GetEndpoint.
func NewClient(cmd *cobra.Command, serviceName string) (*client.GreennodeClient, error) {
	profile, _ := cmd.Flags().GetString("profile")
	region, _ := cmd.Flags().GetString("region")
	endpointURL, _ := cmd.Flags().GetString("endpoint-url")
	noVerifySSL, _ := cmd.Flags().GetBool("no-verify-ssl")
	debug, _ := cmd.Flags().GetBool("debug")
	allowUntrusted, _ := cmd.Flags().GetBool("allow-untrusted-endpoint")
	connectTimeout, _ := cmd.Flags().GetInt("cli-connect-timeout")
	readTimeout, _ := cmd.Flags().GetInt("cli-read-timeout")

	if err := CheckEndpoint(endpointURL, noVerifySSL, allowUntrusted); err != nil {
		return nil, err
	}

	cfg, err := config.LoadConfig(profile)
	if err != nil {
		return nil, err
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("credentials not configured. Run 'grn configure' to set up credentials")
	}

	if region != "" {
		cfg.Region = region
	}

	var baseURL string
	if endpointURL != "" {
		baseURL = endpointURL
	} else {
		baseURL, err = cfg.GetEndpoint(serviceName)
		if err != nil {
			return nil, err
		}
	}

	if noVerifySSL {
		fmt.Fprintln(os.Stderr, "Warning: SSL certificate verification is disabled. This is not recommended for production use.")
	}

	tokenManager := auth.NewTokenManager(cfg.ClientID, cfg.ClientSecret)
	connect := time.Duration(connectTimeout) * time.Second
	read := time.Duration(readTimeout) * time.Second

	return client.NewGreennodeClient(baseURL, tokenManager, connect, read, !noVerifySSL, debug), nil
}
