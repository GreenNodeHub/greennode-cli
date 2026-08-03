package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var generateKubeconfigCmd = &cobra.Command{
	Use:   "generate-kubeconfig",
	Short: "Request generation of a kubeconfig for a VKS cluster",
	Long: "Requests the VKS API to generate (or renew) a kubeconfig for the cluster. " +
		"This is asynchronous; once the kubeconfig becomes ACTIVE, run 'grn vks update-kubeconfig'.",
	RunE: runGenerateKubeconfig,
}

func init() {
	f := generateKubeconfigCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Int("expiration-days", 30, "Number of days until the kubeconfig expires")
	f.Bool("dry-run", false, "Preview without requesting generation")

	generateKubeconfigCmd.MarkFlagRequired("cluster-id")
}

func runGenerateKubeconfig(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	expirationDays, _ := cmd.Flags().GetInt("expiration-days")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	body := map[string]interface{}{"expirationDays": expirationDays}

	if dryRun {
		cli.PrintDryRun("generate", fmt.Sprintf("kubeconfig for cluster %s", clusterID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	_, err = apiClient.Post(
		fmt.Sprintf("/v1/clusters/%s/kubeconfig", clusterID), body,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Kubeconfig generation requested for cluster %s (expires in %d days).\n", clusterID, expirationDays)
	fmt.Println("Generation is asynchronous. Once it is ACTIVE, run:")
	fmt.Printf("  grn vks update-kubeconfig --cluster-id %s\n", clusterID)
	return nil
}
