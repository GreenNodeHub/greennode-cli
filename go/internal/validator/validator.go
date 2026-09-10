package validator

import (
	"fmt"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$`)

// ValidateID validates that an ID contains only safe path-segment characters.
func ValidateID(value, name string) error {
	if value == "" || !idPattern.MatchString(value) {
		return fmt.Errorf("invalid %s: '%s'. Must contain only alphanumeric characters, underscores, and hyphens", name, value)
	}
	return nil
}
