package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

// ── internal network interfaces ────────────────────────────────────────────────

var attachInternalInterfaceCmd = &cobra.Command{
	Use:   "attach-internal-interface",
	Short: "Attach an internal network interface to a server",
	Long: `Attach an internal network interface to a vServer instance.

The interface is created on the given subnet. Provide --ip to request a specific
private IP, or omit it to let the system assign one automatically.`,
	RunE: runAttachInternalInterface,
}

var detachInternalInterfaceCmd = &cobra.Command{
	Use:   "detach-internal-interface",
	Short: "Detach internal network interfaces from a server",
	Long:  "Detach one or more internal network interfaces from a vServer instance.",
	RunE:  runDetachInternalInterface,
}

// ── external network interfaces ────────────────────────────────────────────────

var attachExternalInterfaceCmd = &cobra.Command{
	Use:   "attach-external-interface",
	Short: "Attach an external network interface to a server",
	Long:  "Attach an existing external (elastic) network interface to a vServer instance.",
	RunE:  runAttachExternalInterface,
}

var detachExternalInterfaceCmd = &cobra.Command{
	Use:   "detach-external-interface",
	Short: "Detach an external network interface from a server",
	Long:  "Detach an external (elastic) network interface from a vServer instance.",
	RunE:  runDetachExternalInterface,
}

func init() {
	// attach-internal: server + subnet (+ optional ip)
	fai := attachInternalInterfaceCmd.Flags()
	fai.String("server-id", "", "Server ID (required)")
	fai.String("subnet-id", "", "Subnet ID to create the interface on (required)")
	fai.String("ip", "", "Private IP to request (optional; auto-assigned if omitted)")
	markRequired(attachInternalInterfaceCmd, "server-id", "subnet-id")

	// detach-internal: server + one or more interface IDs
	fdi := detachInternalInterfaceCmd.Flags()
	fdi.String("server-id", "", "Server ID (required)")
	fdi.String("network-interface-id", "", "Internal network interface IDs to detach (comma-separated, required)")
	markRequired(detachInternalInterfaceCmd, "server-id", "network-interface-id")

	// attach-external: server + external interface ID
	fae := attachExternalInterfaceCmd.Flags()
	fae.String("server-id", "", "Server ID (required)")
	fae.String("network-interface-id", "", "External network interface ID to attach (required)")
	markRequired(attachExternalInterfaceCmd, "server-id", "network-interface-id")

	// detach-external: server + external interface ID
	fde := detachExternalInterfaceCmd.Flags()
	fde.String("server-id", "", "Server ID (required)")
	fde.String("network-interface-id", "", "External network interface ID to detach (required)")
	markRequired(detachExternalInterfaceCmd, "server-id", "network-interface-id")
}

func markRequired(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

func runAttachInternalInterface(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	ip, _ := cmd.Flags().GetString("ip")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(subnetID, "subnet-id"); err != nil {
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

	body := map[string]interface{}{
		"subnetRequests": []interface{}{
			map[string]interface{}{
				"subnetId": subnetID,
				"ip":       nilIfEmpty(ip),
			},
		},
	}

	result, err := apiClient.Post(
		fmt.Sprintf("/v2/%s/servers/%s/internal-network-interfaces", projectID, serverID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to attach internal interface to server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}

func runDetachInternalInterface(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	interfaceIDs, _ := cmd.Flags().GetString("network-interface-id")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}

	ids := parseCommaSeparated(interfaceIDs)
	if len(ids) == 0 {
		return fmt.Errorf("at least one network interface ID is required (--network-interface-id)")
	}
	for _, id := range ids {
		if err := validator.ValidateID(id, "network-interface-id"); err != nil {
			return err
		}
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}
	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	body := map[string]interface{}{"networkInterfaceIds": ids}

	result, err := apiClient.DeleteWithBody(
		fmt.Sprintf("/v2/%s/servers/%s/internal-network-interfaces", projectID, serverID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to detach internal interfaces from server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}

func runAttachExternalInterface(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
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

	body := map[string]interface{}{"externalNetworkInterfaceId": interfaceID}

	result, err := apiClient.Post(
		fmt.Sprintf("/v2/%s/servers/%s/external-network-interfaces", projectID, serverID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to attach external interface to server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}

func runDetachExternalInterface(cmd *cobra.Command, args []string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")

	if err := validator.ValidateID(serverID, "server-id"); err != nil {
		return err
	}
	if err := validator.ValidateID(interfaceID, "network-interface-id"); err != nil {
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

	body := map[string]interface{}{"networkInterfaceId": interfaceID}

	result, err := apiClient.DeleteWithBody(
		fmt.Sprintf("/v2/%s/servers/%s/external-network-interfaces", projectID, serverID),
		body,
	)
	if err != nil {
		return fmt.Errorf("failed to detach external interface from server %s: %w", serverID, err)
	}

	return outputResult(cmd, cfg, result)
}
