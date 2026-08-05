package networkinterface

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Rename a network interface",
	Long:  "Rename an elastic network interface. Only the name can be changed.",
	RunE:  runEdit,
}

func init() {
	f := updateCmd.Flags()
	f.String("network-interface-id", "", "Network interface ID (required)")
	f.String("name", "", "New name (required)")

	for _, name := range []string{"network-interface-id", "name"} {
		if err := updateCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runEdit(cmd *cobra.Command, args []string) error {
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	name, _ := cmd.Flags().GetString("name")

	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("flag --name is required")
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Put(
		fmt.Sprintf("/v2/%s/network-interfaces-elastic/%s/rename", projectID, interfaceID),
		map[string]interface{}{"name": name},
	)
	if err != nil {
		return fmt.Errorf("failed to rename network interface %s: %w", interfaceID, err)
	}

	return outputResult(cmd, cfg, result)
}
