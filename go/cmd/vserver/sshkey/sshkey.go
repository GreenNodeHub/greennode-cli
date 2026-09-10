package sshkey

import (
	"github.com/spf13/cobra"
)

// SSHKeyCmd is the parent command for all SSH key subcommands.
var SSHKeyCmd = &cobra.Command{
	Use:   "sshkey",
	Short: "Manage SSH keys",
	Long:  "List and delete SSH keys.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	SSHKeyCmd.AddCommand(listCmd)
	SSHKeyCmd.AddCommand(createCmd)
	SSHKeyCmd.AddCommand(importCmd)
	SSHKeyCmd.AddCommand(deleteCmd)
}
