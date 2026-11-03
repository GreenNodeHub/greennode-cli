package vdbclient

import (
	"net/url"
	"strconv"
	"strings"
)

// DefaultPageSize matches the vserver list default so `grn` feels consistent
// across products.
const DefaultPageSize = 50

// ListOptions are the query parameters shared by every vDB list endpoint. The
// spec models Name/Statuses as a "filterRequest" object, but the API flattens it
// into plain query params — see BuildListQuery.
type ListOptions struct {
	Page     int // 1-based, as the user typed it
	PageSize int
	Name     string
	Statuses []string
}

// BuildListQuery renders ListOptions into vDB's query parameters.
//
// Two quirks are encoded here so no command has to remember them:
//
//   - pageNumber is 1-BASED (page 1 is the first page). This matches vserver but
//     is the opposite of vks, whose pages start at 0 — so --page passes straight
//     through with no offset.
//   - The spec's "filterRequest" object is sent as flat query params
//     (?name=x&status=ACTIVE), not JSON. Multiple statuses repeat the key.
//
// Verified against the live API: requesting pageNumber=1 and pageNumber=2
// returns distinct items and pageObject.number echoes the requested page, so the
// response counter is 1-based too and can be fed back in. Page against
// pageObject.totalPages / totalElements.
func BuildListQuery(o ListOptions) url.Values {
	page := o.Page
	if page < 1 {
		page = 1
	}
	pageSize := o.PageSize
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	v := url.Values{}
	v.Set("pageNumber", strconv.Itoa(page))
	v.Set("pageSize", strconv.Itoa(pageSize))

	if o.Name != "" {
		v.Set("name", o.Name)
	}
	for _, s := range o.Statuses {
		if s != "" {
			v.Add("status", s)
		}
	}

	return v
}

// Get issues a GET with multi-valued query parameters.
//
// GreennodeClient.Get takes a map[string]string, which holds one value per key
// and so cannot express vdb's repeated "?status=ACTIVE&status=ERROR". Encoding
// the query onto the path is equivalent — the client only rewrites the query
// string when its params argument is non-empty — and keeps this vdb-specific
// need out of the shared client.
func Get(c *Client, path string, values url.Values) (interface{}, error) {
	return c.Get(appendQuery(path, values), nil)
}

// appendQuery encodes values onto path, preserving any query string already on it.
func appendQuery(path string, values url.Values) string {
	if len(values) == 0 {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + values.Encode()
}
