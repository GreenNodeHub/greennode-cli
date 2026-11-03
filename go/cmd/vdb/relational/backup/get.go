package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// instanceResourceKey is the completion key registered by the instance package.
// Consumers use the literal, as vks does for the vserver keys, so no package
// dependency is needed just to share a constant.
const instanceResourceKey = "vdb:relational-instance"

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a backup",
	Long: "Show the full record of one backup.\n\n" +
		"Beyond the obvious fields it carries the spec of the instance the backup was " +
		"taken from — flavor, storage, subnets, config group, username — which is what " +
		"'backup restore' uses to build the new instance.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("backup-id", "", "Backup ID (required)")
	getCmd.MarkFlagRequired("backup-id") //nolint:errcheck

	getCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey)) //nolint:errcheck
}

func runGet(cmd *cobra.Command, args []string) error {
	backupID, _ := cmd.Flags().GetString("backup-id")
	if err := requireBackupID(backupID); err != nil {
		return err
	}

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	// Note the path shape: /backups/detail/{backupId}. MemoryStore spells the same
	// operation /backups/{backupId}/detail.
	result, err := apiClient.Get(basePath+"/detail/"+backupID, nil)
	if err != nil {
		return fmt.Errorf("failed to get backup %s: %w", backupID, err)
	}

	// The API answers "not found" with HTTP 200 and a null payload, so a miss has to
	// be detected here rather than by the status code.
	if _, ok := vdbclient.PayloadObject(result); !ok {
		return fmt.Errorf("backup %s not found (the API returned an empty payload; "+
			"a backup whose creation failed reads this way — check 'instance list-histories')", backupID)
	}

	// A backup record has nested arrays (netIds, sharedActions), so print it as
	// key/value rather than letting table extraction pick one.
	return vdbclient.Output(cmd, result)
}
