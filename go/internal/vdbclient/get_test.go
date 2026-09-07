package vdbclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetSendsRepeatedQueryKeys(t *testing.T) {
	// vdb filters by repeating the status key; the client's map-based Get holds
	// one value per key and cannot express that.
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":[]}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)

	q := BuildListQuery(ListOptions{Page: 2, PageSize: 10, Name: "db", Statuses: []string{"ACTIVE", "ERROR"}})
	if _, err := Get(c, "/vdb-relational/v1/database-instances", q); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if got := gotQuery["status"]; len(got) != 2 || got[0] != "ACTIVE" || got[1] != "ERROR" {
		t.Errorf("status = %v, want [ACTIVE ERROR]", got)
	}
	if got := gotQuery.Get("pageNumber"); got != "2" {
		t.Errorf("pageNumber = %q, want 2", got)
	}
	if got := gotQuery.Get("pageSize"); got != "10" {
		t.Errorf("pageSize = %q, want 10", got)
	}
	if got := gotQuery.Get("name"); got != "db" {
		t.Errorf("name = %q, want db", got)
	}
}

func TestAppendQuery(t *testing.T) {
	v := url.Values{}
	v.Set("b", "2")

	if got := appendQuery("/p?a=1", v); got != "/p?a=1&b=2" {
		t.Errorf("with existing query = %q, want /p?a=1&b=2", got)
	}
	if got := appendQuery("/p", v); got != "/p?b=2" {
		t.Errorf("without query = %q, want /p?b=2", got)
	}
	if got := appendQuery("/p", url.Values{}); got != "/p" {
		t.Errorf("with no values = %q, want /p", got)
	}
}
