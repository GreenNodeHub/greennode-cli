package volume

func init() {
	VolumeCmd.AddCommand(listByServerCmd)
	VolumeCmd.AddCommand(getBootCmd)
	VolumeCmd.AddCommand(historyCmd)
	VolumeCmd.AddCommand(mappingCmd)
	VolumeCmd.AddCommand(changeDeviceTypeCmd)
	VolumeCmd.AddCommand(renameCmd)
	VolumeCmd.AddCommand(attachCmd)
	VolumeCmd.AddCommand(detachCmd)
	VolumeCmd.AddCommand(snapshotCmd)
	VolumeCmd.AddCommand(snapshotPolicyCmd)
}
