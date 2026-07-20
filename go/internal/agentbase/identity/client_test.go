package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/auth"
	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test","token_type":"Bearer","expires_in":3600}`))
	}))
	apiSrv := httptest.NewServer(handler)
	t.Cleanup(func() {
		apiSrv.Close()
		tokenSrv.Close()
	})
	provider := auth.NewProvider("id", "secret", tokenSrv.URL)
	return NewClient(apiSrv.URL, provider), apiSrv
}

func sp(s string) *string { return &s }

func TestListAgentIdentities(t *testing.T) {
	resp := PagedResponseAgentIdentityResponse{
		Content: jsonslice.Array[AgentIdentityResponse]{{ID: sp("1"), Name: sp("agent-one")}},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListAgentIdentities(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Content) != 1 || out.Content[0].Name == nil || *out.Content[0].Name != "agent-one" {
		t.Errorf("unexpected content: %+v", out.Content)
	}
}

func TestCreateAgentIdentity(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentIdentityResponse{ID: sp("x"), Name: sp("new-agent")})
	})
	out, err := c.CreateAgentIdentity(context.Background(), &CreateAgentIdentityRequest{Name: "new-agent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "new-agent" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestGetAgentIdentity(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentIdentityResponse{ID: sp("1"), Name: sp("my-agent")})
	})
	out, err := c.GetAgentIdentity(context.Background(), "my-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "my-agent" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestDeleteAgentIdentity(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteAgentIdentity(context.Background(), "my-agent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateApikeyProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApikeyProviderResponse{ID: sp("1"), Name: sp("prov")})
	})
	out, err := c.CreateApikeyProvider(context.Background(), &CreateApikeyProviderRequest{Name: "prov", Apikey: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "prov" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestListApikeyProviders(t *testing.T) {
	resp := PagedResponseApikeyProviderResponse{
		Content: jsonslice.Array[ApikeyProviderResponse]{{ID: sp("1"), Name: sp("static-key")}},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListApikeyProviders(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Content) != 1 {
		t.Errorf("expected 1 result, got %v", out.Content)
	}
}

func TestUpdateApikeyProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.UpdateApikeyProvider(context.Background(), "prov", &UpdateApikeyProviderRequest{Apikey: "newkey"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteApikeyProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteApikeyProvider(context.Background(), "prov"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateOauth2Provider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Oauth2ProviderResponse{ID: sp("1"), Name: sp("google")})
	})
	req := &CreateOauth2ProviderRequest{Name: "google", ClientID: "cid", ClientSecret: "cs",
		AuthorizationURL: "https://auth", TokenURL: "https://token"}
	out, err := c.CreateOauth2Provider(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "google" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestListOauth2Providers(t *testing.T) {
	resp := PagedResponseOauth2ProviderResponse{
		Content: jsonslice.Array[Oauth2ProviderResponse]{{ID: sp("1"), Name: sp("google")}},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListOauth2Providers(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Content) != 1 {
		t.Errorf("expected 1 result, got %v", out.Content)
	}
}

func TestGetOauth2Provider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Oauth2ProviderResponse{ID: sp("1"), Name: sp("google")})
	})
	out, err := c.GetOauth2Provider(context.Background(), "google")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "google" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestDeleteOauth2Provider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteOauth2Provider(context.Background(), "google"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDelegatedProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DelegatedApiKeyProviderResponse{ID: sp("1"), Name: sp("del-prov")})
	})
	out, err := c.CreateDelegatedProvider(context.Background(), &CreateDelegatedApiKeyProviderRequest{Name: "del-prov"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "del-prov" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestListDelegatedProviders(t *testing.T) {
	resp := PagedResponseDelegatedApiKeyProviderResponse{
		Content: jsonslice.Array[DelegatedApiKeyProviderResponse]{{ID: sp("1"), Name: sp("dp")}},
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	out, err := c.ListDelegatedProviders(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Content) != 1 {
		t.Errorf("expected 1 result, got %v", out.Content)
	}
}

func TestGetApikeyForAgentIdentity(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApikeyResponse{Apikey: sp("secret-key")})
	})
	out, err := c.GetApikeyForAgentIdentity(context.Background(), "prov", "agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Apikey == nil || *out.Apikey != "secret-key" {
		t.Errorf("unexpected apikey: %v", out.Apikey)
	}
}

func TestGetM2MToken(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: sp("m2m-token")})
	})
	out, err := c.GetM2MToken(context.Background(), "prov", "agent", &GetM2mTokenRequest{Scopes: jsonslice.Array[string]{"read"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken == nil || *out.AccessToken != "m2m-token" {
		t.Errorf("unexpected token: %v", out.AccessToken)
	}
}

func TestClientError_Returns404(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	_, err := c.GetAgentIdentity(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestUpdateAgentIdentity(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AgentIdentityResponse{ID: sp("1"), Name: sp("updated")})
	})
	desc := "desc"
	out, err := c.UpdateAgentIdentity(context.Background(), "updated", &UpdateAgentIdentityRequest{Description: &desc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "updated" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestGetApikeyProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApikeyProviderResponse{ID: sp("1"), Name: sp("prov")})
	})
	out, err := c.GetApikeyProvider(context.Background(), "prov")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "prov" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestGetDelegatedProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DelegatedApiKeyProviderResponse{ID: sp("1"), Name: sp("dp")})
	})
	out, err := c.GetDelegatedProvider(context.Background(), "dp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name == nil || *out.Name != "dp" {
		t.Errorf("unexpected name: %v", out.Name)
	}
}

func TestDeleteDelegatedProvider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.DeleteDelegatedProvider(context.Background(), "dp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDelegatedApiKey(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GetDelegatedApiKeyResponse{Apikey: sp("delegated-key")})
	})
	req := &GetDelegatedApiKeyRequest{AgentUserID: "u1", ReturnURL: "https://ret"}
	out, err := c.GetDelegatedApiKey(context.Background(), "prov", "agent", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Apikey == nil || *out.Apikey != "delegated-key" {
		t.Errorf("unexpected apikey: %v", out.Apikey)
	}
}

func TestUpdateOauth2Provider(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := &UpdateOauth2ProviderRequest{ClientID: "cid", ClientSecret: "cs",
		AuthorizationURL: "https://auth", TokenURL: "https://token"}
	if err := c.UpdateOauth2Provider(context.Background(), "google", req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet3LOToken(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ThreeLoTokenResponse{AccessToken: sp("3lo-token")})
	})
	req := &ThreeLoTokenRequest{AgentUserID: "u1", ReturnURL: "https://ret", Scopes: jsonslice.Array[string]{"read"}}
	out, err := c.Get3LOToken(context.Background(), "prov", "agent", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken == nil || *out.AccessToken != "3lo-token" {
		t.Errorf("unexpected token: %v", out.AccessToken)
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com", nil)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}
