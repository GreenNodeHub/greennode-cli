package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a vServer instance to a different flavor",
	RunE:  runResize,
}

func init() {
	f := resizeCmd.Flags()
	f.String("server-id", "", "Server ID (required)")
	f.String("flavor-id", "", "New flavor ID — run 'vserver flavor list' to see options (required)")

	if err := resizeCmd.MarkFlagRequired("server-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "server-id", err))
	}
}

func runResize(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	flavorID, _ := cmd.Flags().GetString("flavor-id")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	if flavorID == "" {
		return suggestFlavors()
	}

	result, err := apiClient.Put(
		fmt.Sprintf("/v2/%s/servers/%s/resize", projectID, serverID),
		map[string]interface{}{"flavorId": flavorID},
	)
	if err != nil {
		return fmt.Errorf("failed to resize server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, transformServerResult(result))
}
