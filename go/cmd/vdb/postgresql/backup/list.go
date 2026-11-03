package backup

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the backup records of every PostgreSQL Cluster",
	Long: "List the backup record of each cluster in the project: its policy, its " +
		"location and how much space its backups take.\n\n" +
		"This endpoint takes no pagination or filters — it returns the whole list. Use " +
		"'backup list-restore-points' for the individual snapshots of one cluster.",
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	apiClient, err := createClient(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Get(backupBase, nil)
	if err != nil {
		return fmt.Errorf("failed to list cluster backups: %w", err)
	}

	return vdbclient.OutputWithColumns(cmd, result, backupColumns)
}
