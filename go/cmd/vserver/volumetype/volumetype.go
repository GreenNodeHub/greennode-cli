package volumetype

import (
	"github.com/spf13/cobra"
)

// VolumeTypeCmd is the parent command for all volume type subcommands.
var VolumeTypeCmd = &cobra.Command{
	Use:   "volume-type",
	Short: "Manage vServer volume types",
	Long:  "List available volume types for a zone.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	VolumeTypeCmd.AddCommand(listCmd)
}
