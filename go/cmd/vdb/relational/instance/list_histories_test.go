package instance

import (
	"testing"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
)

func TestHistoriesQueryHasPagingOnly(t *testing.T) {
	// The histories endpoint accepts pageNumber/pageSize and nothing else, and its
	// pages are 1-based like every other vdb listing.
	got := vdbclient.BuildListQuery(historiesOptions(listHistoriesCmd)).Encode()
	want := "pageNumber=1&pageSize=50"

	if got != want {
		t.Errorf("default query = %q, want %q", got, want)
	}
}

func TestHistoriesPathDiffersFromDetailPath(t *testing.T) {
	// Relational hangs get-by-id off an extra "id" segment but the action log
	// directly off the instance ID. Getting these two the wrong way round is a 404
	// that looks like a missing instance.
	const id = "db-1234"

	if got, want := detailPath(id), "/vdb-relational/v1/database-instances/id/db-1234"; got != want {
		t.Errorf("detailPath = %q, want %q", got, want)
	}
	if got, want := historiesPath(id), "/vdb-relational/v1/database-instances/db-1234/histories"; got != want {
		t.Errorf("historiesPath = %q, want %q", got, want)
	}
}
