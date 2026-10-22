package iam

import (
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/spf13/cobra"
)

var IamCmd = &cobra.Command{
	Use:   "iam",
	Short: "IAM account, credential, and authorization commands",
	Long: `Inspect GreenNode IAM accounts, credentials, and authorization policy.

This command group covers the documented IAM Accounts and Policies APIs. Every
remote change has an offline dry run, explicit confirmation, no automatic retry,
and documented-endpoint enforcement.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	IamCmd.AddCommand(
		whoamiCmd,
		identityProviderCmd,
		s3KeyCmd,
		serviceAccountCmd,
		swiftUserCmd,
		IAMUserCmd,
		actionCmd,
		groupCmd,
		policyCmd,
		productCmd,
		resourceCmd,
	)

	cli.RegisterService(IamCmd)
}
