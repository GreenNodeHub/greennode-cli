// Package policy is the typed client for the agent-core-policy service
// (`grn agentbase policy`). The model mirrors the service's REST contract:
// Cedar-backed authorization policy groups + policies (rules), a condition
// operator catalog, and a per-request decision endpoint.
//
// agent-core-policy is a Go/Gin service (unlike the Java runtime/memory
// siblings); only the wire contract matters here. Auth is the shared seam —
// Bearer → IAM ingress → `portal-user-id` header (the service is user-scoped
// on every endpoint, including the internal decision route).
//
// Wire shape notes (load-bearing — do not assume the runtime/memory shape):
//   - Paths are versioned: `/api/v1/policy-groups[/:id[/policies[/:id]]]`,
//     `/api/v1/policies/condition-operators`, and the internal decision route
//     `/internal/api/v1/gateways/:gateway/targets/:target/decisions`.
//   - Policies are NESTED under their group in the URL.
//   - List envelope is a HYBRID: `content` (like identity) plus
//     `page`/`pageSize`/`totalPage`/`totalItem` (like runtime/memory). Query
//     params are `page` (1-based) / `page_size` (1-100) / `name` (substring).
//   - Updates are PUT (full replace for groups; pointer-based merge-patch for
//     policies, where `active`/`statement` are `*`-nullable).
//   - Delete returns 200 with `{"message": "..."}` (not 204).
//   - `DecisionRequest.action` is a JSON-RPC 2.0 envelope (MCP-shaped).
//
// Compiled in ONLY with `-tags agentbase`.
package policy

import "time"

// ----------------------------------------------------------------------------
// PolicyTemplate (the "statement" of a policy) — compiled to Cedar at write time
// ----------------------------------------------------------------------------

// PolicyTemplate is the user-facing policy definition. The cedar converter
// compiles it to a Cedar statement on Create/Update (invalid → 400).
//
//   - Effect: "permit" | "forbid".
//   - Principal: a Cedar principal entity id used VERBATIM, e.g.
//     "jwt_user:abc-123", "iam_role:admin", or "*".
//   - Actions: action names matching ^[A-Za-z0-9][A-Za-z0-9_]*__[A-Za-z0-9][A-Za-z0-9_]*$
//     (e.g. "InsuranceAPI__update_coverage"), or ["*"].
//   - Resources: gateway refs, e.g. ["gateway:my-gw"] or ["gateway:*"]. (At
//     evaluation time the resource is always the gateway named in the URL.)
//   - Condition: optional branches keyed "when" and/or "unless". Each branch is
//     a Clause: operator name → {keyPath: value}. See ConditionOperator.Example.
type PolicyTemplate struct {
	Effect    string            `json:"effect"`
	Principal string            `json:"principal"`
	Actions   []string          `json:"actions"`
	Resources []string          `json:"resources"`
	Condition map[string]Clause `json:"condition,omitempty"`
}

// Clause is one condition branch body: operator name → {keyPath: value}. Value
// may be a scalar (string/number/bool) or, for the "in" operator, an array.
// keyPaths are dotted paths under context.*/principal.*/resource.*.
// Example: {"equals": {"context.role": "admin"}, "in": {"context.env": ["prod","stg"]}}.
type Clause map[string]any

// ----------------------------------------------------------------------------
// Policy group (the container)
// ----------------------------------------------------------------------------

// PolicyGroup is the top-level grouping of policies, scoped to a portal user.
// IDs are prefixed "policyengine_<uuid>". Quota: max 20 per user. Deleting a
// group cascade-deletes its policies.
type PolicyGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreatePolicyGroupRequest is the body for POST /api/v1/policy-groups. name is
// required (unique per user).
type CreatePolicyGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdatePolicyGroupRequest is the body for PUT /api/v1/policy-groups/:id. Both
// fields optional; only set fields are applied (PUT with omitempty).
type UpdatePolicyGroupRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ----------------------------------------------------------------------------
// Policy (a rule within a group)
// ----------------------------------------------------------------------------

// Policy is a single authorization rule within a PolicyGroup. IDs are prefixed
// "policy_<uuid>". Quota: max 10 per group. CedarStatement is server-only
// (json:"-") — the compiled statement is derived from Statement.
type Policy struct {
	ID            string         `json:"id"`
	PolicyGroupID string         `json:"policyGroupId"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Active        bool           `json:"active"`
	Statement     PolicyTemplate `json:"statement"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// CreatePolicyRequest is the body for POST .../policies. name + statement are
// required.
type CreatePolicyRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Statement   PolicyTemplate `json:"statement"`
	Active      bool           `json:"active"`
}

