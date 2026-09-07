package instance

import (
	"github.com/spf13/cobra"
)

// InstanceCmd is the parent command for MemoryStore instance commands.
var InstanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage MemoryStore instances",
	Long: "List, inspect and manage vDB MemoryStore (Redis) instances.\n\n" +
		"Two things differ from Relational Database and shape most of these commands: an " +
		"instance has no volume — capacity comes from the flavor, so there is no " +
		"storage resize — and access is guarded by a master password on the instance " +
		"rather than a database user.",
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
	InstanceCmd.AddCommand(listReplicasCmd)
	InstanceCmd.AddCommand(listSecrulesCmd)

	// Lifecycle. Note there is no resize-storage: MemoryStore has no volume.
	InstanceCmd.AddCommand(createCmd)
	InstanceCmd.AddCommand(resizeInstanceCmd)
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
