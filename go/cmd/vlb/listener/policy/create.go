package policy

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

const (
	actionRedirectToPool = "REDIRECT_TO_POOL"
	actionRedirectToURL  = "REDIRECT_TO_URL"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new L7 policy",
	RunE:  runCreate,
}

func init() {
	addParentFlags(createCmd)
	f := createCmd.Flags()
	f.String("name", "", "Policy name (required)")
	f.String("action", "", "Action: REDIRECT_TO_POOL or REDIRECT_TO_URL (required)")
	f.String("redirect-pool-id", "", "Pool to forward to (required for REDIRECT_TO_POOL)")
	f.String("redirect-url", "", "URL to redirect to (required for REDIRECT_TO_URL)")
	f.Int("redirect-http-code", 0, "Redirect HTTP code: 301 or 302 (REDIRECT_TO_URL only)")
	f.Bool("keep-query-string", false, "Keep the query string when redirecting")
	f.StringArray("rule", nil,
		"Match rule, shorthand (ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api) or JSON (repeatable, at least one required)")
	f.Bool("dry-run", false, "Validate parameters and print the request body without creating")

	for _, name := range []string{"name", "action"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// buildCreateBody assembles the CreateL7PolicyRequest. It rejects
// action/target mismatches client-side so the user gets a precise message
// instead of a generic backend 400.
func buildCreateBody(name, action, redirectPoolID, redirectURL string, redirectHTTPCode int,
	keepQueryString bool, rules []string) (map[string]interface{}, error) {

	body := map[string]interface{}{
		"name":   name,
		"action": action,
	}

	switch action {
	case actionRedirectToPool:
		if redirectPoolID == "" {
			return nil, fmt.Errorf("--redirect-pool-id is required when --action is %s", actionRedirectToPool)
		}
		body["redirectPoolId"] = redirectPoolID
	case actionRedirectToURL:
		if redirectURL == "" {
			return nil, fmt.Errorf("--redirect-url is required when --action is %s", actionRedirectToURL)
		}
		body["redirectUrl"] = redirectURL
		if redirectHTTPCode != 0 {
			if redirectHTTPCode != 301 && redirectHTTPCode != 302 {
				return nil, fmt.Errorf("--redirect-http-code must be 301 or 302, got %d", redirectHTTPCode)
			}
			body["redirectHttpCode"] = redirectHTTPCode
		}
		body["keepQueryString"] = keepQueryString
	default:
		return nil, fmt.Errorf("--action must be %s or %s, got %q", actionRedirectToPool, actionRedirectToURL, action)
	}

	parsedRules, err := cli.ParseStructFlags("rule", rules, ruleIntFields, nil)
	if err != nil {
		return nil, err
	}
	// `rules` is optional in the spec, but the service rejects a policy that has
	// none — verified against the live API, where both omitting the property and
	// sending an empty list return "400 Missing required rules property". There
	// is no such thing as a catch-all L7 policy here; the listener's
	// --default-pool-id and --default-action cover unmatched traffic.
	if len(parsedRules) == 0 {
		return nil, fmt.Errorf("--rule is required: an L7 policy must match on at least " +
			"one rule (use the listener's --default-pool-id for unmatched traffic)")
	}
	body["rules"] = parsedRules

	return body, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	lbID, listenerID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	name, _ := f.GetString("name")
	action, _ := f.GetString("action")
	redirectPoolID, _ := f.GetString("redirect-pool-id")
	redirectURL, _ := f.GetString("redirect-url")
	redirectHTTPCode, _ := f.GetInt("redirect-http-code")
	keepQueryString, _ := f.GetBool("keep-query-string")
	rules, _ := f.GetStringArray("rule")
	dryRun, _ := f.GetBool("dry-run")

	// Refuse flags that do not belong to the chosen action, exactly as
	// `policy update` does. buildCreateBody would otherwise drop them, and the
	// policy would be created without the settings the user asked for.
	if err := rejectActionMismatch(f, action); err != nil {
		return err
	}

	body, err := buildCreateBody(name, action, redirectPoolID, redirectURL, redirectHTTPCode, keepQueryString, rules)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("create", "L7 policy "+name+" on listener "+listenerID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.Policies(projectID, lbID, listenerID), body)
	if err != nil {
		return fmt.Errorf("failed to create L7 policy: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}
