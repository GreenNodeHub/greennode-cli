package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var listAddonsCmd = &cobra.Command{
	Use:   "list-addons",
	Short: "List the add-ons installed on a cluster",
	RunE:  runListAddons,
}

func init() {
	f := listAddonsCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.Bool("detail", false, "Also return the full add-ons (status, version, health, ...)")

	listAddonsCmd.MarkFlagRequired("cluster-id")
}

func runListAddons(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	detail, _ := cmd.Flags().GetBool("detail")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	var params map[string]string
	if detail {
		params = map[string]string{"detail": "true"}
	}
	result, err := apiClient.Get(fmt.Sprintf("/v1/clusters/%s/addons", clusterID), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cli.Output(cmd, result)
}
