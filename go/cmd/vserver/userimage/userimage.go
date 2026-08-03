package userimage

import (
	"github.com/spf13/cobra"
)

// UserImageCmd is the parent command for all user image subcommands.
var UserImageCmd = &cobra.Command{
	Use:   "user-image",
	Short: "Manage user images",
	Long:  "List and delete user images (custom images created from your servers).",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	UserImageCmd.AddCommand(listCmd)
	UserImageCmd.AddCommand(updateTagsCmd)
	UserImageCmd.AddCommand(deleteCmd)
}
