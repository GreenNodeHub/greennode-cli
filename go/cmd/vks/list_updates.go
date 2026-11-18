package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listUpdatesCmd = &cobra.Command{
	Use:   "list-updates",
	Short: "List the update IDs of a cluster, a node group or an add-on",
	Long: `List the update IDs of a cluster, newest first.

Without a filter, the updates of the cluster itself are returned; with
--nodegroup-id those of that node group; with --addon-name those of that
add-on (at most one filter). All pages are fetched unless --max-results or
--next-token is set.`,
	RunE: runListUpdates,
}

func init() {
	f := listUpdatesCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Only updates of this node group")
	f.String("addon-name", "", "Only updates of this add-on")
	f.Int("max-results", 0, "Page size (1-100); disables auto-pagination")
	f.String("next-token", "", "Token of the previous page; disables auto-pagination")

	listUpdatesCmd.MarkFlagRequired("cluster-id")
	listUpdatesCmd.MarkFlagsMutuallyExclusive("nodegroup-id", "addon-name")
	_ = listUpdatesCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchClusterAddonNames))
}

// updateScopeParams returns the nodeGroupId / addonName scope query.
func updateScopeParams(nodegroupID, addonName string) map[string]string {
	params := map[string]string{}
	if nodegroupID != "" {
		params["nodeGroupId"] = nodegroupID
	}
	if addonName != "" {
		params["addonName"] = addonName
	}
	return params
}

func validateUpdateScope(nodegroupID, addonName string) error {
	if nodegroupID != "" {
		if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
			return err
		}
	}
	if addonName != "" {
		return validateAddonName(addonName)
	}
	return nil
}

func runListUpdates(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	clusterID, _ := f.GetString("cluster-id")
	nodegroupID, _ := f.GetString("nodegroup-id")
	addonName, _ := f.GetString("addon-name")
	maxResults, _ := f.GetInt("max-results")
	nextToken, _ := f.GetString("next-token")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validateUpdateScope(nodegroupID, addonName); err != nil {
		return err
	}
	if f.Changed("max-results") && (maxResults < 1 || maxResults > 100) {
		return fmt.Errorf("--max-results must be between 1 and 100")
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/clusters/%s/updates", clusterID)
	params := updateScopeParams(nodegroupID, addonName)

	// One page when the caller drives pagination.
	if f.Changed("max-results") || nextToken != "" {
		if maxResults > 0 {
			params["maxResults"] = fmt.Sprintf("%d", maxResults)
		}
		if nextToken != "" {
			params["nextToken"] = nextToken
		}
		result, err := apiClient.Get(path, params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return cli.Output(cmd, result)
	}

	// Otherwise follow nextToken until the last page.
	all := []string{}
	for {
		result, err := apiClient.Get(path, params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		all = append(all, extractStrings(result, "updateIds")...)
		m, _ := result.(map[string]interface{})
		token, _ := m["nextToken"].(string)
		if token == "" {
			break
		}
		params["nextToken"] = token
	}
	return cli.Output(cmd, map[string]interface{}{"updateIds": all})
}
