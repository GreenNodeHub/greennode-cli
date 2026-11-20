package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseCommaSeparated splits a comma-separated string into a trimmed slice,
// dropping empty entries.
func ParseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ParseStructFlag parses a struct-valued CLI flag accepting either JSON
// ({"minSize":2,"maxSize":10}) or AWS-style shorthand (minSize=2,maxSize=10).
// Keys listed in intFields are coerced to int in the shorthand form; all other
// values stay strings. JSON is passed through as decoded. An empty/blank value
// returns (nil, nil). Returns an error on malformed JSON, a shorthand entry
// without '=', or a non-integer value for an int field.
func ParseStructFlag(value string, intFields ...string) (map[string]interface{}, error) {
	return ParseStructFlagTyped(value, intFields, nil)
}

// ParseStructFlagTyped is ParseStructFlag with explicit int and bool field
// lists. In shorthand form, keys in intFields are coerced to int and keys in
// boolFields to bool ("true"/"false"); all other values stay strings. JSON is
// passed through as decoded.
func ParseStructFlagTyped(value string, intFields, boolFields []string) (map[string]interface{}, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil, nil
	}
	if strings.HasPrefix(v, "{") {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return m, nil
	}

	ints := map[string]bool{}
	for _, f := range intFields {
		ints[f] = true
	}
	bools := map[string]bool{}
	for _, f := range boolFields {
		bools[f] = true
	}
	out := map[string]interface{}{}
	for _, pair := range strings.Split(v, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid shorthand %q: expected key=value", pair)
		}
		key := strings.TrimSpace(pair[:idx])
		val := strings.TrimSpace(pair[idx+1:])
		switch {
		case ints[key]:
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("%s must be an integer, got %q", key, val)
			}
			out[key] = n
		case bools[key]:
			b, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("%s must be a boolean (true/false), got %q", key, val)
			}
			out[key] = b
		default:
			out[key] = val
		}
	}
	return out, nil
}

// BuildEventsQuery builds query params for events endpoints, including only
// flags the user explicitly set. `changed` maps flag name -> was it set.
// VKS pagination is 0-based, so page is passed through verbatim.
func BuildEventsQuery(action, eventType string, page, pageSize int, changed map[string]bool) map[string]string {
	params := map[string]string{}
	if changed["action"] && action != "" {
		params["action"] = action
	}
	if changed["type"] && eventType != "" {
		params["type"] = eventType
	}
	if changed["page"] {
		params["page"] = fmt.Sprintf("%d", page)
	}
	if changed["page-size"] {
		params["pageSize"] = fmt.Sprintf("%d", pageSize)
	}
	return params
}

// ParseStructFlags parses a repeatable struct-valued flag (cobra StringArray)
// into one map per occurrence, applying ParseStructFlagTyped to each. Blank
// entries are skipped; the flag name is used to prefix errors so the user is
// told which occurrence was malformed.
func ParseStructFlags(flagName string, values []string, intFields, boolFields []string) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(values))
	for i, v := range values {
		m, err := ParseStructFlagTyped(v, intFields, boolFields)
		if err != nil {
			return nil, fmt.Errorf("--%s[%d]: %w", flagName, i, err)
		}
		if m == nil {
			continue
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
