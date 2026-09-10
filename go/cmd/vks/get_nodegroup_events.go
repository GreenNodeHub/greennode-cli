package vks

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getNodegroupEventsCmd = &cobra.Command{
	Use:   "get-nodegroup-events",
	Short: "Get the list of events for a VKS node group",
	RunE:  runGetNodegroupEvents,
}

func init() {
	f := getNodegroupEventsCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("nodegroup-id", "", "Node group ID (required)")
	f.String("action", "", "Filter by action")
	f.String("type", "", "Filter by event type")
	f.Int("page", 0, "Page number (0-based)")
	f.Int("page-size", 10, "Page size")

	getNodegroupEventsCmd.MarkFlagRequired("cluster-id")
	getNodegroupEventsCmd.MarkFlagRequired("nodegroup-id")
}

func runGetNodegroupEvents(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	nodegroupID, _ := cmd.Flags().GetString("nodegroup-id")
	action, _ := cmd.Flags().GetString("action")
	eventType, _ := cmd.Flags().GetString("type")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(nodegroupID, "nodegroup-id"); err != nil {
		return err
	}
	if err := validatePage(page, pageSize); err != nil {
		return err
	}

	changed := map[string]bool{
		"action":    cmd.Flags().Changed("action"),
		"type":      cmd.Flags().Changed("type"),
		"page":      cmd.Flags().Changed("page"),
		"page-size": cmd.Flags().Changed("page-size"),
	}
	params := buildEventsQuery(action, eventType, page, pageSize, changed)

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(
		fmt.Sprintf("/v1/clusters/%s/node-groups/%s/events", clusterID, nodegroupID), params,
	)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}

func validatePage(page, pageSize int) error {
	if page < 0 {
		return fmt.Errorf("--page must be 0 or greater")
	}
	if pageSize < 1 {
		return fmt.Errorf("--page-size must be 1 or greater")
	}
	return nil
}
