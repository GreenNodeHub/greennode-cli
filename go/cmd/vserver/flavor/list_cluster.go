package flavor

import (
	"fmt"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listClusterCmd = &cobra.Command{Use: "list-cluster", Short: "List flavors supported for a cluster role", RunE: runListCluster}

func init() {
	f := listClusterCmd.Flags()
	f.String("family", "", "Instance family key (required)")
	f.String("platform", "", "CPU platform key (required)")
	f.Bool("master", false, "List master-node flavors instead of worker-node flavors")
	f.String("zone-id", "", "Availability zone ID")
	for _, name := range []string{"family", "platform"} {
		if err := listClusterCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runListCluster(cmd *cobra.Command, args []string) error {
	family, _ := cmd.Flags().GetString("family")
	platform, _ := cmd.Flags().GetString("platform")
	master, _ := cmd.Flags().GetBool("master")
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
	path := fmt.Sprintf("/v1/%s/flavors/families/%s/platforms/%s/clusters/master/%t", projectID, url.PathEscape(family), url.PathEscape(platform), master)
	result, err := apiClient.Get(path, params)
	if err != nil {
		return fmt.Errorf("failed to list cluster flavors: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
