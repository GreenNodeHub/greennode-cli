package certificate

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

// No expiry column. expiredAt is null on almost every certificate the service
// returns (17 of 18 in the project this was verified against) and contradicted
// its own notAfter on the one that was set, so it rendered blank and hid
// certificates that had already expired. notAfter carries the real value but is
// epoch milliseconds, and the table layer does no per-column value formatting,
// so it rendered as a raw integer — accurate but unreadable. Both are worse than
// leaving expiry out of the list: `certificate get` shows notAfter in full, and
// `certificate delete` formats it and marks (EXPIRED) before you confirm.
var tableColumns = []string{
	"uuid", "name", "certificateType", "domainName", "inUse",
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
