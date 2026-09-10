package vks

import "github.com/spf13/cobra"

var getWorkspaceCmd = &cobra.Command{
	Use:   "get-workspace",
	Short: "Get the current user's VKS workspace",
	RunE:  runGetWorkspace,
}

func runGetWorkspace(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Get("/v1/workspace", nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
