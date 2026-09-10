package secgroup

import (
	"fmt"
	"regexp"
	"strings"
)

var secgroupDescriptionPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.@ -]{0,254}$`)

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

func validateSecgroupDescription(description string) error {
	if description == "" {
		return nil
	}
	if !secgroupDescriptionPattern.MatchString(description) {
		return fmt.Errorf("--description must be empty or 1-255 characters, start with a letter, and contain only letters, digits, spaces, '_', '.', '@', or '-'")
	}
	return nil
}
