package identity

import (
	"encoding/json"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/jsonslice"
)

func TestCreateAgentIdentityRequestJSON(t *testing.T) {
	req := CreateAgentIdentityRequest{Name: "my-agent"}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if m["name"] != "my-agent" {
		t.Error("name field missing")
	}
}

func TestUpdateAgentIdentityRequestAllOptional(t *testing.T) {
	req := UpdateAgentIdentityRequest{}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["description"]; !ok {
		t.Error("expected key \"description\" to be present")
	} else if m["description"] != nil {
		t.Errorf("expected description null, got %v", m["description"])
	}
	if _, ok := m["allowedReturnUrls"]; !ok {
		t.Error("expected key \"allowedReturnUrls\" to be present")
	} else {
		ar, ok := m["allowedReturnUrls"].([]interface{})
		if !ok || len(ar) != 0 {
			t.Errorf("expected allowedReturnUrls empty array, got %v", m["allowedReturnUrls"])
		}
	}
}

func TestCreateOauth2ProviderRequestAllRequired(t *testing.T) {
	req := CreateOauth2ProviderRequest{
		Name:             "provider-1",
		ClientID:         "cid",
		ClientSecret:     "csecret",
		AuthorizationURL: "https://auth.example.com/authorize",
		TokenURL:         "https://auth.example.com/token",
	}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if m["name"] != "provider-1" || m["clientId"] != "cid" {
		t.Errorf("fields wrong: %v", m)
	}
	if m["authorizationUrl"] == nil || m["tokenUrl"] == nil {
		t.Error("authorizationUrl / tokenUrl fields missing")
	}
}

func TestGetM2mTokenRequestScopes(t *testing.T) {
	req := GetM2mTokenRequest{Scopes: jsonslice.Array[string]{"read", "write"}}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	scopes, _ := m["scopes"].([]interface{})
	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes))
	}
}

func TestThreeLoTokenRequestRequiredFields(t *testing.T) {
	req := ThreeLoTokenRequest{
		AgentUserID: "user-123",
		ReturnURL:   "https://callback.example.com",
		Scopes:      jsonslice.Array[string]{"openid"},
	}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if m["agentUserId"] != "user-123" {
		t.Error("agentUserId field missing")
	}
	if m["returnUrl"] != "https://callback.example.com" {
		t.Error("returnUrl field missing")
	}
}

func TestThreeLoTokenRequestOptionalFieldsNull(t *testing.T) {
	req := ThreeLoTokenRequest{AgentUserID: "u", ReturnURL: "https://cb", Scopes: jsonslice.Array[string]{"x"}}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"sessionId", "customParameters", "customState", "forceAuthentication"} {
		if _, ok := m[k]; !ok {
			t.Errorf("expected key %q to be present", k)
			continue
		}
		if m[k] != nil {
			t.Errorf("expected key %q to be null, got %v", k, m[k])
		}
	}
}

func TestCreateApikeyProviderRequestJSON(t *testing.T) {
	req := CreateApikeyProviderRequest{Name: "my-key", Apikey: "secret-value"}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if m["name"] != "my-key" || m["apikey"] != "secret-value" {
		t.Errorf("fields wrong: %v", m)
	}
}

func TestGetDelegatedApiKeyRequestRequiredFields(t *testing.T) {
	req := GetDelegatedApiKeyRequest{
		AgentUserID: "u1",
		ReturnURL:   "https://return.example.com",
	}
	b, _ := json.Marshal(req)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if m["agentUserId"] != "u1" || m["returnUrl"] != "https://return.example.com" {
		t.Errorf("required fields missing: %v", m)
	}
}

func TestPagedResponseUnmarshal(t *testing.T) {
	payload := `{
		"content": [{"id":"1","name":"agent-1"}],
		"page": 0,
		"size": 20,
		"totalElements": 1,
		"totalPages": 1,
		"first": true,
		"last": true,
		"hasNext": false,
		"hasPrevious": false
	}`
	var resp PagedResponseAgentIdentityResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Content) != 1 || resp.Content[0].Name == nil || *resp.Content[0].Name != "agent-1" {
		t.Errorf("unexpected response content: %+v", resp.Content)
	}
	if resp.TotalElements == nil || *resp.TotalElements != 1 {
		t.Errorf("expected totalElements=1, got %v", resp.TotalElements)
	}
}
