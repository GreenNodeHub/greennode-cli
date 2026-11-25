package lbpackage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var tableColumns = []string{
	"uuid", "name", "type", "displayLbType", "connectionNumber", "dataTransfer", "mode",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List load balancer packages",
	RunE:  runList,
}

func init() {
	f := listCmd.Flags()
	f.String("zone-id", "", "Zone to list active packages for (active packages only)")
	f.Bool("all", false, "List every package, including those not currently offered")
}

func runList(cmd *cobra.Command, args []string) error {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	all, _ := cmd.Flags().GetBool("all")

	if err := checkZoneWithAll(all, cmd.Flags().Changed("zone-id")); err != nil {
		return err
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	path := vlbclient.Packages(projectID)
	var params map[string]string
	if all {
		path = vlbclient.AllPackages(projectID)
	} else if zoneID != "" {
		params = map[string]string{"zoneId": zoneID}
	}

	result, err := apiClient.Get(path, params)
	if err != nil {
		return fmt.Errorf("failed to list load balancer packages: %w", err)
	}

	return vlbclient.OutputWithColumns(cmd, cfg, result, tableColumns)
}

// checkZoneWithAll refuses --zone-id together with --all: the all_packages
// endpoint takes no zone filter, so the zone would be dropped silently and every
// zone's packages returned. It checks the VALUE of --all, not just whether it
// was set (as MarkFlagsMutuallyExclusive would), so a script passing
// `--all=$ALL --zone-id Z` still works when ALL=false.
func checkZoneWithAll(all, zoneIDSet bool) error {
	if all && zoneIDSet {
		return fmt.Errorf("--zone-id cannot be used with --all: the full package list is not filtered by zone")
	}
	return nil
}
