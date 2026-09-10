package client

import "github.com/greennodehub/greennode-cli/internal/redact"

// RedactedValue is a compatibility alias for internal/redact.Value.
const RedactedValue = redact.Value

// redactDebugBody delegates credential masking to internal/redact.Body.
func redactDebugBody(raw string) string {
	return redact.Body(raw)
}

// RedactJSON is a compatibility wrapper; new callers should use internal/redact.JSON.
func RedactJSON(value any) any {
	return redact.JSON(value)
}
