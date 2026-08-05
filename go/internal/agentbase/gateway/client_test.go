package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
)

// fakeTokenProvider satisfies coreclient.TokenProvider so the gateway tests do
// not spin up a real IAM token server. The Bearer seam itself is covered in
// internal/agentbase/client.
type fakeTokenProvider struct{ token string }

func (f *fakeTokenProvider) GetToken() (string, error)     { return f.token, nil }
func (f *fakeTokenProvider) RefreshToken() (string, error) { return f.token, nil }

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, &fakeTokenProvider{token: "test"})
}

// TestList verifies the {items, pagination} envelope decodes and that paging is
// 1-based with the pageSize query param (the key difference from identity's
// 0-based page + size).
func TestList(t *testing.T) {
	var gotPage, gotPageSize string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/gateways" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotPage = r.URL.Query().Get("page")
		gotPageSize = r.URL.Query().Get("pageSize")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListGatewaysResponse{
			Items:      []GatewayResponse{{ID: "1", Name: "gw-one", NetworkMode: "PUBLIC", State: "ACTIVE"}},
			Pagination: Pagination{Page: 1, PageSize: 50, TotalItems: 1},
		})
	})
	out, err := c.List(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "1" || gotPageSize != "50" {
		t.Errorf("query page=%q pageSize=%q, want 1/50", gotPage, gotPageSize)
	}
	if len(out.Items) != 1 || out.Items[0].Name != "gw-one" {
		t.Errorf("unexpected items: %+v", out.Items)
	}
	if out.Pagination.TotalItems != 1 {
		t.Errorf("unexpected pagination: %+v", out.Pagination)
	}
}

// TestList_OmittedPagingNotSent: page/pageSize <= 0 must be omitted so the
// server applies its own defaults rather than receiving page=0.
func TestList_OmittedPagingNotSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query when paging omitted, got %q", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListGatewaysResponse{})
	})
	if _, err := c.List(context.Background(), 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/gateways/my-gw" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GatewayResponse{ID: "1", Name: "my-gw", State: "ACTIVE"})
	})
	out, err := c.Get(context.Background(), "my-gw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "my-gw" || out.State != "ACTIVE" {
		t.Errorf("unexpected gateway: %+v", out)
	}
}

