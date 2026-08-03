package server

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

var attachFloatingIPCmd = &cobra.Command{
	Use:   "attach-floating-ip",
	Short: "Attach a floating IP to a server's network interface",
	Long: `Attach a floating IP (WAN IP) to a network interface of a vServer instance.

The floating IP is attached to the network interface given by
--network-interface-id on the specified server.`,
	RunE: func(cmd *cobra.Command, args []string) error { return runFloatingIPAction(cmd, "attach") },
}

var detachFloatingIPCmd = &cobra.Command{
	Use:   "detach-floating-ip",
	Short: "Detach a floating IP from a server's network interface",
	Long: `Detach a floating IP (WAN IP) from a network interface of a vServer instance.

The floating IP is detached from the network interface given by
--network-interface-id on the specified server.`,
	RunE: func(cmd *cobra.Command, args []string) error { return runFloatingIPAction(cmd, "detach") },
}

func init() {
	for _, c := range []*cobra.Command{attachFloatingIPCmd, detachFloatingIPCmd} {
		f := c.Flags()
		f.String("server-id", "", "Server ID (required)")
		f.String("floating-ip-id", "", "Floating IP ID (required)")
		f.String("network-interface-id", "", "Network interface ID (required)")

		for _, name := range []string{"server-id", "floating-ip-id", "network-interface-id"} {
			if err := c.MarkFlagRequired(name); err != nil {
				panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
			}
		}
	}
}

// runFloatingIPAction runs the attach or detach floating-IP request; action is "attach" or "detach".
func runFloatingIPAction(cmd *cobra.Command, action string) error {
	serverID, _ := cmd.Flags().GetString("server-id")
	floatingIPID, _ := cmd.Flags().GetString("floating-ip-id")
	interfaceID, _ := cmd.Flags().GetString("network-interface-id")

	for _, check := range []struct{ val, flag string }{
		{serverID, "server-id"},
		{floatingIPID, "floating-ip-id"},
		{interfaceID, "network-interface-id"},
	} {
		if err := validator.ValidateID(check.val, check.flag); err != nil {
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

	result, err := apiClient.Put(
		fmt.Sprintf("/v2/%s/servers/%s/wan-ips/%s/%s", projectID, serverID, floatingIPID, action),
		map[string]interface{}{"networkInterfaceId": interfaceID},
	)
	if err != nil {
		return fmt.Errorf("failed to %s floating IP %s on server %s: %w", action, floatingIPID, serverID, err)
	}

	return outputResult(cmd, cfg, result)
}
