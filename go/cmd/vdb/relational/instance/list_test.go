package instance

import (
	"testing"

	"github.com/greennodehub/greennode-cli/internal/vdbclient"
)

func TestListFlagDefaults(t *testing.T) {
	// A fresh command must page from 1: vdb's pageNumber is 1-based, so a 0
	// default would skip the first page of results.
	got := listOptions(listCmd)

	if got.Page != 1 {
		t.Errorf("default page = %d, want 1", got.Page)
	}
	if got.PageSize != vdbclient.DefaultPageSize {
		t.Errorf("default page-size = %d, want %d", got.PageSize, vdbclient.DefaultPageSize)
	}
	if got.Name != "" {
		t.Errorf("default name = %q, want empty", got.Name)
	}
	if len(got.Statuses) != 0 {
		t.Errorf("default statuses = %v, want empty", got.Statuses)
	}
}

func TestListFlagsMapToQuery(t *testing.T) {
	cmd := listCmd
	defer func() {
		// listCmd is package-level state shared with the command tree; restore it.
		_ = cmd.Flags().Set("page", "1")
		_ = cmd.Flags().Set("page-size", "50")
		_ = cmd.Flags().Set("name", "")
		_ = cmd.Flags().Set("status", "")
	}()

	for flag, value := range map[string]string{
		"page": "2", "page-size": "10", "name": "my-db", "status": "ACTIVE, ERROR",
	} {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatalf("setting --%s: %v", flag, err)
		}
	}

	got := vdbclient.BuildListQuery(listOptions(cmd)).Encode()
	want := "name=my-db&pageNumber=2&pageSize=10&status=ACTIVE&status=ERROR"
	if got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
}
