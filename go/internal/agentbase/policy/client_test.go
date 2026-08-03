package policy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
)

// fakeTokenProvider satisfies coreclient.TokenProvider so the policy tests do
// not spin up a real IAM token server. The Bearer seam is covered in
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

// TestListGroups verifies the hybrid envelope (content + page/pageSize/...)
// decodes and the query uses ?page=&page_size=&name=.
func TestListGroups(t *testing.T) {
	var gotPage, gotSize, gotName string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotPage = r.URL.Query().Get("page")
		gotSize = r.URL.Query().Get("page_size")
		gotName = r.URL.Query().Get("name")
		_ = json.NewEncoder(w).Encode(ListPolicyGroupsResponse{
			Content: []PolicyGroup{{ID: "policyengine_1", Name: "g", Description: "d"}},
			Page:    2, PageSize: 20, TotalPage: 3, TotalItem: 55,
		})
	})
	out, err := c.ListGroups(context.Background(), 2, 20, "g")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "2" || gotSize != "20" || gotName != "g" {
		t.Errorf("query page=%q page_size=%q name=%q", gotPage, gotSize, gotName)
	}
	if len(out.Content) != 1 || out.Content[0].Name != "g" {
		t.Errorf("unexpected content: %+v", out.Content)
	}
	if out.TotalItem != 55 || out.TotalPage != 3 {
		t.Errorf("paging mismatch: %+v", out)
	}
}

func TestListGroups_OmittedPagingNotSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Encode(); q != "" {
			t.Errorf("expected no query when paging/name omitted, got %q", q)
		}
		_ = json.NewEncoder(w).Encode(ListPolicyGroupsResponse{})
	})
	if _, err := c.ListGroups(context.Background(), 0, 0, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateGroup(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		var got CreatePolicyGroupRequest
		_ = json.Unmarshal(mustReadBody(r), &got)
		if got.Name != "g" || got.Description != "d" {
			t.Errorf("unexpected request: %+v", got)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(PolicyGroup{ID: "policyengine_1", Name: "g"})
	})
	out, err := c.CreateGroup(context.Background(), &CreatePolicyGroupRequest{Name: "g", Description: "d"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "policyengine_1" {
		t.Errorf("unexpected id: %s", out.ID)
	}
	if gotAuth != "Bearer test" {
		t.Errorf("Authorization=%q, want Bearer test", gotAuth)
	}
}

func TestGetGroup(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups/policyengine_1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(PolicyGroup{ID: "policyengine_1", Name: "g"})
	})
	out, err := c.GetGroup(context.Background(), "policyengine_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "g" {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestUpdateGroup(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var got UpdatePolicyGroupRequest
		_ = json.Unmarshal(mustReadBody(r), &got)
		if got.Description != "new" || got.Name != "" {
			t.Errorf("update should send only description: %+v", got)
		}
		_ = json.NewEncoder(w).Encode(PolicyGroup{ID: "policyengine_1", Description: "new"})
	})
	out, err := c.UpdateGroup(context.Background(), "policyengine_1", &UpdatePolicyGroupRequest{Description: "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Description != "new" {
		t.Errorf("unexpected: %+v", out)
	}
}

// TestDeleteGroup verifies DELETE returns 200 and the {message} body decodes.
func TestDeleteGroup(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(deleteMessage{Message: "deleted"})
	})
	msg, err := c.DeleteGroup(context.Background(), "policyengine_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "deleted" {
		t.Errorf("unexpected message: %q", msg)
	}
}

