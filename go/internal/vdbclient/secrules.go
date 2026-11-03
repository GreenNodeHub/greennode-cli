package vdbclient

import (
	"fmt"
	"strconv"
	"strings"
)

// Security rules are shared between the relational and memorystore APIs (and so
// between the Relational Database and PostgreSQL Cluster command groups, which both
// use the relational endpoint). Both take the same body: a JSON ARRAY of
// UpdateSecurityGroupRuleDetail, replacing the whole rule set.
//
// Only the default port differs per engine, so it is a parameter.

// SecurityRuleKeys documents the accepted --rule keys. An unknown key must be an
// error rather than an ignored field: a typo'd "cird=" would otherwise send a rule
// with no network, and since the endpoint REPLACES the rule set, that revokes
// everything.
var SecurityRuleKeys = []string{"cidr", "port", "port-min", "port-max", "id"}

var securityRuleKeySet = func() map[string]bool {
	set := make(map[string]bool, len(SecurityRuleKeys))
	for _, k := range SecurityRuleKeys {
		set[k] = true
	}
	return set
}()

// ParseSecurityRules turns each `key=value,...` spec into an
// UpdateSecurityGroupRuleDetail. defaultPort is used when a rule names no port —
// 3306 for MySQL/MariaDB, 5432 for PostgreSQL, 6379 for Redis.
func ParseSecurityRules(specs []string, defaultPort int) ([]interface{}, error) {
	rules := make([]interface{}, 0, len(specs))

	for _, spec := range specs {
		fields := map[string]string{}
		for _, pair := range strings.Split(spec, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			key, value, found := strings.Cut(pair, "=")
			key = strings.TrimSpace(strings.ToLower(key))
			if !found {
				return nil, fmt.Errorf("invalid --rule %q: %q is not key=value", spec, pair)
			}
			if !securityRuleKeySet[key] {
				return nil, fmt.Errorf("invalid --rule %q: unknown key %q (accepted: %s)",
					spec, key, strings.Join(SecurityRuleKeys, ", "))
			}
			fields[key] = strings.TrimSpace(value)
		}

		cidr := fields["cidr"]
		if cidr == "" {
			return nil, fmt.Errorf("invalid --rule %q: cidr is required", spec)
		}

		portMin, portMax, err := rulePorts(spec, fields, defaultPort)
		if err != nil {
			return nil, err
		}

		rule := map[string]interface{}{
			"portRangeMin":   portMin,
			"portRangeMax":   portMax,
			"remoteIpPrefix": cidr,
		}
		// The API reads a null id as "insert new"; only send one when keeping an
		// existing rule.
		if id := fields["id"]; id != "" {
			rule["id"] = id
		}
		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return nil, fmt.Errorf("no rules given: pass at least one --rule")
	}
	return rules, nil
}

func rulePorts(spec string, fields map[string]string, defaultPort int) (int, int, error) {
	single, hasSingle := fields["port"]
	minValue, hasMin := fields["port-min"]
	maxValue, hasMax := fields["port-max"]

	if hasSingle && (hasMin || hasMax) {
		return 0, 0, fmt.Errorf("invalid --rule %q: use either port or port-min/port-max, not both", spec)
	}

	switch {
	case hasSingle:
		port, err := parsePort(spec, "port", single)
		return port, port, err
	case hasMin || hasMax:
		if !hasMin || !hasMax {
			return 0, 0, fmt.Errorf("invalid --rule %q: port-min and port-max must be given together", spec)
		}
		low, err := parsePort(spec, "port-min", minValue)
		if err != nil {
			return 0, 0, err
		}
		high, err := parsePort(spec, "port-max", maxValue)
		if err != nil {
			return 0, 0, err
		}
		if low > high {
			return 0, 0, fmt.Errorf("invalid --rule %q: port-min %d is above port-max %d", spec, low, high)
		}
		return low, high, nil
	default:
		return defaultPort, defaultPort, nil
	}
}

func parsePort(spec, key, value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid --rule %q: %s=%q is not a port number", spec, key, value)
	}
	return port, nil
}

// SecurityRulesFrom reshapes a GET /secrules response into the update format,
// keeping each rule's ID so re-sending it preserves the rule instead of recreating
// it. Used to show the current set and to implement "append" semantics on top of an
// endpoint that only supports replace.
func SecurityRulesFrom(result interface{}) []interface{} {
	items, ok := Unwrap(result).([]interface{})
	if !ok {
		return nil
	}

	out := make([]interface{}, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		rule := map[string]interface{}{
			"portRangeMin":   row["portRangeMin"],
			"portRangeMax":   row["portRangeMax"],
			"remoteIpPrefix": row["remoteIpPrefix"],
		}
		if id, _ := row["id"].(string); id != "" {
			rule["id"] = id
		}
		out = append(out, rule)
	}
	return out
}

// DescribeSecurityRules renders a rule set as
// "203.0.113.0/24:5432, 10.0.0.0/8:5432-5433" for the confirmation prompt.
func DescribeSecurityRules(rules []interface{}) string {
	if len(rules) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(rules))
	for _, item := range rules {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		low := fmt.Sprintf("%v", rule["portRangeMin"])
		high := fmt.Sprintf("%v", rule["portRangeMax"])
		ports := low
		if low != high {
			ports = low + "-" + high
		}
		parts = append(parts, fmt.Sprintf("%v:%s", rule["remoteIpPrefix"], ports))
	}
	return strings.Join(parts, ", ")
}

// SecurityRuleColumns is the table view of a SecurityGroupRuleEntity. status is
// omitted: the API leaves it empty on every rule.
var SecurityRuleColumns = []string{
	"id", "direction", "protocol", "portRangeMin", "portRangeMax",
	"remoteIpPrefix", "createdAt",
}
