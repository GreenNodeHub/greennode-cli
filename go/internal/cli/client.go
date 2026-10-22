// Package cli provides shared client, output, flag, and service-registration helpers.
package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewClient constructs the service client for an operation.
func NewClient(cmd *cobra.Command, serviceName string) (*client.GreennodeClient, error) {
	apiClient, _, err := BuildClient(cmd, ClientOptions{
		ResolveEndpoint: func(cfg *config.Config) (string, error) {
			return cfg.GetEndpoint(serviceName)
		},
		ApplyRegion: true,
	})
	return apiClient, err
}

// NewClientWithEndpoint uses a global service endpoint; explicit overrides retain endpoint safety checks.
func NewClientWithEndpoint(cmd *cobra.Command, endpoint string) (*client.GreennodeClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("service endpoint must not be empty")
	}
	apiClient, _, err := BuildClient(cmd, ClientOptions{
		ResolveEndpoint: func(*config.Config) (string, error) {
			return endpoint, nil
		},
	})
	return apiClient, err
}

// ClientOptions customizes endpoint resolution, region selection, and post-build setup.
type ClientOptions struct {
	// ResolveEndpoint selects the configured endpoint; required unless overridden.
	ResolveEndpoint func(cfg *config.Config) (string, error)

	// ApplyRegion copies an explicit region flag before endpoint resolution; leave false for global APIs.
	ApplyRegion bool

	// AfterBuild validates or attaches service-specific state; an error aborts client construction.
	AfterBuild func(cmd *cobra.Command, cfg *config.Config, apiClient *client.GreennodeClient) error
}

// BuildClient validates and wires a shared client.
func BuildClient(cmd *cobra.Command, opts ClientOptions) (*client.GreennodeClient, *config.Config, error) {
	if opts.ResolveEndpoint == nil {
		return nil, nil, fmt.Errorf("internal error: ClientOptions.ResolveEndpoint must not be nil")
	}

	profile, err := flagString(cmd, "profile")
	if err != nil {
		return nil, nil, err
	}
	region, err := flagString(cmd, "region")
	if err != nil {
		return nil, nil, err
	}
	endpointURL, err := flagString(cmd, "endpoint-url")
	if err != nil {
		return nil, nil, err
	}
	noVerifySSL, err := flagBool(cmd, "no-verify-ssl")
	if err != nil {
		return nil, nil, err
	}
	debug, err := flagBool(cmd, "debug")
	if err != nil {
		return nil, nil, err
	}
	allowUntrusted, err := flagBool(cmd, "allow-untrusted-endpoint")
	if err != nil {
		return nil, nil, err
	}
	connectTimeoutSeconds, err := flagInt(cmd, "cli-connect-timeout")
	if err != nil {
		return nil, nil, err
	}
	readTimeoutSeconds, err := flagInt(cmd, "cli-read-timeout")
	if err != nil {
		return nil, nil, err
	}

	if err := CheckEndpoint(endpointURL, noVerifySSL, allowUntrusted); err != nil {
		return nil, nil, err
	}

	cfg, err := config.LoadConfig(profile)
	if err != nil {
		return nil, nil, err
	}

	if opts.ApplyRegion && region != "" {
		cfg.Region = region
	}

	var baseURL string
	if endpointURL != "" {
		baseURL = endpointURL
	} else {
		baseURL, err = opts.ResolveEndpoint(cfg)
		if err != nil {
			return nil, nil, err
		}
	}

	if endpointURL == "" {
		if err := CheckEndpoint(baseURL, noVerifySSL, allowUntrusted); err != nil {
			return nil, nil, err
		}
	}
	if connectTimeoutSeconds < 0 || readTimeoutSeconds < 0 {
		return nil, nil, fmt.Errorf("HTTP timeouts must not be negative")
	}

	// Warn once when TLS verification is disabled.
	if noVerifySSL {
		fmt.Fprintln(os.Stderr, "Warning: SSL certificate verification is disabled. This is not recommended for production use.")
	}

	connect := time.Duration(connectTimeoutSeconds) * time.Second
	read := time.Duration(readTimeoutSeconds) * time.Second
	if read == 0 {
		read = 30 * time.Second
	}

	tp, err := NewTokenProvider(cfg)
	if err != nil {
		return nil, nil, err
	}
	if configurable, ok := tp.(interface{ SetHTTPTimeout(time.Duration) }); ok {
		configurable.SetHTTPTimeout(read)
	}
	apiClient := client.NewGreennodeClient(baseURL, tp, connect, read, !noVerifySSL, debug)

	// Share command cancellation with API and IAM requests.
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	apiClient.SetBaseContext(ctx)
	if contextual, ok := tp.(interface{ SetBaseContext(context.Context) }); ok {
		contextual.SetBaseContext(ctx)
	}

	if opts.AfterBuild != nil {
		if err := opts.AfterBuild(cmd, cfg, apiClient); err != nil {
			return nil, nil, err
		}
	}

	return apiClient, cfg, nil
}

// Global flag readers fail on missing registrations or invalid types.
func flagString(cmd *cobra.Command, name string) (string, error) {
	v, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("internal error: read --%s flag: %w", name, err)
	}
	return v, nil
}

func flagBool(cmd *cobra.Command, name string) (bool, error) {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false, fmt.Errorf("internal error: read --%s flag: %w", name, err)
	}
	return v, nil
}

func flagInt(cmd *cobra.Command, name string) (int, error) {
	v, err := cmd.Flags().GetInt(name)
	if err != nil {
		return 0, fmt.Errorf("internal error: read --%s flag: %w", name, err)
	}
	return v, nil
}
