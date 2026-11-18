package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getAddonCmd = &cobra.Command{
	Use:   "get-addon",
	Short: "Get details of an add-on installed on a cluster",
	RunE:  runGetAddon,
}

func init() {
	f := getAddonCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("addon-name", "", "Add-on name (required)")

	getAddonCmd.MarkFlagRequired("cluster-id")
	getAddonCmd.MarkFlagRequired("addon-name")
	_ = getAddonCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchClusterAddonNames))
}

func runGetAddon(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	addonName, _ := cmd.Flags().GetString("addon-name")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validateAddonName(addonName); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s/addons/%s", clusterID, addonName), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
