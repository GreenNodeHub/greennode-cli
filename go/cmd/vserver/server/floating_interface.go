package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var attachInternalFloatingCmd = &cobra.Command{Use: "attach-internal-floating", Short: "Attach an internal interface with a floating IP", RunE: runAttachInternalFloating}
var detachInternalFloatingCmd = &cobra.Command{Use: "detach-internal-floating", Short: "Detach an internal interface with a floating IP", RunE: runDetachInternalFloating}

func init() {
	af := attachInternalFloatingCmd.Flags()
	af.String("server-id", "", "Server ID (required)")
	af.String("subnet-id", "", "Subnet ID (required)")
	af.String("wan-ip-id", "", "WAN IP ID (required)")
	af.String("ip", "", "Requested private IP")
	af.String("zone-id", "", "Availability zone ID")
	af.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	af.Bool("dry-run", false, "Validate the attachment without executing")
	for _, name := range []string{"server-id", "subnet-id", "wan-ip-id"} {
		if err := attachInternalFloatingCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}

	df := detachInternalFloatingCmd.Flags()
	df.String("server-id", "", "Server ID (required)")
	df.String("network-interface-id", "", "Network interface ID (required)")
	df.String("zone-id", "", "Availability zone ID")
	df.StringArray("tag", nil, "Tag in key=value form (repeatable)")
	df.Bool("dry-run", false, "Validate the detachment without executing")
	for _, name := range []string{"server-id", "network-interface-id"} {
		if err := detachInternalFloatingCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runAttachInternalFloating(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	wanIPID, _ := cmd.Flags().GetString("wan-ip-id")
	ip, _ := cmd.Flags().GetString("ip")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{{serverID, "server-id"}, {subnetID, "subnet-id"}, {wanIPID, "wan-ip-id"}} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("attach")
		return nil
	}
	body := map[string]any{"serverId": serverID, "subnetId": subnetID, "wanIpId": wanIPID, "ip": nilIfEmpty(ip), "zoneId": nilIfEmpty(zoneID), "tags": tags}
	return runInternalFloatingWrite(cmd, serverID, "POST", body)
}

func runDetachInternalFloating(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	rawTags, _ := cmd.Flags().GetStringArray("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	for _, check := range []struct{ value, flag string }{{serverID, "server-id"}, {interfaceID, "network-interface-id"}} {
		if err := validator.ValidateID(check.value, check.flag); err != nil {
			return err
		}
	}
	tags, err := parseTags(rawTags)
	if err != nil {
		return err
	}
	if dryRun {
		cli.DryRunNotice("detach")
		return nil
	}
	body := map[string]any{"serverId": serverID, "networkInterfaceId": interfaceID, "zoneId": nilIfEmpty(zoneID), "tags": tags}
	return runInternalFloatingWrite(cmd, serverID, "DELETE", body)
}

func runInternalFloatingWrite(cmd *cobra.Command, serverID, method string, body map[string]any) error {
	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v2/%s/servers/%s/internal-network-interfaces-floating", projectID, serverID)
	var result any
	if method == "POST" {
		result, err = apiClient.Post(path, body)
	} else {
		result, err = apiClient.DeleteWithBody(path, body)
	}
	if err != nil {
		return fmt.Errorf("failed to update internal floating interface for server %s: %w", serverID, err)
	}
	return vserverclient.Output(cmd, cfg, result)
}
