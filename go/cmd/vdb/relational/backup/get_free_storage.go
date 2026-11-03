package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var getFreeStorageCmd = &cobra.Command{
	Use:   "get-free-storage",
	Short: "Show the free backup storage allowance and how much is used",
	Long: "Show the free backup storage that comes with your instances' flavors and how " +
		"much of it your backups currently occupy.\n\n" +
		"Storage beyond this allowance is billed; buy or resize it with " +
		"'grn vdb relational backup-storage'.",
	Args: cobra.NoArgs,
	RunE: runGetFreeStorage,
}

func runGetFreeStorage(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(basePath+"/free-backup", nil)
	if err != nil {
		return fmt.Errorf("failed to read free backup storage usage: %w", err)
	}

	// Two scalar fields (freeBackupStorage, backupUsage), so a key/value view is the
	// whole story.
	return vdbclient.Output(cmd, result)
}
