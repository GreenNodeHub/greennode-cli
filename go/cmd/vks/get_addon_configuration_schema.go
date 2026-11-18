package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var getAddonConfigurationSchemaCmd = &cobra.Command{
	Use:   "get-addon-configuration-schema",
	Short: "Get the configuration schema of an add-on version",
	Long: `Get the JSON Schema (draft-07) that --configuration-values of
create-addon / update-addon must satisfy for an add-on version.

The schema is returned as a string in configurationSchema; extract it with
--query configurationSchema --output text.`,
	RunE: runGetAddonConfigurationSchema,
}

func init() {
	f := getAddonConfigurationSchemaCmd.Flags()
	f.String("addon-name", "", "Add-on name (required)")
	f.String("addon-version", "", "Add-on version, e.g. 1.2.0-vks.1 (required)")

	getAddonConfigurationSchemaCmd.MarkFlagRequired("addon-name")
	getAddonConfigurationSchemaCmd.MarkFlagRequired("addon-version")
	_ = getAddonConfigurationSchemaCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchCatalogAddonNames))
	_ = getAddonConfigurationSchemaCmd.RegisterFlagCompletionFunc("addon-version", cli.FlagFromAPI(fetchAddonVersions))
}

func runGetAddonConfigurationSchema(cmd *cobra.Command, args []string) error {
	addonName, _ := cmd.Flags().GetString("addon-name")
	addonVersion, _ := cmd.Flags().GetString("addon-version")

	if err := validateAddonName(addonName); err != nil {
		return err
	}
	if err := validateAddonVersion(addonVersion); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(
		fmt.Sprintf("/v1/addons/%s/versions/%s/configuration-schema", addonName, addonVersion), nil,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
