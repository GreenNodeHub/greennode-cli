package policy

import (
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an L7 policy",
	Long: "Update a policy's action, redirect target or rules. The API requires the " +
		"full object, so unspecified fields are read from the current policy and " +
		"sent back unchanged; --rule replaces the whole rule list.",
	RunE: runUpdate,
}

func init() {
	addParentFlags(updateCmd)
	addPolicyFlag(updateCmd)
	addUpdateFlags(updateCmd.Flags())
	updateCmd.Flags().Bool("dry-run", false, "Print the request body without applying the update")
}

// addUpdateFlags declares the body-shaping flags. It is separate from init() so
// tests can build an identical flag set without mutating the shared command.
func addUpdateFlags(f *pflag.FlagSet) {
	f.String("action", "", "Action: REDIRECT_TO_POOL or REDIRECT_TO_URL")
	f.String("redirect-pool-id", "", "Pool to forward to (REDIRECT_TO_POOL only)")
	f.String("redirect-url", "", "URL to redirect to (REDIRECT_TO_URL only)")
	f.Int("redirect-http-code", 0, "Redirect HTTP code: 301 or 302 (REDIRECT_TO_URL only)")
	f.Bool("keep-query-string", false, "Keep the query string when redirecting")
	f.StringArray("rule", nil,
		"Match rule, shorthand (ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api) or JSON (repeatable); replaces the current list")
}

var updateFlags = []string{
	"action", "redirect-pool-id", "redirect-url", "redirect-http-code", "keep-query-string", "rule",
}

// currentRules converts the L7RuleDto list on a policy back into the
// L7RuleRequest shape, so an update that does not touch --rule preserves them.
func currentRules(policy map[string]interface{}) []map[string]interface{} {
	raw, ok := policy["l7Rules"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		r, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		converted := map[string]interface{}{}
		for _, k := range []string{"ruleType", "compareType", "ruleValue"} {
			if v, ok := r[k]; ok && v != nil {
				converted[k] = v
			}
		}
		if len(converted) > 0 {
			out = append(out, converted)
		}
	}
	return out
}

// buildUpdateBody assembles the UpdateL7PolicyRequest by seeding from the
// policy's current state and overlaying only what the user set. `current` is
// the unwrapped L7PolicyDto from GET.
func buildUpdateBody(current map[string]interface{}, f *pflag.FlagSet) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	action, _ := f.GetString("action")
	if !f.Changed("action") {
		action, _ = current["action"].(string)
	}
	if action == "" {
		return nil, fmt.Errorf("could not determine the policy's current action; pass --action explicitly")
	}
	body["action"] = action

	switch action {
	case actionRedirectToPool:
		// The URL-only flags have no place in a REDIRECT_TO_POOL body. Refuse
		// them instead of dropping them, or the user's change is silently lost
		// and the PUT still moves the load balancer into UPDATING.
		if err := rejectChanged(f, action, "redirect-url", "redirect-http-code", "keep-query-string"); err != nil {
			return nil, err
		}
		poolID, _ := f.GetString("redirect-pool-id")
		if !f.Changed("redirect-pool-id") {
			poolID, _ = current["redirectPoolId"].(string)
		}
		if poolID == "" {
			return nil, fmt.Errorf("--redirect-pool-id is required when --action is %s", actionRedirectToPool)
		}
		body["redirectPoolId"] = poolID
	case actionRedirectToURL:
		if err := rejectChanged(f, action, "redirect-pool-id"); err != nil {
			return nil, err
		}
		url, _ := f.GetString("redirect-url")
		if !f.Changed("redirect-url") {
			url, _ = current["redirectUrl"].(string)
		}
		if url == "" {
			return nil, fmt.Errorf("--redirect-url is required when --action is %s", actionRedirectToURL)
		}
		body["redirectUrl"] = url

		if f.Changed("redirect-http-code") {
			code, _ := f.GetInt("redirect-http-code")
			if code != 301 && code != 302 {
				return nil, fmt.Errorf("--redirect-http-code must be 301 or 302, got %d", code)
			}
			body["redirectHttpCode"] = code
		} else if v, ok := current["redirectHttpCode"]; ok && v != nil {
			body["redirectHttpCode"] = v
		}

		if f.Changed("keep-query-string") {
			body["keepQueryString"], _ = f.GetBool("keep-query-string")
		} else if v, ok := current["keepQueryString"]; ok && v != nil {
			body["keepQueryString"] = v
		}
	default:
		return nil, fmt.Errorf("--action must be %s or %s, got %q", actionRedirectToPool, actionRedirectToURL, action)
	}

	if f.Changed("rule") {
		raw, _ := f.GetStringArray("rule")
		rules, err := cli.ParseStructFlags("rule", raw, ruleIntFields, nil)
		if err != nil {
			return nil, err
		}
		if rules == nil {
			rules = []map[string]interface{}{}
		}
		body["rules"] = rules
	} else if rules := currentRules(current); len(rules) > 0 {
		body["rules"] = rules
	}

	return body, nil
}

// rejectChanged returns an error when any of the named flags was set, because
// it does not apply to the policy's action.
func rejectChanged(f *pflag.FlagSet, action string, names ...string) error {
	var set []string
	for _, n := range names {
		if f.Changed(n) {
			set = append(set, "--"+n)
		}
	}
	if len(set) > 0 {
		return fmt.Errorf("%s cannot be used when the action is %s", strings.Join(set, ", "), action)
	}
	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	lbID, listenerID, policyID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	anyChanged := false
	for _, name := range updateFlags {
		if cmd.Flags().Changed(name) {
			anyChanged = true
			break
		}
	}
	if !anyChanged {
		return fmt.Errorf("nothing to update: provide at least one of --%s",
			strings.Join(updateFlags, ", --"))
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	response, err := apiClient.Get(vlbclient.Policy(projectID, lbID, listenerID, policyID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch policy %s: %w", policyID, err)
	}

	body, err := buildUpdateBody(vlbclient.UnwrapData(response), cmd.Flags())
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("update", "L7 policy "+policyID, body)
		return nil
	}

	result, err := apiClient.Put(vlbclient.Policy(projectID, lbID, listenerID, policyID), body)
	if err != nil {
		return fmt.Errorf("failed to update policy %s: %w", policyID, err)
	}

	return outputRaw(cmd, cfg, result)
}
