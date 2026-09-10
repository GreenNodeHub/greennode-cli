// Package redact masks credential-shaped JSON, bodies, URLs, and headers.
package redact

import (
	"encoding/json"
	"net/url"
	"strings"
)

// Value is the sentinel written in place of a redacted value.
const Value = "[REDACTED]"

// exactSensitiveKeys matches ambiguous names exactly to avoid masking unrelated fields.
var exactSensitiveKeys = map[string]struct{}{
	"key":            {},
	"auth":           {},
	"authentication": {},
}

// sensitiveFragments matches unambiguous credential fragments in normalized keys.
var sensitiveFragments = []string{
	"secret",
	"password",
	"passphrase",
	"token",
	"privatekey",
	"accesskey",
	"apikey",
	"authorization",
	"credential",
	"clientid",
	"kubeconfig",
	"clientcertificatedata",
	"userdata",
	"consoleurl",
	"clientsecret",
	// Match embedded key material, including kubeconfig keys.
	"keydata",
}

// sensitiveHeaderNames always masks authentication and session headers.
var sensitiveHeaderNames = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
}

// normalizeKey lowercases names and removes separators.
func normalizeKey(key string) string {
	return strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
}

// IsSensitiveKey identifies credential-bearing names.
func IsSensitiveKey(key string) bool {
	normalized := normalizeKey(key)
	if _, ok := exactSensitiveKeys[normalized]; ok {
		return true
	}
	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	// Keys ending in "key" are conservatively masked; keyName and keyword remain visible.
	return strings.HasSuffix(normalized, "key")
}

// JSON recursively copies decoded JSON and masks credential fields without modifying its input.
func JSON(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, item := range typed {
			if IsSensitiveKey(key) {
				redacted[key] = Value
				continue
			}
			redacted[key] = JSON(item)
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, item := range typed {
			redacted[index] = JSON(item)
		}
		return redacted
	default:
		return v
	}
}

// Body masks valid JSON; non-JSON text is unchanged. Log only length metadata for unknown bodies.
func Body(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return raw
	}
	out, err := json.Marshal(JSON(v))
	if err != nil {
		return raw
	}
	return string(out)
}

// URL masks sensitive query values, preserving untouched encoding. Invalid URLs are unchanged.
func URL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	if parsed.RawQuery == "" {
		return u
	}

	pairs := strings.Split(parsed.RawQuery, "&")
	for i, pair := range pairs {
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			continue
		}
		name, err := url.QueryUnescape(pair[:eq])
		if err != nil {
			name = pair[:eq]
		}
		if IsSensitiveKey(name) {
			pairs[i] = pair[:eq+1] + url.QueryEscape(Value)
		}
	}
	parsed.RawQuery = strings.Join(pairs, "&")
	return parsed.String()
}

// PathValues masks declared credentials in raw and escaped forms, including URL path segments.
func PathValues(u string, secrets []string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		u = strings.ReplaceAll(u, secret, Value)
		if escaped := url.PathEscape(secret); escaped != secret {
			u = strings.ReplaceAll(u, escaped, Value)
		}
	}
	return u
}

// Headers copies and masks credential headers using case-insensitive names.
func Headers(h map[string][]string) map[string][]string {
	if h == nil {
		return nil
	}
	redacted := make(map[string][]string, len(h))
	for name, values := range h {
		if isSensitiveHeader(name) {
			masked := make([]string, len(values))
			for i := range masked {
				masked[i] = Value
			}
			redacted[name] = masked
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		redacted[name] = copied
	}
	return redacted
}

func isSensitiveHeader(name string) bool {
	if _, ok := sensitiveHeaderNames[strings.ToLower(name)]; ok {
		return true
	}
	return IsSensitiveKey(name)
}
