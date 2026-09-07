// Package cluster holds the Kafka cluster commands.
//
// Naming note: the noun is "cluster" here and in cmd/vdb/postgresql, but the two
// share no path, no ID space and no operation. Nothing in this package may be
// copied from there.
package cluster

import (
	"github.com/spf13/cobra"
)

// ClusterCmd is the parent command for Kafka cluster commands.
var ClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage Kafka clusters",
	Long: "Create, inspect, resize and delete vDB Kafka clusters.\n\n" +
		"A cluster is 3-10 brokers sharing one volume type and size per broker. Broker " +
		"count and storage size are changed by separate, individually chargeable " +
		"operations - there is no single 'resize'.\n\n" +
		"Authentication (mTLS / SASL) is a cluster-level switch; the users allowed to " +
		"connect and what they may do are managed with 'grn vdb kafka user'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	// Read
	ClusterCmd.AddCommand(listCmd)
	ClusterCmd.AddCommand(getCmd)
	ClusterCmd.AddCommand(listHistoriesCmd)
	ClusterCmd.AddCommand(listSecrulesCmd)

	// Lifecycle
	ClusterCmd.AddCommand(createCmd)
	ClusterCmd.AddCommand(deleteCmd)

	// Chargeable changes, one endpoint each
	ClusterCmd.AddCommand(resizeBrokersCmd)
	ClusterCmd.AddCommand(resizeStorageCmd)
	ClusterCmd.AddCommand(updateVolumeTypeCmd)

	// Settings
	ClusterCmd.AddCommand(updateAuthenticationCmd)
	ClusterCmd.AddCommand(updateConfigGroupCmd)
	ClusterCmd.AddCommand(updatePublicAccessCmd)

	// Security rules
	ClusterCmd.AddCommand(createSecruleCmd)
	ClusterCmd.AddCommand(deleteSecruleCmd)
}
