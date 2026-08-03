package floatingip

import (
	"github.com/spf13/cobra"
)

// FloatingIPCmd is the parent command for all floating IP subcommands.
var FloatingIPCmd = &cobra.Command{
	Use:   "floating-ip",
	Short: "Manage floating IPs (WAN IPs)",
	Long:  "List floating IPs (public WAN IP addresses).",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	FloatingIPCmd.AddCommand(listCmd)
	FloatingIPCmd.AddCommand(deleteCmd)
}
