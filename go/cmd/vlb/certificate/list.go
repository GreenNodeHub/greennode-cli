package certificate

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List certificates",
	Long: "List the certificates in your project. The API returns them all in one " +
		"unpaginated response; use --query to narrow the output.",
	RunE: runList,
}

func init() {
	// No --page/--page-size: the spec declares them (defaults 1/10) but the
	// live API ignores them and returns every certificate, reporting
	// page=0/pageSize=0/totalPage=0 in the envelope. See the note on
	// loadbalancer.buildListQuery.
	listCmd.Flags().String("name", "", "Filter by certificate name (substring match)")
}

func runList(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	var params map[string]string
	if name != "" {
		params = map[string]string{"name": name}
	}

	result, err := apiClient.Get(vlbclient.Certificates(projectID), params)
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