// TestCreate verifies the POST body round-trips (nested targets/outboundAuth)
// and the Authorization header carries the token.
func TestCreate(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var got CreateGatewayRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got.Name != "gw" || got.NetworkMode != "PUBLIC" || got.Replicas != 2 {
			t.Errorf("unexpected request: %+v", got)
		}
		if len(got.Targets) != 1 || got.Targets[0].OutboundAuth.Type != "APIKEY" {
			t.Errorf("unexpected targets: %+v", got.Targets)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GatewayResponse{ID: "1", Name: "gw", State: "WAITING_CREATING"})
	})
	ps := "CUSTOM"
	out, err := c.Create(context.Background(), &CreateGatewayRequest{
		Name: "gw", NetworkMode: "PUBLIC", FlavorID: "f1", Replicas: 2,
		InboundAuth: InboundAuthRequest{Mode: "NONE"},
		Targets: []CreateTargetInput{{
			Name: "t", Type: "MCP", Endpoint: "https://mcp",
			OutboundAuth: OutboundAuthRequest{Type: "APIKEY", ProviderSource: &ps, ProviderName: "p"},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.State != "WAITING_CREATING" {
		t.Errorf("unexpected state: %s", out.State)
	}
	if gotAuth != "Bearer test" {
		t.Errorf("Authorization=%q, want Bearer test", gotAuth)
	}
}

// TestUpdate verifies the merge-patch map is PATCHed verbatim, including a nil
// value that must serialize as JSON null (the policy-group clear signal).
func TestUpdate(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/gateways/my-gw" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GatewayResponse{ID: "1", Name: "my-gw", State: "UPDATING"})
	})
	out, err := c.Update(context.Background(), "my-gw", map[string]interface{}{
		"displayName":   "new name",
		"policyGroupId": nil, // clear
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.State != "UPDATING" {
		t.Errorf("unexpected state: %s", out.State)
	}
	// The patch must carry policyGroupId:null, not omit it.
	var got map[string]interface{}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode patch body: %v", err)
	}
	if v, ok := got["policyGroupId"]; !ok || v != nil {
		t.Errorf("policyGroupId=%v ok=%v, want null key present", v, ok)
	}
	if got["displayName"] != "new name" {
		t.Errorf("displayName=%v, want 'new name'", got["displayName"])
	}
}

func TestDelete(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	if err := c.Delete(context.Background(), "my-gw"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGet_404ReturnsAPIError verifies a non-2xx surfaces as *client.APIError
// (carrying status + body) rather than a silent zero value.
func TestGet_404ReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	_, err := c.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d, want 404", apiErr.StatusCode)
	}
}

func TestNewClient(t *testing.T) {
	if c := NewClient("https://example.com", nil); c == nil {
		t.Fatal("expected non-nil client")
	}
}

// ----------------------------------------------------------------------------
// Slice 5: sub-resources (flavors / access-logs / inbound-auth / private-network)
// ----------------------------------------------------------------------------

func wantMethodPath(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("method: got %s, want %s", r.Method, method)
	}
	if r.URL.Path != path {
		t.Errorf("path: got %s, want %s", r.URL.Path, path)
	}
}

func TestListFlavors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/api/v1/flavors")
		if got := r.URL.Query().Get("resourceType"); got != "GATEWAY" {
			t.Errorf("resourceType: %q", got)
		}
		if got := r.URL.Query().Get("networkMode"); got != "PRIVATE" {
			t.Errorf("networkMode: %q", got)
		}
		if got := r.URL.Query().Get("zoneId"); got != "z1" {
			t.Errorf("zoneId: %q", got)
		}
		_ = json.NewEncoder(w).Encode(FlavorListResponse{
			Items: []FlavorResponse{{ID: "f1", DisplayName: "small", CPU: 2, MemoryGi: 4, SortOrder: 1}},
		})
	})
	out, err := c.ListFlavors(context.Background(), "GATEWAY", "PRIVATE", "z1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 || out.Items[0].ID != "f1" || out.Items[0].MemoryGi != 4 {
		t.Errorf("decode: %+v", out)
	}
}

func TestListFlavors_OmitsEmptyFilters(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query, got %q", q)
		}
		_ = json.NewEncoder(w).Encode(FlavorListResponse{})
	})
	if _, err := c.ListFlavors(context.Background(), "", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestListAccessLogs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/api/v1/gateways/gw/access-logs")
		if got := r.URL.Query().Get("from"); got != "t0" {
			t.Errorf("from: %q", got)
		}
		if got := r.URL.Query().Get("httpStatus"); got != "500" {
			t.Errorf("httpStatus: %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page: %q", got)
		}
		if got := r.URL.Query().Get("interval"); got != "" {
			t.Errorf("interval should be omitted on list, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(AccessLogListResponse{
			Items:      []AccessLogEntry{{TargetName: "t1", DurationMs: 12, Timestamp: "now"}},
			Pagination: AccessLogPagination{Page: 2, PageSize: 10, Total: 1},
		})
	})
	out, err := c.ListAccessLogs(context.Background(), "gw", AccessLogQuery{
		From: "t0", HTTPStatus: "500", Page: 2, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 || out.Items[0].DurationMs != 12 || out.Pagination.Total != 1 {
		t.Errorf("decode: %+v", out)
	}
}

func TestAccessLogStats(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/api/v1/gateways/gw/access-logs/stats")
		if got := r.URL.Query().Get("interval"); got != "1h" {
			t.Errorf("interval: %q", got)
		}
		if got := r.URL.Query().Get("topN"); got != "5" {
			t.Errorf("topN: %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "" {
			t.Errorf("page should be omitted on stats, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(AccessLogStatsResponse{
			TotalRequests: 10, SuccessRate: 0.8, ErrorRate: 0.2,
		})
	})
	out, err := c.AccessLogStats(context.Background(), "gw", AccessLogQuery{
		Interval: "1h", TopN: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalRequests != 10 || out.SuccessRate != 0.8 {
		t.Errorf("decode: %+v", out)
	}
}

func TestPutIdpApp(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPut, "/api/v1/gateways/gw/inbound-auth/jwt/idp-app")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	secret := "s3cret"
	err := c.PutIdpApp(context.Background(), "gw", &PutIdpAppRequest{
		ClientID: "cid", ClientSecret: &secret, Scopes: []string{"openid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got PutIdpAppRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "cid" || *got.ClientSecret != "s3cret" || len(got.Scopes) != 1 {
		t.Errorf("body: %+v", got)
	}
}

func TestPutIdpApp_OmitsNilSecret(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.PutIdpApp(context.Background(), "gw", &PutIdpAppRequest{ClientID: "cid"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatal(err)
	}
	if _, present := got["clientSecret"]; present {
		t.Error("clientSecret should be omitted when nil")
	}
}

func TestClearIdpApp(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodDelete, "/api/v1/gateways/gw/inbound-auth/jwt/idp-app")
		if r.ContentLength > 0 {
			t.Errorf("expected no body, got %d bytes", r.ContentLength)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.ClearIdpApp(context.Background(), "gw"); err != nil {
		t.Fatal(err)
	}
}

func TestGetPrivateRoutes(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodGet, "/api/v1/gateways/gw/private-network/routes")
		_ = json.NewEncoder(w).Encode(PrivateRoutesResponse{Routes: []string{"10.0.0.0/16"}})
	})
	out, err := c.GetPrivateRoutes(context.Background(), "gw")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Routes) != 1 || out.Routes[0] != "10.0.0.0/16" {
		t.Errorf("decode: %+v", out)
	}
}

