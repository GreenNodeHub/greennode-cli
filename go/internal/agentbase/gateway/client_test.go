package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/client"
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
			Items: []GatewayResponse{{ID: "1", Name: "gw-one", NetworkMode: "PUBLIC", State: "ACTIVE"}},
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
