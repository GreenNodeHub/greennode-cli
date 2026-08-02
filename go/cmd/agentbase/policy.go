package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/cliinput"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
	policypkg "github.com/vngcloud/greennode-cli/internal/agentbase/policy"
)

// policyCmd groups the agent-core-policy commands: Cedar-backed authorization
// policy groups + policies (rules), a condition-operator catalog, and a
// per-request decision endpoint. The agentbase /policy endpoint fronts
// agent-core-policy's /api/v1 (CRUD) and /internal/api/v1 (decisions).
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage authorization policy groups, policies, and decisions",
	Long: `Manage Cedar authorization policies (the agent-core-policy service).

A "policy group" is a container of policies owned by a user (max 20/user). Each
group holds "policies" — individual permit/forbid rules (max 10/group) compiled
to Cedar. A gateway binds a group via its policyGroupId; the gateway's policy
enforcement then asks agent-core-policy for an allow/denied decision per inbound
request.

The decision route is internal (called by agent-core-gateway), but is exposed
here as a probe:

    grn agentbase policy group create --name my-group
    grn agentbase policy group policy create <group-id> --file policy.yaml
    grn agentbase policy decide <gateway> <target> --policy-group-id <id> --method tools/call

Policy resources are synchronous (no WAITING_* FSM), so there is no 'wait'.
Policy shares the ~/.greennode profile like the rest of agentbase.`,
}

// newPolicyClient mirrors newMemoryClient: resolve the shared profile + env,
// select the shared token provider, force-mint once so auth failures surface
// before the first call, and point the typed client at the policy endpoint.
func newPolicyClient(ctx context.Context, cmd *cobra.Command) (*policypkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return policypkg.NewClient(ab.endpoints.Policy, provider), nil
}

// ---------------------------------------------------------------------------
// policy group
// ---------------------------------------------------------------------------

var policyGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage policy groups",
	Long: `Manage policy groups (the policy-engine containers).

A group owns a set of policies and is what a gateway binds via its
policyGroupId. Max 20 groups per user; deleting a group cascades to its
policies.`,
}

var policyGroupFile string

var policyGroupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a policy group",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if policyGroupFile != "" {
			data, err := os.ReadFile(policyGroupFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err := loadPolicyGroupSpec(data)
			if err != nil {
				return err
			}
			return createPolicyGroupAndPrint(ctx, cmd, req)
		}
		f := cmd.Flags()
		name, _ := f.GetString("name")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Group name (required, unique per user)")
		if err != nil {
			return err
		}
		description, _ := f.GetString("description")
		return createPolicyGroupAndPrint(ctx, cmd, &policypkg.CreatePolicyGroupRequest{
			Name: name, Description: description,
		})
	},
}

