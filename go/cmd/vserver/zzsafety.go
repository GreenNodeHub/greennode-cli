package vserver

import "strings"

func init() {
	for _, route := range []string{
		"dhcp associate-vpc", "dhcp create", "network-interface create", "network-interface update", "network-interface update-tags",
		"placement-group create", "placement-group update", "secgroup create", "secgroup rule create",
		"server attach-external-interface", "server attach-floating-ip", "server attach-internal-interface", "server create-image",
		"server detach-external-interface", "server detach-floating-ip", "server detach-internal-interface", "server resize", "server start",
		"server update-secgroup", "subnet create", "user-image update-tags",
	} {
		cmd, remaining, err := VServerCmd.Find(strings.Fields(route))
		if err != nil || len(remaining) != 0 || cmd.RunE == nil {
			panic("invalid vServer preview route: " + route)
		}
		cmd.Flags().Bool("dry-run", false, "Preview without contacting the API")
	}
}
