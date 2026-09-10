package output

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/redact"
)

var showSecret bool

// SetShowSecret controls credential output, never diagnostics.
func SetShowSecret(show bool) { showSecret = show }

func credentialField(key string) bool {
	key = strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(key))
	switch key {
	case "clientid", "publickey", "tokentype", "tokenurl", "authorizationurl", "clientsecretexpiresat", "tokenendpointauthmethod":
		return false
	default:
		return redact.IsSensitiveKey(key)
	}
}

func redactCredentials(v any, opaque bool) any {
	switch value := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			if credentialField(key) || (opaque && strings.EqualFold(strings.ReplaceAll(key, "_", ""), "authorizationurl")) {
				result[key] = redact.Value
				continue
			}
			childOpaque := opaque && (key == "data" || key == "result" || key == "value")
			result[key] = redactCredentials(item, childOpaque)
		}
		return result
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = redactCredentials(item, opaque)
		}
		return result
	case string:
		if opaque {
			return redact.Value
		}
	}
	return v
}

func prepareJSON(v any, opaque bool) (any, error) {
	if showSecret {
		return v, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	return redactCredentials(normalized, opaque), nil
}

// SecretJSON also masks scalar and envelope credentials.
func SecretJSON(v any) error { return writeJSON(v, true) }

// Secret masks an explicitly credential-bearing value.
func Secret(value string) string {
	if showSecret {
		return value
	}
	return redact.Value
}

func PrintSecret(value string) { PrintID(Secret(value)) }
