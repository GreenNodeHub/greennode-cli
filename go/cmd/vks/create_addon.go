package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var createAddonCmd = &cobra.Command{
	Use:   "create-addon",
	Short: "Install an add-on on a cluster",
	Long: `Install an add-on on a cluster.

Accepted while the cluster is WAITING_*, CREATING or ACTIVE, so add-ons can be
requested right after create-cluster. The add-on is returned with status
CREATING; use "grn vks wait addon-active" to wait until it is ACTIVE.

Find add-on names and versions with list-addon-versions, and the accepted
--configuration-values with get-addon-configuration-schema.`,
	RunE: runCreateAddon,
}

func init() {
	f := createAddonCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("addon-name", "", "Add-on name, e.g. block-storage-csi (required)")
	addAddonRequestFlags(createAddonCmd,
		"Add-on version (default: the default version for the cluster's Kubernetes minor)",
		"Configuration values as a JSON or YAML string, or file://<path>")
	f.String("namespace", "", "Namespace to install into (default: the version's default namespace)")
	f.Bool("dry-run", false, "Preview the request without executing")

	createAddonCmd.MarkFlagRequired("cluster-id")
	createAddonCmd.MarkFlagRequired("addon-name")
	_ = createAddonCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchCatalogAddonNames))
}

// buildCreateAddonBody builds the CreateAddon request body.
func buildCreateAddonBody(addonName, namespace string, a addonFlags, changed map[string]bool) map[string]interface{} {
	body := buildAddonBody(a, changed)
	body["addonName"] = addonName
	if namespace != "" {
		body["namespaceConfig"] = map[string]interface{}{"namespace": namespace}
	}
	return body
}

func runCreateAddon(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	addonName, _ := cmd.Flags().GetString("addon-name")
	namespace, _ := cmd.Flags().GetString("namespace")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validateAddonName(addonName); err != nil {
		return err
	}
	if namespace != "" {
		if err := validateNamespace(namespace); err != nil {
			return err
		}
	}
	a, changed, err := readAddonFlags(cmd)
	if err != nil {
		return err
	}

	body := buildCreateAddonBody(addonName, namespace, a, changed)

	if dryRun {
		cli.PrintDryRun("create", fmt.Sprintf("add-on %s on cluster %s", addonName, clusterID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(fmt.Sprintf("/v1/clusters/%s/addons", clusterID), body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
