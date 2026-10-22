package userimage

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a user image",
	RunE:  runGet,
}

func init() {
	getCmd.Flags().String("user-image-id", "", "User image ID (required)")
	if err := getCmd.MarkFlagRequired("user-image-id"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "user-image-id", err))
	}
}

func runGet(cmd *cobra.Command, args []string) error {
	imageID, _ := cmd.Flags().GetString("user-image-id")
	if err := validator.ValidateID(imageID, "user-image-id"); err != nil {
		return err
	}

	apiClient, cfg, err := vserverclient.BuildOperationClient(cmd, true)
	if err != nil {
		return err
	}
	projectID, err := vserverclient.ProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(fmt.Sprintf("/v2/%s/user-images/%s", projectID, imageID), nil)
	if err != nil {
		return fmt.Errorf("failed to get user image %s: %w", imageID, err)
	}
	return outputImageList(cmd, cfg, result)
}
