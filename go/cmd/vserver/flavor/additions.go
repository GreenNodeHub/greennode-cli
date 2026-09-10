package flavor

func init() {
	FlavorCmd.AddCommand(getCmd)
	FlavorCmd.AddCommand(listCustomsCmd)
	FlavorCmd.AddCommand(listCustomClusterCmd)
	FlavorCmd.AddCommand(listClusterCmd)
	FlavorCmd.AddCommand(listByZoneCmd)
}
