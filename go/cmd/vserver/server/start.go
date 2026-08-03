package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a stopped vServer instance",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().String("server-id", "", "Server ID (required)")
	startCmd.MarkFlagRequired("server-id")
}

func runStart(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
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

	result, err := apiClient.Put(fmt.Sprintf("/v2/%s/servers/%s/start", projectID, serverID), nil)
	if err != nil {
		return fmt.Errorf("failed to start server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, transformServerResult(result))
}
