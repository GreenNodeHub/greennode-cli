package agentbase

import (
	"fmt"
	"os"

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
	"github.com/vngcloud/greennode-cli/internal/agentbase/config"
)

// mustLoadConfig loads config, applying the --env flag override when set, and
// exits on any error. Commands use this (not config.Load() directly) so the
// persistent --env flag is respected.
func mustLoadConfig() *config.Config {
	cfg, err := config.LoadWithEnv(envOverride)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return cfg
}

// mustLoadConfigWithCreds loads config and ensures credentials are present.
func mustLoadConfigWithCreds() *config.Config {
	cfg := mustLoadConfig()
	if err := cfg.RequireCredentials(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return cfg
}

// newAuthProvider builds an auth provider from the loaded config.
func newAuthProvider(cfg *config.Config) *auth.Provider {
	return auth.NewProvider(cfg.ClientID, cfg.ClientSecret, cfg.Endpoints.OAuth2Token)
}
