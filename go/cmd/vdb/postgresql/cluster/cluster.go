package cluster

import (
	"github.com/spf13/cobra"
)

// ClusterCmd is the parent command for PostgreSQL Cluster commands.
var ClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage PostgreSQL Clusters",
	Long: "Create, inspect, resize and delete vDB PostgreSQL Clusters.\n\n" +
		"Some of these call the Relational Database API, which is where the cluster " +
		"product's read, reboot, delete and security-rule operations live; each " +
		"command's help says which. Cluster IDs start with 'pg-' and a 'db-' ID is " +
		"rejected, since those relational endpoints would otherwise act on a " +
		"Relational Database instance.",
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
	ClusterCmd.AddCommand(getVolumeUsedCmd)
	ClusterCmd.AddCommand(listSecrulesCmd)

	// Lifecycle
	ClusterCmd.AddCommand(createCmd)
	ClusterCmd.AddCommand(resizeCmd)
	ClusterCmd.AddCommand(rebootCmd)
	ClusterCmd.AddCommand(deleteCmd)

	// Configuration
	ClusterCmd.AddCommand(updateSettingsCmd)
	ClusterCmd.AddCommand(updateConfigGroupCmd)
	ClusterCmd.AddCommand(updateSecruleCmd)
}
