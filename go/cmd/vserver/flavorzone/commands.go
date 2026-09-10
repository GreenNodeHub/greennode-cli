package flavorzone

import (
	"fmt"
	"net/url"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listCodesCmd = &cobra.Command{
	Use: "list-codes", Short: "List flavor platform codes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, "/v1/%s/flavor_zones/codes", "list flavor platform codes", nil)
	},
}

var listCustomsCmd = &cobra.Command{
	Use: "list-customs", Short: "List custom flavor zones",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, "/v1/%s/flavor_zones/customs", "list custom flavor zones", zoneParams(cmd))
	},
}

var listCustomClustersCmd = &cobra.Command{
	Use: "list-custom-clusters", Short: "List custom flavor zones supported for a cluster role",
	RunE: func(cmd *cobra.Command, args []string) error {
		master, _ := cmd.Flags().GetBool("master")
		path := fmt.Sprintf("/v1/%%s/flavor_zones/customs/clusters/master/%t", master)
		return runGet(cmd, path, "list custom cluster flavor zones", zoneParams(cmd))
	},
}

var listFamiliesCmd = &cobra.Command{
	Use: "list-families", Short: "List flavor families",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, "/v1/%s/flavor_zones/families", "list flavor families", zoneParams(cmd))
	},
}

var listFamilyClustersCmd = &cobra.Command{
	Use: "list-family-clusters", Short: "List flavor families supported for clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, "/v1/%s/flavor_zones/families/clusters", "list cluster flavor families", nil)
	},
}

var listProductsCmd = &cobra.Command{
	Use: "list-products", Short: "List product flavor zones",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(cmd, "/v1/%s/flavor_zones/product", "list product flavor zones", nil)
	},
}

var listProductCmd = &cobra.Command{Use: "list-product", Short: "List flavor zones for a product", RunE: runListProduct}

var getCmd = &cobra.Command{Use: "get", Short: "Get a flavor zone", RunE: runGetFlavorZone}

func init() {
	listCustomsCmd.Flags().String("zone-id", "", "Availability zone ID")
	listCustomClustersCmd.Flags().String("zone-id", "", "Availability zone ID")
	listCustomClustersCmd.Flags().Bool("master", false, "List master-node zones instead of worker-node zones")
	listFamiliesCmd.Flags().String("zone-id", "", "Availability zone ID")

	listProductCmd.Flags().String("product", "", "Product name (required)")
	if err := listProductCmd.MarkFlagRequired("product"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "product", err))
	}
	getCmd.Flags().String("flavor-zone-id", "", "Flavor zone ID (required)")
	if err := getCmd.MarkFlagRequired("flavor-zone-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "flavor-zone-id", err))
	}
}

func runListProduct(cmd *cobra.Command, args []string) error {
	product, _ := cmd.Flags().GetString("product")
	return runGet(cmd, "/v1/%s/flavor_zones/product/"+url.PathEscape(product), "list product flavor zones", nil)
}

func runGetFlavorZone(cmd *cobra.Command, args []string) error {
	zoneID, _ := cmd.Flags().GetString("flavor-zone-id")
	if err := validator.ValidateID(zoneID, "flavor-zone-id"); err != nil {
		return err
	}
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf("/v1/%s/flavor_zones/%s", projectID, zoneID), nil)
	if err != nil {
		return fmt.Errorf("failed to get flavor zone %s: %w", zoneID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