// UpdatePolicyRequest is the body for PUT .../policies/:id. All fields are
// pointers → true merge-patch semantics: omit a field to leave it unchanged,
// set `active`/`statement` to null to clear them.
type UpdatePolicyRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Active      *bool           `json:"active,omitempty"`
	Statement   *PolicyTemplate `json:"statement,omitempty"`
}

// ----------------------------------------------------------------------------
// List envelopes — HYBRID paging (content + page/pageSize/totalPage/totalItem)
// ----------------------------------------------------------------------------

// ListPolicyGroupsResponse is the body of GET /api/v1/policy-groups. The items
// live under `content` (not items/listData); paging is page/pageSize/totalPage/
// totalItem. Query: ?page=&page_size=&name= (1-based; page_size clamped [1,100]).
type ListPolicyGroupsResponse struct {
	Content   []PolicyGroup `json:"content"`
	Page      int           `json:"page"`
	PageSize  int           `json:"pageSize"`
	TotalPage int64         `json:"totalPage"`
	TotalItem int64         `json:"totalItem"`
}

// ListPoliciesResponse is the body of GET .../policies. Same hybrid envelope.
type ListPoliciesResponse struct {
	Content   []Policy `json:"content"`
	Page      int      `json:"page"`
	PageSize  int      `json:"pageSize"`
	TotalPage int64    `json:"totalPage"`
	TotalItem int64    `json:"totalItem"`
}

// ----------------------------------------------------------------------------
// Condition operator catalog (meta endpoint, unpaginated)
// ----------------------------------------------------------------------------

// ConditionOperator describes one accepted condition operator. Example is a
// ready-made JSON snippet for that operator. Arity is "single" or "list".
type ConditionOperator struct {
	Name               string   `json:"name"`
	DisplayName        string   `json:"displayName"`
	Description        string   `json:"description"`
	Arity              string   `json:"arity"`
	ValueTypes         []string `json:"valueTypes"`
	AcceptsKeyPrefixes []string `json:"acceptsKeyPrefixes"`
	Example            any      `json:"example"`
}

// ListConditionOperatorsResponse is the body of GET
// /api/v1/policies/condition-operators — {operators: [...]}, unpaginated.
type ListConditionOperatorsResponse struct {
	Operators []ConditionOperator `json:"operators"`
}

// ----------------------------------------------------------------------------
// Decision (the internal authorization endpoint)
// ----------------------------------------------------------------------------

// DecisionRequest is the body for POST
// /internal/api/v1/gateways/:gateway/targets/:target/decisions. GatewayName and
// GatewayTargetName come from the URL path (not the JSON body); the CLI client
// injects them as path segments, so they are not modeled here.
//
// Action is a JSON-RPC 2.0 envelope (MCP-shaped): the inbound tool call being
// authorized. The service mirrors action.params.arguments into the Cedar
// context as context.input.
type DecisionRequest struct {
	PolicyGroupID string         `json:"policyGroupId"`
	User          UserInput      `json:"user"`
	Principal     map[string]any `json:"principal,omitempty"`
	Action        JSONRPCAction  `json:"action"`
	Context       map[string]any `json:"context,omitempty"`
}

// UserInput identifies the end user being evaluated. Type is "iam" or "jwt"
// (case-insensitive; normalized server-side).
type UserInput struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// JSONRPCAction is the JSON-RPC 2.0 envelope (MCP tool-call shape).
type JSONRPCAction struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  *JSONRPCParams `json:"params,omitempty"`
	ID      any            `json:"id,omitempty"`
}

// JSONRPCParams holds the params object; Name is the effective action name
// (falls back to Method when unset). Arguments are mirrored into the Cedar
// context as context.input at evaluation time.
type JSONRPCParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// DecisionResult is the response. The endpoint ALWAYS returns 200; an allow has
// no Reason, a deny carries a Reason explaining why.
type DecisionResult struct {
	Allow  bool    `json:"allow"`
	Reason *Reason `json:"reason,omitempty"`
}

// Reason explains a deny decision.
type Reason struct {
	Code     string `json:"code"`
	PolicyID string `json:"policyId,omitempty"`
	Message  string `json:"message"`
}

// ----------------------------------------------------------------------------
// Delete response — 200 with {"message": "..."}
// ----------------------------------------------------------------------------

// deleteMessage is the 200 body returned by group/policy DELETE.
type deleteMessage struct {
	Message string `json:"message"`
}
