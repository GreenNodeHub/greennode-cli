package networkinterface

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an elastic network interface",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("network-interface-id", "", "Network interface ID (required)")
	if err := getCmd.MarkFlagRequired("network-interface-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "network-interface-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
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

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/network-interfaces-elastic/%s", projectID, interfaceID), nil)
	if err != nil {
		return fmt.Errorf("failed to get network interface %s: %w", interfaceID, err)
	}
	return outputInterfaceList(cmd, cfg, result)
}
