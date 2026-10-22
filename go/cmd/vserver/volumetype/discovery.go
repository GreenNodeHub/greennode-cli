package volumetype

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listAllCmd = &cobra.Command{Use: "list-all", Short: "List all volume types", RunE: runListAll}
var listZonesCmd = &cobra.Command{Use: "list-zones", Short: "List volume type zones", RunE: runListZones}
var getDefaultCmd = &cobra.Command{Use: "get-default", Short: "Get the default volume type ID", RunE: runGetDefault}
var getCmd = &cobra.Command{Use: "get", Short: "Get a volume type", RunE: runGet}
var getZoneCmd = &cobra.Command{Use: "get-zone", Short: "Get a volume type zone", RunE: runGetZone}

func init() {
	listZonesCmd.Flags().String("zone-id", "", "Filter by availability zone ID")
	getDefaultCmd.Flags().String("zone-id", "", "Filter by availability zone ID")
	getCmd.Flags().String("volume-type-id", "", "Volume type ID (required)")
	if err := getCmd.MarkFlagRequired("volume-type-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "volume-type-id", err))
	}
	getZoneCmd.Flags().String("volume-type-zone-id", "", "Volume type zone ID (required)")
	if err := getZoneCmd.MarkFlagRequired("volume-type-zone-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "volume-type-zone-id", err))
	}
}

func runListAll(cmd *cobra.Command, args []string) error {
	return getDiscovery(cmd, "/v1/%s/volume_types", "list all volume types", nil)
}

func runListZones(cmd *cobra.Command, args []string) error {
	return getDiscovery(cmd, "/v1/%s/volume_type_zones", "list volume type zones", optionalZoneParams(cmd))
}

func runGetDefault(cmd *cobra.Command, args []string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}

	params := optionalZoneParams(cmd)
	if params == nil {
		zoneResult, err := apiClient.Get(fmt.Sprintf("/v1/%s/zones", projectID), nil)
		if err != nil {
			return fmt.Errorf("failed to list availability zones for the default volume type: %w", err)
		}
		zoneID, err := defaultAvailabilityZoneID(zoneResult)
		if err != nil {
			return err
		}
		params = map[string]string{"zoneId": zoneID}
	}

	result, err := apiClient.Get(fmt.Sprintf("/v1/%s/volume_default_id", projectID), params)
	if err != nil {
		return fmt.Errorf("failed to get the default volume type ID: %w", err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func runGet(cmd *cobra.Command, args []string) error {
	volumeTypeID, _ := cmd.Flags().GetString("volume-type-id")
	if err := validator.ValidateID(volumeTypeID, "volume-type-id"); err != nil {
		return err
	}
	return getDiscovery(cmd, "/v1/%s/volume_types/"+volumeTypeID, "get volume type "+volumeTypeID, nil)
}

func runGetZone(cmd *cobra.Command, args []string) error {
	zoneID, _ := cmd.Flags().GetString("volume-type-zone-id")
	if err := validator.ValidateID(zoneID, "volume-type-zone-id"); err != nil {
		return err
	}
	return getDiscovery(cmd, "/v1/%s/volume_type_zones/"+zoneID, "get volume type zone "+zoneID, nil)
}

func getDiscovery(cmd *cobra.Command, pathTemplate, description string, params map[string]string) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf(pathTemplate, projectID), params)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", description, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func optionalZoneParams(cmd *cobra.Command) map[string]string {
	zoneID, _ := cmd.Flags().GetString("zone-id")
	if zoneID == "" {
		return nil
	}
	return map[string]string{"zoneId": zoneID}
}
