package backupstorage

import (
	"context"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vdbclient"
	"github.com/spf13/cobra"
)

// Completers for the two ids these commands take. Each is bound in the init() of the
// file that defines the flag — see the init-order note in
// cmd/vdb/relational/instance/completion.go.
//
// Both are API-backed and read the same endpoints the read commands use, which is why
// the mapping is worth stating once: on THIS API, basePath lists the packages you can
// buy and basePath+"/information" the storage you own. MemoryStore inverts that pair.

func storageIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		apiClient, err := vdbclient.BuildClient(cmd)
		if err != nil {
			return nil, err
		}
		result, err := apiClient.Get(basePath+"/information", nil)
		if err != nil {
			return nil, err
		}
		return cli.ExtractIDs(vdbclient.Unwrap(result), "id"), nil
	})
}

// packageIDCompletion suggests packageId, not id. Two details matter: the packages can
// arrive nested under their engine group, so the listing is flattened first — the same
// shape the table shows — and packageId is a JSON NUMBER, so it needs
// vdbclient.ExtractIDValues rather than cli.ExtractIDs, which would find nothing.
func packageIDCompletion() cli.CompFunc {
	return cli.FlagFromAPI(func(_ context.Context, cmd *cobra.Command) ([]string, error) {
		apiClient, err := vdbclient.BuildClient(cmd)
		if err != nil {
			return nil, err
		}
		result, err := apiClient.Get(basePath, nil)
		if err != nil {
			return nil, err
		}
		return vdbclient.ExtractIDValues(flattenPackages(vdbclient.Unwrap(result)), "packageId"), nil
	})
}
