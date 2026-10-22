package secgroup

func init() {
	SecgroupCmd.AddCommand(updateCmd)
	SecgroupCmd.AddCommand(listServersCmd)
}
