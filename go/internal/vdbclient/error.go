package vdbclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/client"
)

// Error is an API error with the vDB error payload unpacked into the message.
//
// Why this exists: on any status >= 400 the shared client builds its message from
// the first recognised field of the body, preferring "message". vDB always sets
// that field — but to a machine code, not a sentence:
//
//	{"code":400,
//	 "message":"in_valid",
//	 "errors":[{"typeError":"pageSize","fieldError":"invalid",
//	            "message":"The field pageSize of the request must be greater than 0"}]}
//
// So the user saw "API error (HTTP 400 Bad Request): in_valid" and the sentence
// that actually says what to fix was only visible under --debug. This type merges
// the two, and is applied to every vdb request by Client.
//
// It embeds the shared *client.APIError and unwraps to it, so errors.As still
// reaches StatusCode/Body for callers that branch on them (waiters, retry logic).
type Error struct {
	*client.APIError
	message string
}

func (e *Error) Error() string { return e.message }

// Unwrap exposes the underlying APIError to errors.As / errors.Is.
func (e *Error) Unwrap() error { return e.APIError }

// vdbErrorBody is the subset of the vDB error payload worth showing. Fields not
// listed here (action, request) come back null in practice.
type vdbErrorBody struct {
	Message string `json:"message"`
	Errors  []struct {
		Message    string `json:"message"`
		TypeError  string `json:"typeError"`
		FieldError string `json:"fieldError"`
	} `json:"errors"`
}

// enrich upgrades an API error's message with the vDB error details, and removes from it
// any secret the request carried. Anything that is not an *client.APIError, or whose body
// does not carry usable details, is returned untouched unless it needs redacting —
// enrich never hides an error it cannot improve.
//
// requestBody is the body that was sent, and is read only to learn which strings are
// secret; nil for requests that have none.
func enrich(err error, requestBody interface{}) error {
	if err == nil {
		return nil
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		return err
	}

	secrets := secretValues(requestBody)

	var body vdbErrorBody
	if jsonErr := json.Unmarshal([]byte(apiErr.Body), &body); jsonErr != nil {
		return redacted(apiErr, apiErr.Error(), secrets)
	}

	details := make([]string, 0, len(body.Errors))
	for _, detail := range body.Errors {
		text := strings.TrimSpace(detail.Message)
		if text == "" {
			continue
		}
		// Name the offending field when the API identifies one: "pageSize" is what
		// the user has to change, and typeError is what support will ask for.
		if detail.TypeError != "" {
			text = fmt.Sprintf("%s (%s)", text, detail.TypeError)
		}
		details = append(details, text)
	}
	if len(details) == 0 {
		return redacted(apiErr, apiErr.Error(), secrets)
	}

	// Keep the original message — it is the machine code the API team greps for —
	// and append the human-readable part after it.
	return redacted(apiErr, fmt.Sprintf("%s: %s", apiErr.Error(), strings.Join(details, "; ")), secrets)
}

// redacted returns the error carrying message with every secret masked. When nothing
// changes and the message is the API error's own, the original error is returned so
// callers keep the exact value they would have had.
func redacted(apiErr *client.APIError, message string, secrets []string) error {
	for _, secret := range secrets {
		message = strings.ReplaceAll(message, secret, "***")
	}
	if message == apiErr.Error() {
		return apiErr
	}
	return &Error{APIError: apiErr, message: message}
}

// secretValues collects the values a request body carried under a sensitive key, so they
// can be masked wherever the API repeats them. Short values are skipped: masking a
// two-character password would blank out unrelated text in the message.
func secretValues(v interface{}) []string {
	var out []string
	collectSecrets(v, &out)
	return out
}

func collectSecrets(v interface{}, out *[]string) {
	switch value := v.(type) {
	case map[string]interface{}:
		for key, item := range value {
			if text, isText := item.(string); isText && sensitiveKeys[key] {
				if len(text) >= 4 {
					*out = append(*out, text)
				}
				continue
			}
			collectSecrets(item, out)
		}
	case []interface{}:
		for _, item := range value {
			collectSecrets(item, out)
		}
	}
}
