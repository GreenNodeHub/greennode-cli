package policy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/agentbase/client"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
)

// Client is the API client for the agent-core-policy service. It wraps the
// shared base client (same auth seam as identity/gateway/runtime/memory).
type Client struct {
	http *client.Client
}

// NewClient creates a policy Client backed by the shared
// coreclient.TokenProvider. The policy service authenticates inbound via the
// upstream IAM ingress (Bearer → portal-user-id header) on every route —
// including the internal decision endpoint — so the same provider the rest of
// agentbase uses works here unchanged. Pass nil only in construction tests
// that never issue a request.
func NewClient(baseURL string, tp coreclient.TokenProvider) *Client {
	return &Client{http: client.New(baseURL, tp)}
}

// listQuery builds the ?page=&page_size=&name= query. page/size default to 1/10
// server-side when omitted (sent only when > 0); name is sent only when non-empty.
// Note the snake_case page_size query key (the response field is camelCase pageSize).
func listQuery(page, size int, name string) url.Values {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("page_size", strconv.Itoa(size))
	}
	if name != "" {
		q.Set("name", name)
	}
	return q
}

// ----------------------------------------------------------------------------
// Policy group
// ----------------------------------------------------------------------------

// List returns a page of policy groups for the authenticated user.
func (c *Client) ListGroups(ctx context.Context, page, size int, name string) (*ListPolicyGroupsResponse, error) {
	var out ListPolicyGroupsResponse
	if err := c.http.Get(ctx, "/api/v1/policy-groups", listQuery(page, size, name), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateGroup creates a policy group (max 20 per user).
func (c *Client) CreateGroup(ctx context.Context, req *CreatePolicyGroupRequest) (*PolicyGroup, error) {
	var out PolicyGroup
	if err := c.http.Post(ctx, "/api/v1/policy-groups", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetGroup retrieves a policy group by id.
func (c *Client) GetGroup(ctx context.Context, id string) (*PolicyGroup, error) {
	var out PolicyGroup
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/policy-groups/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateGroup applies a PUT (only set fields are applied via omitempty).
func (c *Client) UpdateGroup(ctx context.Context, id string, req *UpdatePolicyGroupRequest) (*PolicyGroup, error) {
	var out PolicyGroup
	if err := c.http.Put(ctx, fmt.Sprintf("/api/v1/policy-groups/%s", id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteGroup deletes a group and cascades to its policies. Returns the
// server's localized confirmation message.
func (c *Client) DeleteGroup(ctx context.Context, id string) (string, error) {
	var msg deleteMessage
	if err := c.http.Delete(ctx, fmt.Sprintf("/api/v1/policy-groups/%s", id), &msg); err != nil {
		return "", err
	}
	return msg.Message, nil
}

// ----------------------------------------------------------------------------
// Policy (rule within a group)
// ----------------------------------------------------------------------------

// ListPolicies returns a page of policies within a group.
func (c *Client) ListPolicies(ctx context.Context, groupID string, page, size int, name string) (*ListPoliciesResponse, error) {
	var out ListPoliciesResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/policy-groups/%s/policies", groupID), listQuery(page, size, name), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatePolicy creates a policy within a group (max 10 per group).
func (c *Client) CreatePolicy(ctx context.Context, groupID string, req *CreatePolicyRequest) (*Policy, error) {
	var out Policy
	if err := c.http.Post(ctx, fmt.Sprintf("/api/v1/policy-groups/%s/policies", groupID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPolicy retrieves a policy by id within a group.
func (c *Client) GetPolicy(ctx context.Context, groupID, id string) (*Policy, error) {
	var out Policy
	if err := c.http.Get(ctx, fmt.Sprintf("/api/v1/policy-groups/%s/policies/%s", groupID, id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePolicy applies a PUT with pointer-based merge-patch semantics (omit a
// field to leave it unchanged; null active/statement to clear).
func (c *Client) UpdatePolicy(ctx context.Context, groupID, id string, req *UpdatePolicyRequest) (*Policy, error) {
	var out Policy
	if err := c.http.Put(ctx, fmt.Sprintf("/api/v1/policy-groups/%s/policies/%s", groupID, id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletePolicy deletes a policy. Returns the server's localized confirmation
// message.
func (c *Client) DeletePolicy(ctx context.Context, groupID, id string) (string, error) {
	var msg deleteMessage
	if err := c.http.Delete(ctx, fmt.Sprintf("/api/v1/policy-groups/%s/policies/%s", groupID, id), &msg); err != nil {
		return "", err
	}
	return msg.Message, nil
}

// ----------------------------------------------------------------------------
// Condition operator catalog
// ----------------------------------------------------------------------------

// ListConditionOperators returns the accepted condition operators (unpaginated).
func (c *Client) ListConditionOperators(ctx context.Context) (*ListConditionOperatorsResponse, error) {
	var out ListConditionOperatorsResponse
	if err := c.http.Get(ctx, "/api/v1/policies/condition-operators", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ----------------------------------------------------------------------------
// Decision (internal authorization endpoint)
// ----------------------------------------------------------------------------

// Decide asks for an allow/deny decision for an inbound request against a
// gateway target. The endpoint ALWAYS returns 200 with {allow, reason?}.
func (c *Client) Decide(ctx context.Context, gatewayName, targetName string, req *DecisionRequest) (*DecisionResult, error) {
	var out DecisionResult
	path := fmt.Sprintf("/internal/api/v1/gateways/%s/targets/%s/decisions", gatewayName, targetName)
	if err := c.http.Post(ctx, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
