package userimage

import (
	"github.com/spf13/cobra"
)

// UserImageCmd is the parent command for all user image subcommands.
var UserImageCmd = &cobra.Command{
	Use:   "user-image",
	Short: "Manage user images",
	Long:  "List and delete user images (custom images created from your servers).",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	UserImageCmd.AddCommand(listCmd)
	UserImageCmd.AddCommand(updateTagsCmd)
	UserImageCmd.AddCommand(deleteCmd)
}
