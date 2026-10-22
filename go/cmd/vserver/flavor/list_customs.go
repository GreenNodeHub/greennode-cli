package flavor

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listCustomsCmd = &cobra.Command{Use: "list-customs", Short: "List custom flavors", RunE: runListCustoms}

func init() {
	listCustomsCmd.Flags().String("zone-id", "", "Availability zone ID")
}

func runListCustoms(cmd *cobra.Command, args []string) error {
	return runFlavorZoneQuery(cmd, "/v1/%s/flavors/customs", "custom flavors")
}

var listCustomClusterCmd = &cobra.Command{Use: "list-custom-clusters", Short: "List custom flavors supported for clusters", RunE: runListCustomClusters}

func init() {
	listCustomClusterCmd.Flags().String("zone-id", "", "Availability zone ID")
}

func runListCustomClusters(cmd *cobra.Command, args []string) error {
	return runFlavorZoneQuery(cmd, "/v1/%s/flavors/customs/clusters", "custom cluster flavors")
}

func runFlavorZoneQuery(cmd *cobra.Command, pathTemplate, description string) error {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	params := map[string]string{}
	if zoneID != "" {
		params["zoneId"] = zoneID
	}
	result, err := apiClient.Get(fmt.Sprintf(pathTemplate, projectID), params)
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", description, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
