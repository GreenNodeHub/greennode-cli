package policy

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestBuildCreateBody(t *testing.T) {
	t.Run("REDIRECT_TO_POOL requires and sends the pool", func(t *testing.T) {
		got, err := buildCreateBody("api", actionRedirectToPool, "pool-1", "", 0, false,
			[]string{"ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["redirectPoolId"] != "pool-1" {
			t.Errorf("redirectPoolId = %v", got["redirectPoolId"])
		}
		if _, ok := got["redirectUrl"]; ok {
			t.Error("redirectUrl must not be sent for a REDIRECT_TO_POOL policy")
		}
		rules := got["rules"].([]map[string]interface{})
		if len(rules) != 1 || rules[0]["ruleValue"] != "/api" {
			t.Errorf("rules = %#v", rules)
		}
	})

	t.Run("REDIRECT_TO_URL requires and sends the URL", func(t *testing.T) {
		got, err := buildCreateBody("redir", actionRedirectToURL, "", "https://example.com", 301, true,
			[]string{"ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=old.example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["redirectUrl"] != "https://example.com" || got["redirectHttpCode"] != 301 {
			t.Errorf("body = %v", got)
		}
		if got["keepQueryString"] != true {
			t.Errorf("keepQueryString = %v, want true", got["keepQueryString"])
		}
		rules, ok := got["rules"].([]map[string]interface{})
		if !ok || len(rules) != 1 {
			t.Fatalf("rules = %#v, want the one rule passed", got["rules"])
		}
	})

	t.Run("mismatched action and target are rejected", func(t *testing.T) {
		for _, tc := range []struct {
			name, action, pool, url string
			code                    int
		}{
			{"pool action without a pool", actionRedirectToPool, "", "", 0},
			{"url action without a url", actionRedirectToURL, "", "", 0},
			{"unknown action", "REDIRECT_TO_MARS", "pool-1", "", 0},
			{"invalid redirect code", actionRedirectToURL, "", "https://example.com", 307},
		} {
			rules := []string{"ruleType=PATH,compareType=STARTS_WITH,ruleValue=/x"}
			if _, err := buildCreateBody("p", tc.action, tc.pool, tc.url, tc.code, false, rules); err == nil {
				t.Errorf("%s: expected a validation error", tc.name)
			}
		}
	})
}

// currentPolicy is a trimmed L7PolicyDto as returned by GET.
func currentPolicy() map[string]interface{} {
	return map[string]interface{}{
		"uuid": "pol-1", "name": "api", "action": actionRedirectToPool,
		"redirectPoolId": "pool-1", "redirectPoolName": "web", "position": float64(0),
		"displayStatus": "ACTIVE",
		"l7Rules": []interface{}{
			map[string]interface{}{
				"uuid": "r-1", "ruleType": "PATH", "compareType": "STARTS_WITH",
				"ruleValue": "/api", "provisioningStatus": "ACTIVE",
			},
		},
	}
}

func updateFlagSet(t *testing.T, set map[string]string) *pflag.FlagSet {
	t.Helper()
	f := pflag.NewFlagSet("update", pflag.ContinueOnError)
	addUpdateFlags(f)
	for name, value := range set {
		if err := f.Set(name, value); err != nil {
			t.Fatalf("Set(%q, %q): %v", name, value, err)
		}
	}
	return f
}

func TestBuildUpdateBody(t *testing.T) {
	t.Run("the current action and target are carried over", func(t *testing.T) {
		got, err := buildUpdateBody(currentPolicy(), updateFlagSet(t, map[string]string{"redirect-pool-id": "pool-2"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["action"] != actionRedirectToPool {
			t.Errorf("action = %v, want the current REDIRECT_TO_POOL", got["action"])
		}
		if got["redirectPoolId"] != "pool-2" {
			t.Errorf("redirectPoolId = %v, want the override", got["redirectPoolId"])
		}
	})

	t.Run("unset rules are preserved and stripped of read-only fields", func(t *testing.T) {
		got, _ := buildUpdateBody(currentPolicy(), updateFlagSet(t, map[string]string{"redirect-pool-id": "pool-2"}))
		rules := got["rules"].([]map[string]interface{})
		if len(rules) != 1 {
			t.Fatalf("rules length = %d, want the 1 existing rule", len(rules))
		}
		want := map[string]interface{}{"ruleType": "PATH", "compareType": "STARTS_WITH", "ruleValue": "/api"}
		if !reflect.DeepEqual(rules[0], want) {
			t.Errorf("rule = %#v, want %#v (uuid/provisioningStatus must be dropped)", rules[0], want)
		}
	})

	t.Run("--rule replaces the whole list", func(t *testing.T) {
		f := pflag.NewFlagSet("update", pflag.ContinueOnError)
		addUpdateFlags(f)
		if err := f.Set("rule", "ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=example.com"); err != nil {
			t.Fatal(err)
		}
		got, err := buildUpdateBody(currentPolicy(), f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rules := got["rules"].([]map[string]interface{})
		if len(rules) != 1 || rules[0]["ruleValue"] != "example.com" {
			t.Errorf("rules = %#v, want only the new rule", rules)
		}
	})

	t.Run("switching action to REDIRECT_TO_URL requires a url", func(t *testing.T) {
		if _, err := buildUpdateBody(currentPolicy(), updateFlagSet(t, map[string]string{"action": actionRedirectToURL})); err == nil {
			t.Error("expected an error: the current policy has no redirectUrl to fall back on")
		}
		got, err := buildUpdateBody(currentPolicy(), updateFlagSet(t, map[string]string{
			"action": actionRedirectToURL, "redirect-url": "https://example.com",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["redirectUrl"] != "https://example.com" {
			t.Errorf("redirectUrl = %v", got["redirectUrl"])
		}
		if _, ok := got["redirectPoolId"]; ok {
			t.Error("redirectPoolId must not be sent once the action is REDIRECT_TO_URL")
		}
	})

	t.Run("an unknown current action is an actionable error", func(t *testing.T) {
		if _, err := buildUpdateBody(map[string]interface{}{}, updateFlagSet(t, map[string]string{"redirect-url": "x"})); err == nil {
			t.Error("expected an error when the action cannot be determined")
		}
	})
}

func TestUpdateFlagsCoverEveryBodyFlag(t *testing.T) {
	f := pflag.NewFlagSet("update", pflag.ContinueOnError)
	addUpdateFlags(f)
	declared := map[string]bool{}
	f.VisitAll(func(fl *pflag.Flag) { declared[fl.Name] = true })

	for _, name := range updateFlags {
		if !declared[name] {
			t.Errorf("updateFlags lists %q, which addUpdateFlags does not declare", name)
		}
		delete(declared, name)
	}
	for name := range declared {
		t.Errorf("flag %q is declared but missing from updateFlags; "+
			"setting it alone would fail the 'nothing to update' check", name)
	}
}

// The service rejects an L7 policy with no rules — both omitting the property
// and sending an empty list return "400 Missing required rules property". The
// spec marks rules optional, so this is checked client-side with a message that
// points at the alternative.
func TestBuildCreateBodyRequiresAtLeastOneRule(t *testing.T) {
	for _, rules := range [][]string{nil, {}, {""}} {
		_, err := buildCreateBody("p", actionRedirectToPool, "pool-1", "", 0, false, rules)
		if err == nil {
			t.Errorf("rules=%#v: expected the create to be refused", rules)
			continue
		}
		if !strings.Contains(err.Error(), "--rule is required") {
			t.Errorf("rules=%#v: error = %q, want it to name --rule", rules, err)
		}
	}
}
