package cli

import "github.com/greennodehub/greennode-cli/internal/redact"

// RedactJSON is a compatibility wrapper; new callers should use internal/redact.JSON.
func RedactJSON(value any) any {
	return redact.JSON(value)
}
