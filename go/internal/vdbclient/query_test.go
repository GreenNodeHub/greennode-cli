package vdbclient

import (
	"testing"
)

func TestBuildListQuery(t *testing.T) {
	tests := []struct {
		name string
		opts ListOptions
		want string
	}{
		{
			name: "defaults when unset",
			opts: ListOptions{},
			want: "pageNumber=1&pageSize=50",
		},
		{
			// pageNumber is 1-based: --page 1 must go out as 1, not 0. A 0 here
			// would silently skip the first page on a 1-based API.
			name: "page passes through without offset",
			opts: ListOptions{Page: 1, PageSize: 20},
			want: "pageNumber=1&pageSize=20",
		},
		{
			name: "page 3",
			opts: ListOptions{Page: 3, PageSize: 20},
			want: "pageNumber=3&pageSize=20",
		},
		{
			name: "non-positive page and size fall back to defaults",
			opts: ListOptions{Page: 0, PageSize: -5},
			want: "pageNumber=1&pageSize=50",
		},
		{
			// filterRequest is flattened into plain query params, not JSON.
			name: "name filter is a flat param",
			opts: ListOptions{Page: 1, PageSize: 10, Name: "my-db"},
			want: "name=my-db&pageNumber=1&pageSize=10",
		},
		{
			name: "multiple statuses repeat the key",
			opts: ListOptions{Page: 1, PageSize: 10, Statuses: []string{"ACTIVE", "ERROR"}},
			want: "pageNumber=1&pageSize=10&status=ACTIVE&status=ERROR",
		},
		{
			name: "empty status entries are dropped",
			opts: ListOptions{Page: 1, PageSize: 10, Statuses: []string{"ACTIVE", ""}},
			want: "pageNumber=1&pageSize=10&status=ACTIVE",
		},
		{
			name: "empty name is omitted entirely",
			opts: ListOptions{Page: 2, PageSize: 10, Name: ""},
			want: "pageNumber=2&pageSize=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildListQuery(tt.opts).Encode(); got != tt.want {
				t.Errorf("BuildListQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}
