package vks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/kubeconfig"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateKubeconfigCmd = &cobra.Command{
	Use:   "update-kubeconfig",
	Short: "Fetch the cluster kubeconfig and merge it into your kubeconfig file",
	RunE:  runUpdateKubeconfig,
}

func init() {
	f := updateKubeconfigCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	f.String("alias", "", "Context name to use (default: vks_<cluster-id>)")
	f.Bool("no-set-context", false, "Do not set the merged context as current-context")
	f.Bool("dry-run", false, "Print what would be written without modifying the file")

	updateKubeconfigCmd.MarkFlagRequired("cluster-id")
}

// resolveKubeconfigPath picks the target path: explicit flag, then first entry
// of $KUBECONFIG, then ~/.kube/config.
func resolveKubeconfigPath(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return strings.Split(env, string(os.PathListSeparator))[0], nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

func runUpdateKubeconfig(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig")
	alias, _ := cmd.Flags().GetString("alias")
	noSetContext, _ := cmd.Flags().GetBool("no-set-context")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	contextName := alias
	if contextName == "" {
		contextName = "vks_" + clusterID
	}
	targetPath, err := resolveKubeconfigPath(kubeconfigPath)
	if err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s/kubeconfig", clusterID), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected kubeconfig response format")
	}

	status, _ := resMap["status"].(string)
	switch status {
	case "NONE", "":
		return fmt.Errorf("no kubeconfig exists for cluster %s. Run 'grn vks generate-kubeconfig --cluster-id %s' first", clusterID, clusterID)
	case "CREATING":
		return fmt.Errorf("kubeconfig for cluster %s is still being generated; try again shortly", clusterID)
	case "ERROR":
		return fmt.Errorf("kubeconfig for cluster %s is in ERROR state; re-run 'grn vks generate-kubeconfig'", clusterID)
	}

	if warn, _ := resMap["renewalWarning"].(bool); warn {
		fmt.Fprintf(os.Stderr, "Warning: this kubeconfig is nearing expiration. Run 'grn vks generate-kubeconfig --cluster-id %s' to renew.\n", clusterID)
	}

	rawYAML, _ := resMap["kubeConfig"].(string)
	if rawYAML == "" {
		return fmt.Errorf("kubeconfig response did not contain kubeconfig data")
	}

	if dryRun {
		fmt.Printf("=== DRY RUN ===\n")
		fmt.Printf("Would merge context %q into %s\n", contextName, targetPath)
		if !noSetContext {
			fmt.Printf("Would set current-context to %q\n", contextName)
		}
		return nil
	}

	res, err := kubeconfig.Merge(targetPath, rawYAML, contextName, !noSetContext)
	if err != nil {
		return err
	}

	fmt.Printf("Updated context %q in %s\n", res.ContextName, res.Path)
	return nil
}
