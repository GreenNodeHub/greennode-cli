package vpc

func init() {
	VpcCmd.AddCommand(listActiveCmd)
	VpcCmd.AddCommand(updateCmd)
	VpcCmd.AddCommand(enableDNSCmd)
}
