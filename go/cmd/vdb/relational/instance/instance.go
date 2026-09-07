package instance

import (
	"github.com/spf13/cobra"
)

// InstanceCmd is the parent command for Relational Database instance commands.
var InstanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage Relational Database instances",
	Long: "List, inspect and manage vDB Relational Database instances " +
		"(MySQL, MariaDB, PostgreSQL).\n\n" +
		"Instance IDs start with 'db-'. The listing, get and history endpoints also serve " +
		"PostgreSQL Clusters ('pg-'), but every command here that CHANGES an instance " +
		"rejects those IDs — clusters are managed with 'grn vdb postgresql cluster'.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() //nolint:errcheck
	},
}

func init() {
	// Read
	InstanceCmd.AddCommand(listCmd)
	InstanceCmd.AddCommand(getCmd)
	InstanceCmd.AddCommand(listHistoriesCmd)
	InstanceCmd.AddCommand(listSecrulesCmd)
	InstanceCmd.AddCommand(listReplicasCmd)

	// Lifecycle
	InstanceCmd.AddCommand(createCmd)
	InstanceCmd.AddCommand(resizeInstanceCmd)
	InstanceCmd.AddCommand(resizeStorageCmd)
	InstanceCmd.AddCommand(deleteCmd)
	for _, spec := range actions {
		InstanceCmd.AddCommand(newActionCmd(spec))
	}

	// Replicas
	InstanceCmd.AddCommand(createReplicaCmd)
	InstanceCmd.AddCommand(detachReplicaCmd)

	// Configuration
	InstanceCmd.AddCommand(updateSettingsCmd)
	InstanceCmd.AddCommand(updateConfigGroupCmd)
	InstanceCmd.AddCommand(updateSecruleCmd)
}
