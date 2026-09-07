package backup

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// BackupResourceKey is the shared completion key for a backup ID. Registered here
// because this package owns the listing; consumers dispatch through the registry.
const BackupResourceKey = "vdb:relational-backup"

func init() {
	cli.RegisterResourceCompleter(BackupResourceKey, cli.FlagFromAPI(backupIDs))
}

// backupIDs suggests backup IDs from the paginated listing. One page of 100 is
// plenty for completion; a project with more simply gets the first page.
func backupIDs(_ context.Context, cmd *cobra.Command) ([]string, error) {
	apiClient, err := vdbclient.BuildClient(cmd)
	if err != nil {
		return nil, err
	}

	query := vdbclient.BuildListQuery(vdbclient.ListOptions{Page: 1, PageSize: 100})
	result, err := vdbclient.Get(apiClient, basePath, query)
	if err != nil {
		return nil, err
	}

	// The items live under "content", which cli.ExtractIDs finds by scanning for the
	// only array in the payload.
	return cli.ExtractIDs(vdbclient.Unwrap(result), "id"), nil
}