func createPolicyGroupAndPrint(ctx context.Context, cmd *cobra.Command, req *policypkg.CreatePolicyGroupRequest) error {
	client, err := newPolicyClient(ctx, cmd)
	if err != nil {
		return err
	}
	g, err := client.CreateGroup(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Policy group %q created (id %s).\n", g.Name, g.ID)
	return output.PrintResource(g, func() string { return g.ID }, func() error { return renderPolicyGroupDetail(g) })
}

var policyGroupGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print a policy-group create template (YAML or JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			b, err := json.MarshalIndent(&policypkg.CreatePolicyGroupRequest{
				Name: "my-group", Description: "A group of authorization policies",
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(policyGroupCreateTemplateYAML)
		return nil
	},
}

const policyGroupCreateTemplateYAML = `# Policy group create spec.
# Fill in and apply with:  grn agentbase policy group create --file <this-file>
#
# Required: name (unique per user). Max 20 groups per user.
name: my-group
description: "A group of authorization policies"
`

var policyGroupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List policy groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		page, _ := f.GetInt("page")
		size, _ := f.GetInt("size")
		name, _ := f.GetString("name")
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListGroups(ctx, page, size, name)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			output.JSON(resp)
			return nil
		case output.FormatID:
			for _, g := range resp.Content {
				output.PrintID(g.ID)
			}
			return nil
		}
		if len(resp.Content) == 0 {
			fmt.Fprintln(os.Stderr, "No policy groups found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Content))
		for i := range resp.Content {
			g := resp.Content[i]
			rows = append(rows, []string{g.ID, g.Name, output.StrOrDash(g.Description), formatTimeVal(g.CreatedAt)})
		}
		output.Table([]string{"ID", "Name", "Description", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

var policyGroupGetCmd = &cobra.Command{
	Use:   "get <group-id>",
	Short: "Show a policy group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		g, err := client.GetGroup(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(g, func() string { return g.ID }, func() error { return renderPolicyGroupDetail(g) })
	},
}

var policyGroupUpdateCmd = &cobra.Command{
	Use:   "update <group-id>",
	Short: "Update a policy group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		req := &policypkg.UpdatePolicyGroupRequest{}
		if policyGroupFile != "" {
			data, err := os.ReadFile(policyGroupFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			createReq, err := loadPolicyGroupSpec(data)
			if err != nil {
				return err
			}
			req.Name = createReq.Name
			req.Description = createReq.Description
		} else {
			f := cmd.Flags()
			// Only send fields the caller explicitly set (PUT + omitempty merge).
			if f.Changed("name") {
				req.Name, _ = f.GetString("name")
			}
			if f.Changed("description") {
				req.Description, _ = f.GetString("description")
			}
		}
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		g, err := client.UpdateGroup(ctx, args[0], req)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Policy group %q updated.\n", g.Name)
		return output.PrintResource(g, func() string { return g.ID }, func() error { return renderPolicyGroupDetail(g) })
	},
}

var policyGroupDeleteCmd = &cobra.Command{
	Use:   "delete <group-id>",
	Short: "Delete a policy group (cascades to its policies)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		msg, err := client.DeleteGroup(ctx, args[0])
		if err != nil {
			return err
		}
		if msg != "" {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "Policy group %q deleted.\n", args[0])
		}
		output.PrintDeletedID(args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// policy (rule within a group) — nested under `group`
// ---------------------------------------------------------------------------

var policyPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policies within a group",
	Long: `Manage policies (authorization rules) within a policy group.

Each policy is a permit/forbid rule compiled to Cedar. Max 10 policies per
group. The group-id is the first positional argument of each subcommand (it
maps to /policy-groups/<group-id>/policies).`,
}

var policyFile string

var policyCreateCmd = &cobra.Command{
	Use:   "create <group-id>",
	Short: "Create a policy within a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		groupID := args[0]
		req, err := buildPolicyCreateReq(cmd)
		if err != nil {
			return err
		}
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		p, err := client.CreatePolicy(ctx, groupID, req)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Policy %q created (id %s, active=%t).\n", p.Name, p.ID, p.Active)
		return output.PrintResource(p, func() string { return p.ID }, func() error { return renderPolicyDetail(p) })
	},
}

// buildPolicyCreateReq resolves the create request from --file (authoritative)
// or the simple flag path (name + a statement built from effect/principal/
// actions/resources). Conditions are only expressible via --file.
func buildPolicyCreateReq(cmd *cobra.Command) (*policypkg.CreatePolicyRequest, error) {
	if policyFile != "" {
		data, err := os.ReadFile(policyFile)
		if err != nil {
			return nil, fmt.Errorf("read --file: %w", err)
		}
		return loadPolicySpec(data)
	}
	f := cmd.Flags()
	name, _ := f.GetString("name")
	name, err := cliinput.RequireOrPromptString(name, "--name", "Policy name (required)")
	if err != nil {
		return nil, err
	}
	effect, _ := f.GetString("effect")
	effect, err = cliinput.RequireOrPromptString(effect, "--effect", "Effect (permit|forbid)")
	if err != nil {
		return nil, err
	}
	principal, _ := f.GetString("principal")
	principal, err = cliinput.RequireOrPromptString(principal, "--principal", "Principal (e.g. jwt_user:abc-123, iam_role:admin, or *)")
	if err != nil {
		return nil, err
	}
	actions, _ := f.GetStringArray("action")
	if len(actions) == 0 {
		var actionName string
		actionName, err = cliinput.RequireOrPromptString("", "--action", "Action (e.g. InsuranceAPI__read, or *)")
		if err != nil {
			return nil, err
		}
		actions = []string{actionName}
	}
	resources, _ := f.GetStringArray("resource")
	description, _ := f.GetString("description")
	active, _ := f.GetBool("active")
	return &policypkg.CreatePolicyRequest{
		Name:        name,
		Description: description,
		Active:      active,
		Statement: policypkg.PolicyTemplate{
			Effect:    effect,
			Principal: principal,
			Actions:   actions,
			Resources: resources,
		},
	}, nil
}

var policyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print a policy create template (YAML or JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			example := &policypkg.CreatePolicyRequest{
				Name: "allow-admin-read",
				Statement: policypkg.PolicyTemplate{
					Effect: "permit", Principal: "jwt_role:admin",
					Actions:   []string{"InsuranceAPI__read"},
					Resources: []string{"gateway:*"},
				},
				Active: true,
			}
			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(policyCreateTemplateYAML)
		return nil
	},
}

// policyCreateTemplateYAML is a hand-written, commented skeleton of
// CreatePolicyRequest with a full statement (including a condition example).
// Keys are the JSON (camelCase) field names so the file round-trips through
// 'create --file' exactly.
const policyCreateTemplateYAML = `# Policy create spec (a rule within a policy group).
# Apply with:  grn agentbase policy group policy create <group-id> --file <this-file>
#
# Required: name, statement.{effect, principal, actions}. Max 10 policies/group.
name: allow-admin-read
description: "Permit admin role to read"
active: true
statement:
  effect: permit                       # permit | forbid
  principal: "jwt_role:admin"          # Cedar principal entity id, verbatim (or *)
  actions:                             # ^[A-Za-z0-9_]+__[A-Za-z0-9_]+$ , or ["*"]
    - InsuranceAPI__read
  resources:                           # gateway refs, e.g. gateway:<name> or gateway:*
    - "gateway:*"
  # condition is optional. Branches: when / unless (both may be present).
  # Each branch maps an operator -> {keyPath: value}. Operators: equals,
  # notEquals, lessThan, lessThanOrEqual, greaterThan, greaterThanOrEqual,
  # like, contains, in. keyPaths: context.* / principal.* / resource.*
  # condition:
  #   when:
  #     equals: {context.role: admin}
  #     in: {context.env: [prod, stg]}
  #   unless:
  #     lessThan: {context.hour: 9}
`

var policyListCmd = &cobra.Command{
	Use:   "list <group-id>",
	Short: "List policies within a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		page, _ := f.GetInt("page")
		size, _ := f.GetInt("size")
		name, _ := f.GetString("name")
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListPolicies(ctx, args[0], page, size, name)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			output.JSON(resp)
			return nil
		case output.FormatID:
			for _, p := range resp.Content {
				output.PrintID(p.ID)
			}
			return nil
		}
		if len(resp.Content) == 0 {
			fmt.Fprintln(os.Stderr, "No policies found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Content))
		for i := range resp.Content {
			p := resp.Content[i]
			rows = append(rows, []string{p.ID, p.Name, activeStr(p.Active), p.Statement.Effect, p.Statement.Principal, formatTimeVal(p.CreatedAt)})
		}
		output.Table([]string{"ID", "Name", "Active", "Effect", "Principal", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

var policyGetCmd = &cobra.Command{
	Use:   "get <group-id> <policy-id>",
	Short: "Show a policy",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		p, err := client.GetPolicy(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		return output.PrintResource(p, func() string { return p.ID }, func() error { return renderPolicyDetail(p) })
	},
}

var policyUpdateCmd = &cobra.Command{
	Use:   "update <group-id> <policy-id>",
	Short: "Update a policy (merge-patch)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		req := &policypkg.UpdatePolicyRequest{}
		if policyFile != "" {
			data, err := os.ReadFile(policyFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			createReq, err := loadPolicySpec(data)
			if err != nil {
				return err
			}
			// --file is a full replacement: send name/description/statement/active.
			req.Name = createReq.Name
			req.Description = createReq.Description
			req.Statement = &createReq.Statement
			req.Active = &createReq.Active
		} else {
			f := cmd.Flags()
			if f.Changed("name") {
				req.Name, _ = f.GetString("name")
			}
			if f.Changed("description") {
				req.Description, _ = f.GetString("description")
			}
			if f.Changed("active") {
				v, _ := f.GetBool("active")
				req.Active = &v
			}
			// Statement from flags (effect/principal/actions/resources) — only when set.
			if f.Changed("effect") || f.Changed("principal") || f.Changed("action") || f.Changed("resource") {
				effect, _ := f.GetString("effect")
				principal, _ := f.GetString("principal")
				actions, _ := f.GetStringArray("action")
				resources, _ := f.GetStringArray("resource")
				req.Statement = &policypkg.PolicyTemplate{
					Effect: effect, Principal: principal, Actions: actions, Resources: resources,
				}
			}
		}
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		p, err := client.UpdatePolicy(ctx, args[0], args[1], req)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Policy %q updated.\n", p.Name)
		return output.PrintResource(p, func() string { return p.ID }, func() error { return renderPolicyDetail(p) })
	},
}

var policyDeleteCmd = &cobra.Command{
	Use:   "delete <group-id> <policy-id>",
	Short: "Delete a policy",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		msg, err := client.DeletePolicy(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		if msg != "" {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "Policy %q deleted.\n", args[1])
		}
		output.PrintDeletedID(args[1])
		return nil
	},
}

// ---------------------------------------------------------------------------
// condition-operators (meta)
// ---------------------------------------------------------------------------

var policyConditionOperatorsCmd = &cobra.Command{
	Use:   "condition-operators",
	Short: "List accepted policy condition operators",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListConditionOperators(ctx)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			output.JSON(resp)
			return nil
		case output.FormatID:
			for _, op := range resp.Operators {
				output.PrintID(op.Name)
			}
			return nil
		}
		if len(resp.Operators) == 0 {
			fmt.Fprintln(os.Stderr, "No condition operators.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Operators))
		for _, op := range resp.Operators {
			rows = append(rows, []string{op.Name, op.Arity, strList(op.ValueTypes), op.DisplayName})
		}
		output.Table([]string{"Name", "Arity", "Value Types", "Display"}, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// decide (internal decision probe)
// ---------------------------------------------------------------------------

var policyDecideFile string

var policyDecideCmd = &cobra.Command{
	Use:   "decide <gateway> <target>",
	Short: "Probe an authorization decision for a gateway target",
	Long: `Probe an authorization decision (the internal route agent-core-gateway calls
per inbound request). Always returns a decision: allow, or deny with a reason.

The minimal flag path covers the common probe (policy group + user + action
method). For principal/context/action.params.arguments, pass the full
DecisionRequest body via --file (see the JSON-RPC action shape with --output
json is NOT provided — craft the body from the API docs).`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		var req *policypkg.DecisionRequest
		if policyDecideFile != "" {
			data, err := os.ReadFile(policyDecideFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err = loadDecisionSpec(data)
			if err != nil {
				return err
			}
		} else {
			f := cmd.Flags()
			pgID, _ := f.GetString("policy-group-id")
			if pgID == "" {
				return fmt.Errorf("required flag %q not set", "policy-group-id")
			}
			method, _ := f.GetString("method")
			if method == "" {
				return fmt.Errorf("required flag %q not set", "method")
			}
			jsonrpc, _ := f.GetString("jsonrpc")
			action := policypkg.JSONRPCAction{JSONRPC: jsonrpc, Method: method}
			if name, _ := f.GetString("action-name"); name != "" {
				action.Params = &policypkg.JSONRPCParams{Name: name}
			}
			userID, _ := f.GetString("user-id")
			userType, _ := f.GetString("user-type")
			req = &policypkg.DecisionRequest{
				PolicyGroupID: pgID,
				User: policypkg.UserInput{
					ID:   userID,
					Type: userType,
				},
				Action: action,
			}
		}
		client, err := newPolicyClient(ctx, cmd)
		if err != nil {
			return err
		}
		res, err := client.Decide(ctx, args[0], args[1], req)
		if err != nil {
			return err
		}
		return renderDecision(res, args[0], args[1])
	},
}

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderPolicyGroupDetail(g *policypkg.PolicyGroup) error {
	rows := [][]string{
		{"ID", g.ID},
		{"Name", g.Name},
		{"Description", output.StrOrDash(g.Description)},
		{"Created", formatTimeVal(g.CreatedAt)},
		{"Updated", formatTimeVal(g.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

func renderPolicyDetail(p *policypkg.Policy) error {
	rows := [][]string{
		{"ID", p.ID},
		{"Group", p.PolicyGroupID},
		{"Name", p.Name},
		{"Description", output.StrOrDash(p.Description)},
		{"Active", activeStr(p.Active)},
		{"Effect", p.Statement.Effect},
		{"Principal", p.Statement.Principal},
		{"Actions", strList(p.Statement.Actions)},
		{"Resources", strList(p.Statement.Resources)},
		{"Created", formatTimeVal(p.CreatedAt)},
		{"Updated", formatTimeVal(p.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

func renderDecision(res *policypkg.DecisionResult, gateway, target string) error {
	decision := "DENY"
	if res.Allow {
		decision = "ALLOW"
	}
	fmt.Fprintf(os.Stderr, "%s  gateway=%s target=%s\n", decision, gateway, target)
	if !res.Allow && res.Reason != nil {
		rows := [][]string{
			{"Code", res.Reason.Code},
			{"Policy", output.StrOrDash(res.Reason.PolicyID)},
			{"Message", res.Reason.Message},
		}
		output.Table([]string{"Field", "Value"}, rows)
	}
	return nil
}

func activeStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ---------------------------------------------------------------------------
// spec loaders (YAML or JSON -> struct via map bridge)
// ---------------------------------------------------------------------------

func loadPolicyGroupSpec(data []byte) (*policypkg.CreatePolicyGroupRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req policypkg.CreatePolicyGroupRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid policy-group spec: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("spec is missing required field: name")
	}
	return &req, nil
}

func loadPolicySpec(data []byte) (*policypkg.CreatePolicyRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req policypkg.CreatePolicyRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid policy spec: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("spec is missing required field: name")
	}
	if req.Statement.Effect == "" || req.Statement.Principal == "" || len(req.Statement.Actions) == 0 {
		return nil, fmt.Errorf("spec.statement is missing required field(s): effect, principal, actions")
	}
	return &req, nil
}

func loadDecisionSpec(data []byte) (*policypkg.DecisionRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req policypkg.DecisionRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid decision spec: %w", err)
	}
	if req.PolicyGroupID == "" || req.Action.Method == "" {
		return nil, fmt.Errorf("decision spec is missing required field(s): policyGroupId, action.method")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(policyCmd)

	// group
	policyCmd.AddCommand(policyGroupCmd)

	groupCreate := policyGroupCreateCmd
	groupCreate.Flags().StringVarP(&policyGroupName, "name", "n", "", "Group name (required without --interactive)")
	groupCreate.Flags().String("description", "", "Description")
	groupCreate.Flags().StringVar(&policyGroupFile, "file", "", "Apply a spec file (see 'group generate')")
	policyGroupCmd.AddCommand(groupCreate)

	policyGroupCmd.AddCommand(policyGroupGenerateCmd)

	policyGroupListCmd.Flags().Int("page", 1, "Page number (1-based)")
	policyGroupListCmd.Flags().Int("size", 10, "Page size (1-100)")
	policyGroupListCmd.Flags().String("name", "", "Filter by name (case-insensitive substring)")
	policyGroupCmd.AddCommand(policyGroupListCmd)

	policyGroupCmd.AddCommand(policyGroupGetCmd)

	groupUpdate := policyGroupUpdateCmd
	groupUpdate.Flags().String("name", "", "New name (only applied when set)")
	groupUpdate.Flags().String("description", "", "New description (only applied when set)")
	groupUpdate.Flags().StringVar(&policyGroupFile, "file", "", "Replacement spec (see 'group generate')")
	policyGroupCmd.AddCommand(groupUpdate)

	policyGroupCmd.AddCommand(policyGroupDeleteCmd)

	// policy (nested under group)
	policyGroupCmd.AddCommand(policyPolicyCmd)

	addPolicyStatementFlags := func(c *cobra.Command) {
		c.Flags().String("effect", "", "Statement effect (permit|forbid)")
		c.Flags().String("principal", "", "Principal entity id (e.g. jwt_user:abc-123, iam_role:admin, or *)")
		c.Flags().StringArray("action", nil, "Action name (repeat, or *); e.g. InsuranceAPI__read")
		c.Flags().StringArray("resource", nil, "Resource gateway ref (repeat); e.g. gateway:my-gw or gateway:*")
	}

	policyCreate := policyCreateCmd
	policyCreate.Flags().StringVarP(&policyPolicyName, "name", "n", "", "Policy name (required without --interactive)")
	policyCreate.Flags().String("description", "", "Description")
	policyCreate.Flags().Bool("active", false, "Whether the policy is active")
	addPolicyStatementFlags(policyCreate)
	policyCreate.Flags().StringVar(&policyFile, "file", "", "Apply a spec file (see 'policy generate'); authoritative when set")
	policyPolicyCmd.AddCommand(policyCreate)

	policyPolicyCmd.AddCommand(policyGenerateCmd)

	policyListCmd.Flags().Int("page", 1, "Page number (1-based)")
	policyListCmd.Flags().Int("size", 10, "Page size (1-100)")
	policyListCmd.Flags().String("name", "", "Filter by name (case-insensitive substring)")
	policyPolicyCmd.AddCommand(policyListCmd)

	policyPolicyCmd.AddCommand(policyGetCmd)

	policyUpdate := policyUpdateCmd
	policyUpdate.Flags().String("name", "", "New name (only applied when set)")
	policyUpdate.Flags().String("description", "", "New description (only applied when set)")
	policyUpdate.Flags().Bool("active", false, "Activate/deactivate (only applied when set)")
	addPolicyStatementFlags(policyUpdate)
	policyUpdate.Flags().StringVar(&policyFile, "file", "", "Full replacement spec (see 'policy generate')")
	policyPolicyCmd.AddCommand(policyUpdate)

	policyPolicyCmd.AddCommand(policyDeleteCmd)

	// condition-operators
	policyCmd.AddCommand(policyConditionOperatorsCmd)

	// decide
	decide := policyDecideCmd
	decide.Flags().String("policy-group-id", "", "Policy group id (required without --file)")
	decide.Flags().String("user-id", "", "End user id being evaluated")
	decide.Flags().String("user-type", "", "End user type (iam|jwt)")
	decide.Flags().String("method", "", "JSON-RPC action method (required without --file), e.g. tools/call")
	decide.Flags().String("action-name", "", "JSON-RPC params.name (effective action); optional")
	decide.Flags().String("jsonrpc", "2.0", "JSON-RPC version")
	decide.Flags().StringVar(&policyDecideFile, "file", "", "Full DecisionRequest body (for principal/context/arguments)")
	policyCmd.AddCommand(decide)
}

// policyGroupName / policyPolicyName hold --name for the two create commands.
var (
	policyGroupName  string
	policyPolicyName string
)
