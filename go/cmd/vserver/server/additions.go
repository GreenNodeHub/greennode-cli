package server

func init() {
	ServerCmd.AddCommand(listBySubnetCmd)
	ServerCmd.AddCommand(getExternalInterfaceCmd)
	ServerCmd.AddCommand(listActionsCmd)
	ServerCmd.AddCommand(getConsoleLogCmd)
	ServerCmd.AddCommand(getConsoleURLCmd)
	ServerCmd.AddCommand(listSecgroupsCmd)
	ServerCmd.AddCommand(renameCmd)
	ServerCmd.AddCommand(attachInternalFloatingCmd)
	ServerCmd.AddCommand(detachInternalFloatingCmd)
	ServerCmd.AddCommand(migrateCmd)
	ServerCmd.AddCommand(startMigrationCmd)
	ServerCmd.AddCommand(completeMigrationCmd)
	ServerCmd.AddCommand(snapshotCmd)
	ServerCmd.AddCommand(snapshotPolicyCmd)
}
