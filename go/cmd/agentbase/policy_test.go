package agentbase

import (
	"strings"
	"testing"

	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
	policypkg "github.com/vngcloud/greennode-cli/internal/agentbase/policy"
)

// TestAgentbaseCmd_HasPolicySubtree verifies the policy group + nested policy
// subtrees, the condition-operators meta command, and the decide probe. Policy
// resources are synchronous (no async FSM), so there is no 'wait'.
func TestAgentbaseCmd_HasPolicySubtree(t *testing.T) {
	polCmd, _, err := AgentbaseCmd.Find([]string{"policy"})
	if err != nil {
		t.Fatalf("agentbase has no 'policy' subcommand: %v", err)
	}
	assertSubCommands(t, polCmd, "group", "condition-operators", "decide")
	assertNoSubCommands(t, polCmd, "wait")

	groupCmd, _, err := polCmd.Find([]string{"group"})
	if err != nil {
		t.Fatalf("policy has no 'group' subcommand: %v", err)
	}
	assertSubCommands(t, groupCmd, "create", "generate", "list", "get", "update", "delete", "policy")

	polLeafCmd, _, err := groupCmd.Find([]string{"policy"})
	if err != nil {
		t.Fatalf("policy group has no 'policy' subcommand: %v", err)
	}
	assertSubCommands(t, polLeafCmd, "create", "generate", "list", "get", "update", "delete")
	assertNoSubCommands(t, polLeafCmd, "wait")
}

// TestPolicyGroupGenerate_RoundTripsThroughLoad proves the group template prints
// a valid, loadable create spec.
func TestPolicyGroupGenerate_RoundTripsThroughLoad(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := policyGroupGenerateCmd.RunE(policyGroupGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	req, err := loadPolicyGroupSpec([]byte(template))
	if err != nil {
		t.Fatalf("generated template is not a valid spec: %v", err)
	}
	if req.Name != "my-group" {
		t.Errorf("round-trip decoded unexpected spec: %+v", req)
	}
}

// TestPolicyGenerate_RoundTripsThroughLoad proves the policy template (with a
// full statement) prints a valid, loadable create spec. The condition block is
// commented out in the template, so it must decode as absent.
func TestPolicyGenerate_RoundTripsThroughLoad(t *testing.T) {
	output.SetFormat(output.FormatTable)
	template := captureStdout(t, func() {
		if err := policyGenerateCmd.RunE(policyGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	req, err := loadPolicySpec([]byte(template))
	if err != nil {
		t.Fatalf("generated template is not a valid spec: %v", err)
	}
	if req.Name != "allow-admin-read" || !req.Active {
		t.Errorf("round-trip decoded unexpected spec: %+v", req)
	}
	s := req.Statement
	if s.Effect != "permit" || s.Principal != "jwt_role:admin" {
		t.Errorf("statement mismatch: %+v", s)
	}
	if len(s.Actions) != 1 || s.Actions[0] != "InsuranceAPI__read" {
		t.Errorf("actions mismatch: %+v", s.Actions)
	}
	if len(s.Condition) != 0 {
		t.Errorf("condition should be absent (commented in template): %+v", s.Condition)
	}
}

func TestPolicyGenerate_PrintsJSONSkeleton(t *testing.T) {
	output.SetFormat(output.FormatJSON)
	defer output.SetFormat(output.FormatTable)
	got := captureStdout(t, func() {
		if err := policyGenerateCmd.RunE(policyGenerateCmd, nil); err != nil {
			t.Fatalf("generate RunE: %v", err)
		}
	})
	if !strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Errorf("expected JSON skeleton; got:\n%s", got)
	}
}

// TestLoadPolicySpec_NestedConditionDecodes exercises the YAML→map→JSON→struct
// bridge on the nested statement + condition (operator → {keyPath: value}).
func TestLoadPolicySpec_NestedConditionDecodes(t *testing.T) {
	spec := []byte(`
name: complex
description: a policy
active: true
statement:
  effect: permit
  principal: "jwt_user:u-1"
  actions:
    - InsuranceAPI__read
    - InsuranceAPI__write
  resources:
    - "gateway:gw-1"
  condition:
    when:
      equals: {context.role: admin}
      in: {context.env: [prod, stg]}
    unless:
      lessThan: {context.hour: 9}
`)
	req, err := loadPolicySpec(spec)
	if err != nil {
		t.Fatalf("loadPolicySpec: %v", err)
	}
	s := req.Statement
	if len(s.Actions) != 2 || s.Actions[1] != "InsuranceAPI__write" {
		t.Errorf("actions mismatch: %+v", s.Actions)
	}
	when, ok := s.Condition["when"]
	if !ok {
		t.Fatalf("missing condition.when: %+v", s.Condition)
	}
	eq, _ := when["equals"].(map[string]any)
	if eq["context.role"] != "admin" {
		t.Errorf("equals mismatch: %+v", eq)
	}
	inOp, _ := when["in"].(map[string]any)
	inEnv, _ := inOp["context.env"].([]any)
	if len(inEnv) != 2 || inEnv[0] != "prod" {
		t.Errorf("in mismatch: %+v", inOp["context.env"])
	}
	unless, ok := s.Condition["unless"]
	if !ok {
		t.Fatalf("missing condition.unless")
	}
	lt, _ := unless["lessThan"].(map[string]any)
	if lt["context.hour"] != float64(9) {
		t.Errorf("lessThan mismatch: %+v", lt)
	}

	// Ensure it round-trips into the request struct cleanly.
	_ = policypkg.CreatePolicyRequest{Name: req.Name, Statement: req.Statement}
}

func TestLoadPolicySpec_MissingRequiredFails(t *testing.T) {
	cases := map[string]string{
		"missing name":    `statement: {effect: permit, principal: "*", actions: ["*"]}`,
		"missing effect":  "name: p\nstatement: {principal: \"*\", actions: [\"*\"]}",
		"missing actions": "name: p\nstatement: {effect: permit, principal: \"*\"}",
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadPolicySpec([]byte(spec)); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestLoadPolicyGroupSpec_MissingNameFails(t *testing.T) {
	if _, err := loadPolicyGroupSpec([]byte(`description: x`)); err == nil {
		t.Fatal("expected error for missing name")
	}
}

// TestLoadDecisionSpec_Decodes verifies the JSON-RPC action envelope + user
// decode through the map bridge, and required-field validation.
func TestLoadDecisionSpec_Decodes(t *testing.T) {
	spec := []byte(`
policyGroupId: policyengine_1
user: {id: u-1, type: jwt}
action:
  jsonrpc: "2.0"
  method: tools/call
  params:
    name: InsuranceAPI__read
    arguments: {id: x}
context: {ip: 10.0.0.1}
`)
	req, err := loadDecisionSpec(spec)
	if err != nil {
		t.Fatalf("loadDecisionSpec: %v", err)
	}
	if req.PolicyGroupID != "policyengine_1" || req.User.Type != "jwt" {
		t.Errorf("top-level mismatch: %+v", req)
	}
	if req.Action.Method != "tools/call" || req.Action.Params.Name != "InsuranceAPI__read" {
		t.Errorf("action mismatch: %+v", req.Action)
	}
}

func TestLoadDecisionSpec_MissingRequiredFails(t *testing.T) {
	if _, err := loadDecisionSpec([]byte(`user: {id: u}`)); err == nil {
		t.Fatal("expected error for missing policyGroupId/method")
	}
}

func TestActiveStr(t *testing.T) {
	if activeStr(true) != "true" || activeStr(false) != "false" {
		t.Errorf("activeStr mismatch")
	}
}
