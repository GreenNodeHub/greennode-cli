package server

import (
	"fmt"
	"net/http"

	"github.com/greennodehub/greennode-cli/internal/redact"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var listBySubnetCmd = &cobra.Command{Use: "list-by-subnet", Short: "List servers in a subnet", RunE: runListBySubnet}
var getExternalInterfaceCmd = &cobra.Command{Use: "get-external-interface", Short: "Get an attached external network interface", RunE: runGetExternalInterface}
var listActionsCmd = &cobra.Command{Use: "list-actions", Short: "List actions for a server", RunE: runListActions}
var getConsoleLogCmd = &cobra.Command{Use: "get-console-log", Short: "Get a server console log", RunE: runGetConsoleLog}
var getConsoleURLCmd = &cobra.Command{Use: "get-console-url", Short: "Get a server console URL", RunE: runGetConsoleURL}
var listSecgroupsCmd = &cobra.Command{Use: "list-secgroups", Short: "List security groups attached to a server", RunE: runListSecgroups}

func init() {
	getConsoleURLCmd.Flags().Bool("show-secret", false, "Reveal the console URL")
	listBySubnetCmd.Flags().String("subnet-id", "", "Subnet ID (required)")
	if err := listBySubnetCmd.MarkFlagRequired("subnet-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "subnet-id", err))
	}
	getExternalInterfaceCmd.Flags().String("network-interface-id", "", "External network interface ID (required)")
	if err := getExternalInterfaceCmd.MarkFlagRequired("network-interface-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "network-interface-id", err))
	}
	for _, cmd := range []*cobra.Command{listActionsCmd, getConsoleLogCmd, getConsoleURLCmd, listSecgroupsCmd} {
		cmd.Flags().String("server-id", "", "Server ID (required)")
		if err := cmd.MarkFlagRequired("server-id"); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "server-id", err))
		}
	}
}

func runListBySubnet(cmd *cobra.Command, args []string) error {
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	if err := validator.ValidateID(subnetID, "subnet-id"); err != nil {
		return err
	}
	return runServerGet(cmd, "/v2/%s/servers/subnets/"+subnetID, "list servers in subnet "+subnetID, true)
}

func runGetExternalInterface(cmd *cobra.Command, args []string) error {
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
		return err
	}
	return runServerGet(cmd, "/v2/%s/servers/external-network-interfaces/"+interfaceID, "get external network interface "+interfaceID, false)
}

func runListActions(cmd *cobra.Command, args []string) error {
	return runServerResourceGet(cmd, "actions", "list server actions")
}

func runGetConsoleLog(cmd *cobra.Command, args []string) error {
	return runServerResourceGet(cmd, "console-log", "get the server console log")
}

func runGetConsoleURL(cmd *cobra.Command, args []string) error {
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
	response, err := apiClient.RequestWithStatusNoRetrySensitive(http.MethodGet, fmt.Sprintf("/v2/%s/servers/%s/console-url", projectID, serverID), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get the server console URL: %w", err)
	}
	result := response.Data
	show, _ := cmd.Flags().GetBool("show-secret")
	if !show {
		result = maskConsoleURL(result)
	}
	return vserverclient.Output(cmd, cfg, result)
}

func maskConsoleURL(value any) any {
	switch v := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, item := range v {
			result[key] = maskConsoleURL(item)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = maskConsoleURL(item)
		}
		return result
	case string:
		return redact.Value
	default:
		return value
	}
}

func runListSecgroups(cmd *cobra.Command, args []string) error {
	return runServerResourceGet(cmd, "sec-groups", "list attached security groups")
}

func runServerResourceGet(cmd *cobra.Command, suffix, description string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	return runServerGet(cmd, "/v2/%s/servers/"+serverID+"/"+suffix, description, false)
}

func runServerGet(cmd *cobra.Command, pathTemplate, description string, listOutput bool) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	result, err := apiClient.Get(fmt.Sprintf(pathTemplate, projectID), nil)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", description, err)
	}
	if listOutput {
		return outputServerList(cmd, cfg, result)
	}
	return vserverclient.Output(cmd, cfg, result)
}
