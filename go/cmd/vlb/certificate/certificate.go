package certificate

import (
	"github.com/spf13/cobra"
)

// CertificateCmd is the parent command for certificate (CA) subcommands.
var CertificateCmd = &cobra.Command{
	Use:     "certificate",
	Aliases: []string{"ca"},
	Short:   "Manage load balancer certificates",
	Long:    "Import, inspect and delete the CA and TLS/SSL certificates used by HTTPS listeners.",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	CertificateCmd.AddCommand(listCmd)
	CertificateCmd.AddCommand(getCmd)
	CertificateCmd.AddCommand(createCmd)
	CertificateCmd.AddCommand(deleteCmd)
}
