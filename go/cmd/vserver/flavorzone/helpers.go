package flavorzone

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var createClient = vserverclient.BuildClient

func runGet(cmd *cobra.Command, pathTemplate, description string, params map[string]string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	path := strings.Replace(pathTemplate, "%s", projectID, 1)
	result, err := apiClient.Get(path, params)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", description, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func zoneParams(cmd *cobra.Command) map[string]string {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	params := map[string]string{}
	if zoneID != "" {
		params["zoneId"] = zoneID
	}
	return params
}
