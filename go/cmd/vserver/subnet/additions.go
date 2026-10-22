package subnet

func init() {
	SubnetCmd.AddCommand(updateCmd)
	SubnetCmd.AddCommand(createSecondaryCmd)
	SubnetCmd.AddCommand(deleteSecondaryCmd)
}
