// Package lbpackage implements `grn vlb package`. The directory is named
// lbpackage because `package` is a Go keyword.
package lbpackage

import (
	"github.com/spf13/cobra"
)

// PackageCmd is the parent command for load balancer package subcommands.
var PackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Browse load balancer packages",
	Long:  "List the packages a load balancer can be created with or resized to.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	PackageCmd.AddCommand(listCmd)
}
