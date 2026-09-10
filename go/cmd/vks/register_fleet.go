package vks

import (
	"fmt"
	"regexp"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var fleetNamePattern = regexp.MustCompile(`^[a-z0-9-]{5,20}$`)

var registerFleetCmd = &cobra.Command{
	Use:   "register-fleet",
	Short: "Register a VKS cluster with a fleet",
	RunE:  runRegisterFleet,
}

func init() {
	f := registerFleetCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("fleet-type", "", "Fleet type: NEW or EXISTING (required)")
	f.String("fleet-id", "", "Fleet ID")
	f.String("fleet-name", "", "Fleet name (5-20 lowercase letters, digits, or hyphens)")
	f.String("enable-east-west-traffic", "", "Enable or disable east-west fleet traffic")
	f.String("enable-north-south-traffic", "", "Enable or disable north-south fleet traffic")
	f.Bool("dry-run", false, "Preview fleet registration without executing")

	registerFleetCmd.MarkFlagRequired("cluster-id")
	registerFleetCmd.MarkFlagRequired("fleet-type")
}

func buildFleetBody(fleetType, fleetID, fleetName, eastWestTraffic, northSouthTraffic string, changed map[string]bool) (map[string]any, error) {
	if fleetType != "NEW" && fleetType != "EXISTING" {
		return nil, fmt.Errorf("--fleet-type must be NEW or EXISTING, got %q", fleetType)
	}
	if changed["fleet-name"] && !fleetNamePattern.MatchString(fleetName) {
		return nil, fmt.Errorf("--fleet-name must contain 5-20 lowercase letters, digits, or hyphens")
	}

	body := map[string]any{"fleetType": fleetType}
	if changed["fleet-id"] {
		// Body-only ID; no documented pattern.
		body["id"] = fleetID
	}
	if changed["fleet-name"] {
		body["name"] = fleetName
	}
	if changed["enable-east-west-traffic"] {
		enabled, err := parseToggle("enable-east-west-traffic", eastWestTraffic)
		if err != nil {
			return nil, err
		}
		body["enableEastWestTraffic"] = enabled
	}
	if changed["enable-north-south-traffic"] {
		enabled, err := parseToggle("enable-north-south-traffic", northSouthTraffic)
		if err != nil {
			return nil, err
		}
		body["enableNorthSouthTraffic"] = enabled
	}
	return body, nil
}

func runRegisterFleet(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	fleetType, _ := cmd.Flags().GetString("fleet-type")
	fleetID, _ := cmd.Flags().GetString("fleet-id")
	fleetName, _ := cmd.Flags().GetString("fleet-name")
	eastWestTraffic, _ := cmd.Flags().GetString("enable-east-west-traffic")
	northSouthTraffic, _ := cmd.Flags().GetString("enable-north-south-traffic")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	changed := map[string]bool{
		"fleet-id":                   cmd.Flags().Changed("fleet-id"),
		"fleet-name":                 cmd.Flags().Changed("fleet-name"),
		"enable-east-west-traffic":   cmd.Flags().Changed("enable-east-west-traffic"),
		"enable-north-south-traffic": cmd.Flags().Changed("enable-north-south-traffic"),
	}
	body, err := buildFleetBody(fleetType, fleetID, fleetName, eastWestTraffic, northSouthTraffic, changed)
	if err != nil {
		return err
	}
	if dryRun {
		cli.PrintDryRun("register", fmt.Sprintf("fleet for cluster %s", clusterID), body)
		return nil
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Post(fmt.Sprintf("/v1/clusters/%s/register-fleet", clusterID), body)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
