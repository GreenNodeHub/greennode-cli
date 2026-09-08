package certificate

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var tableColumns = []string{
	"uuid", "name", "certificateType", "domainName", "issuer", "expiredAt", "inUse",
}

func resolve(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, string, error) {
	return vlbclient.Resolve(cmd)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.OutputWithColumns(cmd, cfg, data, tableColumns)
}

func outputRaw(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vlbclient.Output(cmd, cfg, data)
}

// addCertFlag declares --certificate-id, required by the per-certificate subcommands.
func addCertFlag(c *cobra.Command) {
	c.Flags().String("certificate-id", "", "Certificate ID (required)")
	if err := c.MarkFlagRequired("certificate-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "certificate-id", err))
	}
}

func certIDOf(cmd *cobra.Command) (string, error) {
	id, _ := cmd.Flags().GetString("certificate-id")
	if err := validator.ValidateID(id, "certificate-id"); err != nil {
		return "", err
	}
	return id, nil
}

// unwrapData returns the payload of a {"data": ...} envelope, or the response
// itself when it is already the bare object.
func unwrapData(response interface{}) map[string]interface{} {
	m, ok := response.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	if data, ok := m["data"].(map[string]interface{}); ok {
		return data
	}
	return m
}
