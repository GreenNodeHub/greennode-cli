package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups",
	Long: "List the backups in your project, or just those of one instance with " +
		"--instance-id.\n\n" +
		"These are two different endpoints: without --instance-id the listing is " +
		"paginated (items under 'content'), with it the API returns every backup of " +
		"that instance as a plain array and --page/--page-size do not apply.",
	Args: cobra.NoArgs,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.String("instance-id", "", "List only the backups of this instance")
	f.Int("page", 1, "Page number (1-based; ignored with --instance-id)")
	f.Int("page-size", vdbclient.DefaultPageSize, "Items per page (ignored with --instance-id)")

	// Bound here, next to the flag: see the init-order note in
	// cmd/vdb/relational/instance/completion.go.
	listCmd.RegisterFlagCompletionFunc("instance-id", cli.ResourceCompletion(instanceResourceKey)) //nolint:errcheck
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
		// Per-instance listing: a bare array, no paging.
		result, err := apiClient.Get(basePath+"/insId/"+instanceID, nil)
		if err != nil {
			return fmt.Errorf("failed to list the backups of database instance %s: %w", instanceID, err)
		}
		// This endpoint leaves dbInstanceId and instanceName empty on every row — you
		// asked by instance, so it does not repeat it back. Showing them would be two
		// blank columns.
		return vdbclient.OutputWithColumns(cmd, result, instanceBackupColumns)
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: page, PageSize: pageSize})

	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, backupColumns)
}
