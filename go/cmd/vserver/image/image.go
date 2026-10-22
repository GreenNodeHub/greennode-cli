package image

import (
	"github.com/spf13/cobra"
)

// ImageCmd is the parent command for all image subcommands.
var ImageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage vServer images",
	Long:  "List available vServer images by type (os, gpu).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	ImageCmd.AddCommand(listCmd)
}
