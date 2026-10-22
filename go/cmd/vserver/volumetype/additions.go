package volumetype

func init() {
	VolumeTypeCmd.AddCommand(listAllCmd)
	VolumeTypeCmd.AddCommand(listZonesCmd)
	VolumeTypeCmd.AddCommand(getCmd)
	VolumeTypeCmd.AddCommand(getDefaultCmd)
	VolumeTypeCmd.AddCommand(getZoneCmd)
}
