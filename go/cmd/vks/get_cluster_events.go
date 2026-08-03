package vks

import (
	"fmt"
	"os"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var getClusterEventsCmd = &cobra.Command{
	Use:   "get-cluster-events",
	Short: "Get the list of events for a VKS cluster",
	RunE:  runGetClusterEvents,
}

func init() {
	f := getClusterEventsCmd.Flags()
	f.String("cluster-id", "", "Cluster ID (required)")
	f.String("action", "", "Filter by action")
	f.String("type", "", "Filter by event type")
	f.Int("page", 0, "Page number (0-based)")
	f.Int("page-size", 50, "Page size")

	getClusterEventsCmd.MarkFlagRequired("cluster-id")
}

func runGetClusterEvents(cmd *cobra.Command, args []string) error {
	clusterID, _ := cmd.Flags().GetString("cluster-id")
	action, _ := cmd.Flags().GetString("action")
	eventType, _ := cmd.Flags().GetString("type")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	if err := validator.ValidateID(clusterID, "cluster-id"); err != nil {
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
		fmt.Sprintf("/v1/clusters/%s/events", clusterID), params,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return outputResult(cmd, result)
}
