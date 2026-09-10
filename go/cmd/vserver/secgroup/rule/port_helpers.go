package rule

import (
	"fmt"
	"strings"
)

func parseTags(raw []string) ([]any, error) {
	tags := make([]any, 0, len(raw))
	for _, item := range raw {
		key, value, found := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --tag %q: expected key=value form with a non-empty key", item)
		}
		tags = append(tags, map[string]any{"key": key, "value": strings.TrimSpace(value)})
	}
	return tags, nil
}
