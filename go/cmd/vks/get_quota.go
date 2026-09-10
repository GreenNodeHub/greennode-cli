package vks

import (
	"github.com/spf13/cobra"
)

var getQuotaCmd = &cobra.Command{
	Use:   "get-quota",
	Short: "Get VKS quota for the current user",
	RunE:  runGetQuota,
}

func runGetQuota(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get("/v1/quota", nil)
	if err != nil {
		return err
	}

	return outputResult(cmd, result)
}