// TestCreatePolicy verifies the nested path and the statement (with a condition)
// round-trips.
func TestCreatePolicy(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups/policyengine_1/policies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var got CreatePolicyRequest
		if err := json.Unmarshal(mustReadBody(r), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Name != "p" || got.Statement.Effect != "permit" {
			t.Errorf("unexpected request: %+v", got)
		}
		if len(got.Statement.Actions) != 1 || got.Statement.Actions[0] != "InsuranceAPI__read" {
			t.Errorf("actions mismatch: %+v", got.Statement.Actions)
		}
		whens, ok := got.Statement.Condition["when"]
		if !ok {
			t.Fatalf("missing condition.when: %+v", got.Statement.Condition)
		}
		eq, _ := whens["equals"].(map[string]any)
		if eq["context.role"] != "admin" {
			t.Errorf("condition mismatch: %+v", whens)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Policy{ID: "policy_1", Name: "p", Active: true})
	})
	out, err := c.CreatePolicy(context.Background(), "policyengine_1", &CreatePolicyRequest{
		Name: "p",
		Statement: PolicyTemplate{
			Effect: "permit", Principal: "*",
			Actions:   []string{"InsuranceAPI__read"},
			Resources: []string{"gateway:*"},
			Condition: map[string]Clause{
				"when": {"equals": map[string]any{"context.role": "admin"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "policy_1" || !out.Active {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestListPolicies(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups/policyengine_1/policies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListPoliciesResponse{
			Content: []Policy{{ID: "policy_1", Name: "p"}},
			Page:    1, PageSize: 10, TotalPage: 1, TotalItem: 1,
		})
	})
	out, err := c.ListPolicies(context.Background(), "policyengine_1", 1, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Content) != 1 || out.Content[0].ID != "policy_1" {
		t.Errorf("unexpected: %+v", out.Content)
	}
}

func TestGetPolicy(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups/policyengine_1/policies/policy_1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Policy{ID: "policy_1", Name: "p", Active: true})
	})
	out, err := c.GetPolicy(context.Background(), "policyengine_1", "policy_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Active {
		t.Errorf("expected active")
	}
}

// TestUpdatePolicy verifies PUT with pointer merge-patch: setting Active=false
// (explicit) sends "active":false; a nil Statement is omitted.
func TestUpdatePolicy(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var raw map[string]any
		_ = json.Unmarshal(mustReadBody(r), &raw)
		if v, ok := raw["active"]; !ok || v != false {
			t.Errorf("active should be explicitly false, got %v", raw["active"])
		}
		if _, ok := raw["statement"]; ok {
			t.Errorf("statement should be omitted when nil, got %v", raw["statement"])
		}
		_ = json.NewEncoder(w).Encode(Policy{ID: "policy_1", Active: false})
	})
	f := false
	out, err := c.UpdatePolicy(context.Background(), "policyengine_1", "policy_1", &UpdatePolicyRequest{Active: &f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Active {
		t.Errorf("expected inactive")
	}
}

func TestDeletePolicy(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy-groups/policyengine_1/policies/policy_1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(deleteMessage{Message: "policy deleted"})
	})
	msg, err := c.DeletePolicy(context.Background(), "policyengine_1", "policy_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "policy deleted" {
		t.Errorf("unexpected message: %q", msg)
	}
}

func TestListConditionOperators(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/condition-operators" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListConditionOperatorsResponse{
			Operators: []ConditionOperator{
				{Name: "equals", Arity: "single", ValueTypes: []string{"string", "long", "bool"}},
				{Name: "in", Arity: "list"},
			},
		})
	})
	out, err := c.ListConditionOperators(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Operators) != 2 || out.Operators[0].Name != "equals" {
		t.Errorf("unexpected: %+v", out.Operators)
	}
}

// TestDecide verifies the internal decision route path, the JSON-RPC action body
// round-trip, and allow/deny decode (deny carries a Reason).
func TestDecide(t *testing.T) {
	var gotBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/internal/api/v1/gateways/gw-1/targets/svc-a/decisions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotBody = mustReadBody(r)
		_ = json.NewEncoder(w).Encode(DecisionResult{
			Allow:  false,
			Reason: &Reason{Code: "no_matching_permit", PolicyID: "policy_1", Message: "no permit matched"},
		})
	})
	out, err := c.Decide(context.Background(), "gw-1", "svc-a", &DecisionRequest{
		PolicyGroupID: "policyengine_1",
		User:          UserInput{ID: "u-1", Type: "jwt"},
		Action: JSONRPCAction{
			JSONRPC: "2.0", Method: "tools/call",
			Params: &JSONRPCParams{Name: "InsuranceAPI__read", Arguments: map[string]any{"id": "x"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Allow {
		t.Errorf("expected deny")
	}
	if out.Reason == nil || out.Reason.Code != "no_matching_permit" || out.Reason.PolicyID != "policy_1" {
		t.Errorf("unexpected reason: %+v", out.Reason)
	}
	var body DecisionRequest
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Action.Method != "tools/call" || body.Action.Params.Name != "InsuranceAPI__read" {
		t.Errorf("action mismatch: %+v", body.Action)
	}
	if body.User.Type != "jwt" {
		t.Errorf("user mismatch: %+v", body.User)
	}
}

func TestGetGroup_404ReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	_, err := c.GetGroup(context.Background(), "missing")
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

func mustReadBody(r *http.Request) []byte {
	b, _ := io.ReadAll(r.Body)
	return b
}
