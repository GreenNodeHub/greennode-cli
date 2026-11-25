package loadbalancer

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List load balancers",
	Long: "List the load balancers in your project. The API returns them all in " +
		"one unpaginated response; use --query to narrow the output.",
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.String("name", "", "Filter by load balancer name (substring match)")
	f.String("zone-id", "", "Filter by zone ID (exact match)")
}

// buildListQuery builds the list query params.
//
// Only `name` and `zoneId` are here, because they are the only filters the
// backend actually honours — verified against the live HCM-3 API. The spec also
// declares ids/names/subnetIds/vpcIds/status filters and a pageRequest object,
// but every encoding of those (flat, `filter.`-prefixed, bracketed, repeated)
// returns exactly the same result as a parameter that does not exist, and the
// response envelope always reports page=0/pageSize=0/totalPage=0: this
// deployment returns the full list, unpaginated. Flags for them would promise
// filtering the service does not do, so they are deliberately not offered —
// use --query to narrow the output client-side. Re-add them (flat form, which
// is what `name` proves correct) once the backend implements them.
func buildListQuery(name, zoneID string) map[string]string {
	params := map[string]string{}
	if name != "" {
		params["name"] = name
	}
	if zoneID != "" {
		params["zoneId"] = zoneID
	}
	return params
}

func runList(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	zoneID, _ := cmd.Flags().GetString("zone-id")

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	params := buildListQuery(name, zoneID)
	result, err := apiClient.Get(vlbclient.LoadBalancers(projectID), params)
	if err != nil {
		return fmt.Errorf("failed to list load balancers: %w", err)
	}

	return outputResult(cmd, cfg, result)
}
