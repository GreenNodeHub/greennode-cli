package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getUpdateCmd = &cobra.Command{
	Use:   "get-update",
	Short: "Get the status of an update (add-on, node group or cluster)",
	Long: `Get the status of an update.

The scope must match the one the update was recorded with: pass --addon-name
for an update returned by update-addon, --nodegroup-id for a node group update.`,
	RunE: runGetUpdate,
}

func init() {
	f := getUpdateCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("update-id", "", "Update ID (required)")
	f.String("nodegroup-id", "", "Node group the update belongs to")
	f.String("addon-name", "", "Add-on the update belongs to")

	getUpdateCmd.MarkFlagRequired("cluster-id")
	getUpdateCmd.MarkFlagRequired("update-id")
	getUpdateCmd.MarkFlagsMutuallyExclusive("nodegroup-id", "addon-name")
	_ = getUpdateCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchClusterAddonNames))
	_ = getUpdateCmd.RegisterFlagCompletionFunc("update-id", cli.FlagFromAPI(fetchUpdateIDs))
}

func runGetUpdate(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	updateID, _ := cmd.Flags().GetString("update-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	addonName, _ := cmd.Flags().GetString("addon-name")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(updateID, "update-id"); err != nil {
		return err
	}
	if err := validateUpdateScope(nodegroupID, addonName); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(
		fmt.Sprintf("/v1/clusters/%s/updates/%s", clusterID, updateID), updateScopeParams(nodegroupID, addonName),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
