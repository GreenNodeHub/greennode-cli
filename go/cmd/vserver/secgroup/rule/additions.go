package rule

func init() {
	RuleCmd.AddCommand(updateCmd)
	RuleCmd.AddCommand(listSamplesCmd)
}
