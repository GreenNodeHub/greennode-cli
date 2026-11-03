package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups, all or per instance",
	Long: "List the MemoryStore backups in your project, or just those of one instance " +
		"with --instance-id.\n\n" +
		"These are two different endpoints: without --instance-id the listing is " +
		"paginated (items under 'content'), with it the API returns every backup of that " +
		"instance as a plain array and --page/--page-size do not apply.",
	Args: cobra.NoArgs,
	RunE: runList,
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a backup",
	Long: "Show the full record of one backup.\n\n" +
		"It carries the spec of the instance the backup came from — flavor, config group, " +
		"subnets — which is what 'backup restore' uses to build the new instance.",
	Args: cobra.NoArgs,
	RunE: runGet,
}

var getFreeStorageCmd = &cobra.Command{
	Use:   "get-free-storage",
	Short: "Show the free backup storage allowance and how much is used",
	Long: "Show the free backup storage that comes with your instances' flavors and how " +
		"much of it your MemoryStore backups occupy.\n\n" +
		"The allowance is the sum of your instances' own allowances, so it changes as " +
		"instances come and go. Storage beyond it is billed; buy or resize it with " +
		"'grn vdb memorystore backup-storage'.",
	Args: cobra.NoArgs,
	RunE: runGetFreeStorage,
}

func init() {
	l := listCmd.Flags()
	l.String("instance-id", "", "List only the backups of this instance")
	l.Int("page", 1, "Page number (1-based; ignored with --instance-id)")
	l.Int("page-size", vdbclient.DefaultPageSize, "Items per page (ignored with --instance-id)")
	listCmd.RegisterFlagCompletionFunc("instance-id", cli.ResourceCompletion(instanceResourceKey)) //nolint:errcheck

	g := getCmd.Flags()
	g.String("backup-id", "", "Backup ID (required)")
	getCmd.MarkFlagRequired("backup-id")                                                      //nolint:errcheck
	getCmd.RegisterFlagCompletionFunc("backup-id", cli.ResourceCompletion(BackupResourceKey)) //nolint:errcheck
}

func runList(cmd *cobra.Command, args []string) error {
	instanceID, _ := cmd.Flags().GetString("instance-id")

	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	if instanceID != "" {
		if err := validateInstanceID(instanceID); err != nil {
			return err
		}
		// Per-instance listing hangs off the INSTANCE, not the backups collection.
		result, err := apiClient.Get(instanceBase+"/"+instanceID+"/backups", nil)
		if err != nil {
			return fmt.Errorf("failed to list the backups of MemoryStore instance %s: %w", instanceID, err)
		}
		return vdbclient.OutputWithColumns(cmd, result, instanceBackupColumns)
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: page, PageSize: pageSize})

	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return fmt.Errorf("failed to list MemoryStore backups: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, backupColumns)
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

	// /backups/{id}/detail — the relational API spells it /backups/detail/{id}.
	result, err := apiClient.Get(basePath+"/"+backupID+"/detail", nil)
	if err != nil {
		return fmt.Errorf("failed to get backup %s: %w", backupID, err)
	}
	if _, ok := vdbclient.PayloadObject(result); !ok {
		return fmt.Errorf("backup %s not found (the API returned an empty payload; "+
			"a backup whose creation failed reads this way — check 'instance list-histories')", backupID)
	}

	return vdbclient.Output(cmd, result)
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

	return vdbclient.Output(cmd, result)
}
