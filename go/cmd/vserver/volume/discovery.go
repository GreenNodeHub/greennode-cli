package volume

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listByServerCmd = &cobra.Command{Use: "list-by-server", Short: "Get volumes attached to a server", RunE: runListByServer}
var getBootCmd = &cobra.Command{Use: "get-boot", Short: "Get a server's boot volume", RunE: runGetBoot}
var historyCmd = &cobra.Command{Use: "history", Short: "Get volume history", RunE: runHistory}
var mappingCmd = &cobra.Command{Use: "mapping", Short: "Get volume mapping details", RunE: runMapping}

func init() {
	for _, cmd := range []*cobra.Command{listByServerCmd, getBootCmd} {
		cmd.Flags().String("server-id", "", "Server ID (required)")
		if err := cmd.MarkFlagRequired("server-id"); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "server-id", err))
		}
	}
	for _, cmd := range []*cobra.Command{historyCmd, mappingCmd} {
		cmd.Flags().String("volume-id", "", "Volume ID (required)")
		if err := cmd.MarkFlagRequired("volume-id"); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "volume-id", err))
		}
	}
}

func runListByServer(cmd *cobra.Command, args []string) error {
	return runServerVolumeGet(cmd, "")
}

func runGetBoot(cmd *cobra.Command, args []string) error {
	return runServerVolumeGet(cmd, "/boot")
}

func runServerVolumeGet(cmd *cobra.Command, suffix string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
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
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/volumes/servers/%s%s", projectID, serverID, suffix), nil)
	if err != nil {
		return fmt.Errorf("failed to get volumes for server %s: %w", serverID, err)
	}
	return outputResult(cmd, cfg, result)
}

func runHistory(cmd *cobra.Command, args []string) error {
	return runVolumeResourceGet(cmd, "history")
}

func runMapping(cmd *cobra.Command, args []string) error {
	return runVolumeResourceGet(cmd, "mapping")
}

func runVolumeResourceGet(cmd *cobra.Command, suffix string) error {
	volumeID, _ := cmd.Flags().GetString("volume-id")
	if err := validator.ValidateID(volumeID, "volume-id"); err != nil {
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
	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/volumes/%s/%s", projectID, volumeID, suffix), nil)
	if err != nil {
		return fmt.Errorf("failed to get volume %s %s: %w", volumeID, suffix, err)
	}
	return outputResult(cmd, cfg, result)
}
