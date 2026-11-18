package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateAddonCmd = &cobra.Command{
	Use:   "update-addon",
	Short: "Update the version or configuration of an add-on",
	Long: `Update the version or configuration of an add-on.

Only the flags you set are sent: an omitted --addon-version or
--configuration-values keeps the current one, and --configuration-values ""
clears the values. The update is asynchronous: the command returns an update
with status InProgress; use "grn vks wait update-successful" or get-update.`,
	RunE: runUpdateAddon,
}

func init() {
	f := updateAddonCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("addon-name", "", "Add-on name (required)")
	addAddonRequestFlags(updateAddonCmd,
		"Target add-on version (default: keep the current version)",
		`Configuration values as a JSON or YAML string, or file://<path> ("" clears them)`)
	f.Bool("dry-run", false, "Preview the request without executing")

	updateAddonCmd.MarkFlagRequired("cluster-id")
	updateAddonCmd.MarkFlagRequired("addon-name")
	_ = updateAddonCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchClusterAddonNames))
}

func runUpdateAddon(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	addonName, _ := cmd.Flags().GetString("addon-name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validateAddonName(addonName); err != nil {
		return err
	}
	a, changed, err := readAddonFlags(cmd)
	if err != nil {
		return err
	}
	if !changed["addon-version"] && !changed["configuration-values"] {
		return fmt.Errorf("nothing to update: set --addon-version and/or --configuration-values")
	}

	body := buildAddonBody(a, changed)

	if dryRun {
		cli.PrintDryRun("update", fmt.Sprintf("add-on %s on cluster %s", addonName, clusterID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(fmt.Sprintf("/v1/clusters/%s/addons/%s/update", clusterID, addonName), body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
