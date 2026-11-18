package vks

import (
	"fmt"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var listAddonVersionsCmd = &cobra.Command{
	Use:   "list-addon-versions",
	Short: "List the add-ons of the catalog and their versions",
	Long: `List the add-ons of the catalog and their versions.

Without --k8s-version the whole catalog is returned. DEPRECATED versions are
returned only with --include-deprecated. For each Kubernetes minor, the
version marked defaultVersion is the one create-addon installs when
--addon-version is omitted.`,
	RunE: runListAddonVersions,
}

func init() {
	f := listAddonVersionsCmd.Flags()
	f.String("addon-name", "", "Only this add-on")
	f.String("k8s-version", "", "Only versions compatible with this Kubernetes minor (1.32) or version (1.32.9)")
	f.String("owners", "", "Owners, comma-separated (vks, community, marketplace)")
	f.String("publishers", "", "Publishers, comma-separated")
	f.String("types", "", "Types, comma-separated (networking, dns, storage, loadbalancing, observability, security, gitops, other)")
	f.Bool("include-deprecated", false, "Include DEPRECATED versions")

	_ = listAddonVersionsCmd.RegisterFlagCompletionFunc("addon-name", cli.FlagFromAPI(fetchCatalogAddonNames))
	_ = listAddonVersionsCmd.RegisterFlagCompletionFunc("owners", cli.FlagValues("vks", "community", "marketplace"))
	_ = listAddonVersionsCmd.RegisterFlagCompletionFunc("types", cli.FlagValues(
		"networking", "dns", "storage", "loadbalancing", "observability", "security", "gitops", "other"))
}

// buildAddonVersionsQuery builds the /v1/addon-versions query from the set flags.
func buildAddonVersionsQuery(addonName, k8sVersion, owners, publishers, types string, includeDeprecated bool) map[string]string {
	params := map[string]string{}
	if addonName != "" {
		params["addonName"] = addonName
	}
	if k8sVersion != "" {
		params["kubernetesVersion"] = strings.TrimPrefix(k8sVersion, "v")
	}
	for key, v := range map[string]string{"owners": owners, "publishers": publishers, "types": types} {
		if list := parseCommaSeparated(v); len(list) > 0 {
			params[key] = strings.Join(list, ",")
		}
	}
	if includeDeprecated {
		params["includeDeprecated"] = "true"
	}
	return params
}

func runListAddonVersions(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	addonName, _ := f.GetString("addon-name")
	k8sVersion, _ := f.GetString("k8s-version")
	owners, _ := f.GetString("owners")
	publishers, _ := f.GetString("publishers")
	types, _ := f.GetString("types")
	includeDeprecated, _ := f.GetBool("include-deprecated")

	if addonName != "" {
		if err := validateAddonName(addonName); err != nil {
			return err
		}
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	params := buildAddonVersionsQuery(addonName, k8sVersion, owners, publishers, types, includeDeprecated)
	result, err := apiClient.Get("/v1/addon-versions", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
