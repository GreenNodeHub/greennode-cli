package vks

import "github.com/spf13/cobra"

var listNodegroupImagesCmd = &cobra.Command{
	Use:   "list-nodegroup-images",
	Short: "List images available for VKS node groups",
	RunE:  runListNodegroupImages,
}

func runListNodegroupImages(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}
	result, err := apiClient.Get("/v1/node-group-images", nil)
	if err != nil {
		return err
	}
	return outputResult(cmd, result)
}