// TestGetPrivateRoutes_PublicMode404 verifies a PUBLIC-mode gateway 404s and the
// error surfaces as *client.APIError (so the command can render it as-is).
func TestGetPrivateRoutes_PublicMode404(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"private_network_not_applicable"}`, http.StatusNotFound)
	})
	_, err := c.GetPrivateRoutes(context.Background(), "gw")
	if err == nil {
		t.Fatal("expected 404 error")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected *client.APIError 404, got %T: %v", err, err)
	}
}

func TestReplacePrivateRoutes_IfMatch(t *testing.T) {
	var gotIfMatch string
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPut, "/api/v1/gateways/gw/private-network/routes")
		gotIfMatch = r.Header.Get("If-Match")
		gotBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(PrivateRoutesResponse{Routes: []string{"10.0.0.0/16"}})
	})
	out, err := c.ReplacePrivateRoutes(context.Background(), "gw", "v3", &ReplacePrivateRoutesRequest{
		Routes: []string{"10.0.0.0/16"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotIfMatch != "v3" {
		t.Errorf("If-Match: %q, want v3", gotIfMatch)
	}
	var got ReplacePrivateRoutesRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Routes) != 1 || got.Routes[0] != "10.0.0.0/16" {
		t.Errorf("body: %+v", got)
	}
	if len(out.Routes) != 1 {
		t.Errorf("decode: %+v", out)
	}
}

func TestReplacePrivateRoutes_NoIfMatch(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-Match"); got != "" {
			t.Errorf("If-Match should be absent, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(PrivateRoutesResponse{})
	})
	if _, err := c.ReplacePrivateRoutes(context.Background(), "gw", "", &ReplacePrivateRoutesRequest{Routes: []string{}}); err != nil {
		t.Fatal(err)
	}
}

func TestRepairServiceAccount(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantMethodPath(t, r, http.MethodPost, "/api/v1/gateways/gw/service-account/repair")
		if r.ContentLength > 0 {
			t.Errorf("expected no body, got %d bytes", r.ContentLength)
		}
		_ = json.NewEncoder(w).Encode(GatewayResponse{ID: "1", Name: "gw", State: "ACTIVE"})
	})
	out, err := c.RepairServiceAccount(context.Background(), "gw")
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "gw" || out.State != "ACTIVE" {
		t.Errorf("decode: %+v", out)
	}
}
