package vdb

import (
	"github.com/greennodehub/greennode-cli/cmd/vdb/kafka"
	"github.com/greennodehub/greennode-cli/cmd/vdb/memorystore"
	"github.com/greennodehub/greennode-cli/cmd/vdb/postgresql"
	"github.com/greennodehub/greennode-cli/cmd/vdb/relational"
	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// VdbCmd is the parent command for all vDB subcommands.
//
// vDB is four separate products behind one gateway (Relational, MemoryStore,
// Kafka, PostgreSQL Cluster) whose APIs share almost nothing — different paths,
// different response envelopes, different HTTP verbs for the same operation. The
// command tree mirrors that split instead of hiding it behind an --engine flag,
// so each group can follow its own API without leaking flags that only apply to
// one of them.
var VdbCmd = &cobra.Command{
	Use:   "vdb",
	Short: "VNG Database (vDB) commands",
	Long:  "Manage vDB database instances and clusters.",
	// Reject unknown subcommands (parent groups don't error by default in cobra).
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	VdbCmd.PersistentFlags().String("user-type", "",
		"Billing flow for chargeable operations: ROOT_USER (Checkout) or IAM_USER (Auto Payment). Default: ROOT_USER")
	VdbCmd.RegisterFlagCompletionFunc("user-type", cli.FlagValues(vdbclient.UserTypeValues...)) //nolint:errcheck

	VdbCmd.AddCommand(relational.RelationalCmd)
	VdbCmd.AddCommand(postgresql.PostgresqlCmd)
	VdbCmd.AddCommand(memorystore.MemorystoreCmd)
	VdbCmd.AddCommand(kafka.KafkaCmd)

	// Self-register with the CLI so root.go mounts this product automatically.
	cli.RegisterService(VdbCmd)
}
