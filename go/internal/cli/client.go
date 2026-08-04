// Package cli holds CLI infrastructure shared across all product services
// (client construction, output formatting, common flag parsing, and the
// service registry). It is service-agnostic: callers pass their service name.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
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

	// Auth source is profile-driven (one auth type per profile): auth_mode=user
	// → login refresh-token provider; else → machine client_credentials. The
	// machine-cred check lives inside the provider picker's default branch, so a
	// login-only profile (no machine creds) now works and a machine profile is
	// unchanged. NewTokenProvider reads the RESOLVED profile off cfg (cfg.Profile,
	// set by LoadConfig) — not the raw --profile flag, which may be "".
	tp, err := NewTokenProvider(cfg)
	if err != nil {
		return nil, err
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

	connect := time.Duration(connectTimeout) * time.Second
	read := time.Duration(readTimeout) * time.Second

	return client.NewGreennodeClient(baseURL, tp, connect, read, !noVerifySSL, debug), nil
}
