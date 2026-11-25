package certificate

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a certificate",
	RunE:  runGet,
}

func init() {
	addCertFlag(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	certID, err := certIDOf(cmd)
	if err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(vlbclient.Certificate(projectID, certID), nil)
	if err != nil {
		return fmt.Errorf("failed to get certificate %s: %w", certID, err)
	}

	return outputResult(cmd, cfg, result)
}
