package iamclient

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/spf13/cobra"
)

const (
	AccountsEndpoint = "https://iamapis.vngcloud.vn/accounts-api"
	PoliciesEndpoint = "https://iamapis.vngcloud.vn/policies-api"
)

func BuildAccountsClient(cmd *cobra.Command) (*client.GreennodeClient, error) {
	return cli.NewClientWithEndpoint(cmd, AccountsEndpoint)
}

func BuildPoliciesClient(cmd *cobra.Command) (*client.GreennodeClient, error) {
	return cli.NewClientWithEndpoint(cmd, PoliciesEndpoint)
}
