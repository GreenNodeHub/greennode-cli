package vdbclient

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/spf13/cobra"
)

const Endpoint = "https://vdb-gateway.vngcloud.vn"

func BuildClient(cmd *cobra.Command, requirePortalUserID bool) (*client.GreennodeClient, error) {
	apiClient, _, err := cli.BuildClient(cmd, cli.ClientOptions{
		ResolveEndpoint: func(*config.Config) (string, error) {
			return Endpoint, nil
		},
		AfterBuild: func(_ *cobra.Command, cfg *config.Config, apiClient *client.GreennodeClient) error {
			if !requirePortalUserID {
				return nil
			}
			if err := ValidatePortalUserID(cfg.PortalUserID); err != nil {
				return err
			}
			apiClient.SetHeader("portal-user-id", cfg.PortalUserID)
			return nil
		},
	})
	return apiClient, err
}

func ValidatePortalUserID(value string) error {
	if value == "" {
		return fmt.Errorf("portal_user_id is required by the documented vDB API; run 'grn configure set portal_user_id <id>' or set GRN_PORTAL_USER_ID")
	}
	if _, err := config.ValidatePositiveInt32(value); err != nil {
		return fmt.Errorf("portal_user_id must be a positive 32-bit integer")
	}
	return nil
}
