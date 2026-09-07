// Package user holds the Kafka user commands.
//
// Users are a Kafka-only concept in vDB. The other three products have nothing
// comparable: MemoryStore authenticates with a single master password set through
// its instance settings, and Relational Database creates database users as part of
// the instance order. Do not model anything here on either.
//
// A user is a set of per-topic permissions plus the authentication mechanisms it may
// use. Its credentials are issued by the cluster and read back with `get-creds`.
package user

import (
	"github.com/spf13/cobra"
)

// UserCmd is the parent command for Kafka user commands.
var UserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage the users of a Kafka cluster",
	Long: "Create, inspect and delete the users of a vDB Kafka cluster, and read or " +
		"regenerate their credentials.\n\n" +
		"A user holds four kinds of permission — produce, consume, produce+consume and " +
		"admin — each either on a named list of topics or on all of them. Topics are " +
		"referenced by NAME here, not by ID.\n\n" +
		"A user can only use a mechanism the cluster itself accepts: enabling SASL on a " +
		"user does nothing until 'cluster update-authentication' enables it on the " +
		"cluster.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	UserCmd.AddCommand(listCmd)
	UserCmd.AddCommand(getCmd)
	UserCmd.AddCommand(getCredsCmd)
	UserCmd.AddCommand(createCmd)
	UserCmd.AddCommand(updateCmd)
	UserCmd.AddCommand(deleteCmd)
	UserCmd.AddCommand(generateCredsCmd)
}
