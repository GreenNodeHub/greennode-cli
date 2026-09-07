package vdbclient

import (
	"os"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/spf13/cobra"
)

// Unwrap strips the vDB response envelope.
//
// Every wrapped endpoint uses exactly one envelope — all 42 Wrap* schemas in the
// API spec have the properties {code, message, data} and nothing else:
//
//	{"code":200,"message":"...","data": <payload>}
//
// Unwrap removes only that outer layer, so every field of <payload> keeps its
// API name. It deliberately makes no assumption about the payload, because the
// payload shape varies per endpoint (checked against all 128 wrapped operations):
//
//   - 76 endpoints: data is a bare array
//   - paginated lists: an object holding the items plus pageObject, but under
//     TWO different key names — "data" for database-instances, "content" for
//     backups, configurations and histories
//   - detail endpoints: the resource object itself
//   - a few: a plain scalar
//
// The envelope is detected by shape (code AND message AND data present) rather
// than by the "data" key alone, so a payload that merely has a data field of its
// own is not stripped. No non-Wrap schema in the spec carries both code and
// message, so this cannot misfire on a real response.
//
// Kafka is the exception: only 9 of its 36 operations are wrapped, the rest
// return bare objects, bare arrays or plain strings. Unwrap passes those through
// untouched, so it is safe to call — but a kafka command must not ASSUME its
// response was wrapped.
func Unwrap(result interface{}) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return result
	}
	// Identify the envelope by its shape rather than by code alone: a payload
	// that happens to carry a "data" key but no "code"/"message" is not one.
	if _, hasCode := obj["code"]; !hasCode {
		return result
	}
	if _, hasMessage := obj["message"]; !hasMessage {
		return result
	}
	data, hasData := obj["data"]
	if !hasData || data == nil {
		return result
	}
	return data
}

// Payload unwraps the envelope and reports whether it actually carried anything.
//
// It exists because **vDB answers "not found" with HTTP 200 and a null payload**:
// asking for a backup that failed to be created returns
// `{"code":200,"message":"success","data":null}`, not a 404 (verified live,
// 2026-08-13). Unwrap alone cannot express that — with a nil data it hands back the
// envelope, so a caller doing Unwrap(...).(map[string]interface{}) gets the envelope
// and happily reads empty fields off it.
//
// Any command that fetches ONE resource and then reads fields from it must go
// through this and report a miss, rather than acting on a phantom.
func Payload(result interface{}) (interface{}, bool) {
	obj, ok := result.(map[string]interface{})
	if !ok {
		// Not an envelope at all (bare array, scalar): whatever it is, it is the
		// payload.
		return result, result != nil
	}
	if _, hasCode := obj["code"]; !hasCode {
		return result, true
	}
	if _, hasMessage := obj["message"]; !hasMessage {
		return result, true
	}
	data, hasData := obj["data"]
	if !hasData || data == nil {
		return nil, false
	}
	return data, true
}

// PayloadObject is Payload for the common case of a single resource object.
func PayloadObject(result interface{}) (map[string]interface{}, bool) {
	payload, ok := Payload(result)
	if !ok {
		return nil, false
	}
	object, ok := payload.(map[string]interface{})
	return object, ok
}

// Output unwraps the envelope and prints using the shared --output/--query flags.
func Output(cmd *cobra.Command, data interface{}) error {
	return write(cmd, Unwrap(data), nil)
}

// OutputWithColumns is Output with a fixed column set for table output. vDB list
// items are very wide (a database instance carries ~68 fields, billing metadata
// included), so table output would be unreadable without one. JSON output is
// unaffected and still carries every field.
//
// Use it for LIST responses only. It routes through formatter.extractRows, which
// picks the first slice it finds while ranging over the map — deterministic for a
// list payload (exactly one array: the items) but NOT for a detail payload, where
// a single instance carries four nested arrays (ip, securityGroup, replicas,
// sharedActions) and Go randomizes map iteration order. Detail commands should
// call Output, which renders an object as a key/value table instead.
func OutputWithColumns(cmd *cobra.Command, data interface{}, columns []string) error {
	return write(cmd, Unwrap(data), columns)
}

func write(cmd *cobra.Command, data interface{}, columns []string) error {
	output, _ := cmd.Flags().GetString("output")
	query, _ := cmd.Flags().GetString("query")

	if output == "" {
		profile, _ := cmd.Flags().GetString("profile")
		cfg, _ := config.LoadConfig(profile)
		if cfg != nil {
			output = cfg.Output
		}
	}
	if output == "" {
		output = "json"
	}

	colorMode, _ := cmd.Flags().GetString("color")
	color := formatter.ColorEnabled(colorMode, os.Stdout)

	if output == "table" {
		data = plainNumbers(data)
	}
	if output == "table" && len(columns) > 0 {
		return formatter.FormatTableWithColumnsColor(data, columns, query, os.Stdout, color)
	}
	return formatter.FormatColor(data, output, query, os.Stdout, color)
}

// plainNumbers returns a copy of data with every float64 rendered as a decimal
// string, for TABLE output only.
//
// Every JSON number decodes to float64, and the table formatter prints cells with
// %v, which switches float64 to exponent form once that is shorter: a backup of
// 5861298 bytes shows up as "5.861298e+06". vDB reports plenty of byte counts and
// IDs in that range. Converting here keeps it to the vdb commands — JSON and text
// output still carry the original numbers, so --query arithmetic and anything
// parsing the output are unaffected.
func plainNumbers(data interface{}) interface{} {
	switch value := data.(type) {
	case float64:
		// 'f' with precision -1 is the shortest form that never uses an exponent:
		// 5861298 -> "5861298", 0.06 -> "0.06".
		return strconv.FormatFloat(value, 'f', -1, 64)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(value))
		for k, item := range value {
			out[k] = plainNumbers(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(value))
		for i, item := range value {
			out[i] = plainNumbers(item)
		}
		return out
	default:
		return data
	}
}

// ExtractIDValues collects one field from every item of a listing, for shell completion,
// accepting NUMBERS as well as strings.
//
// It exists because cli.ExtractIDs takes only string values, and several vDB ids arrive as
// JSON numbers: a flavor's `id` and a backup package's `packageId` are 179 and 1, not
// "179" and "1". Completion for --package-id was silently empty until this was noticed
// live. Numbers are formatted without an exponent or trailing zeros, so the suggestion is
// exactly what the flag expects.
//
// Duplicates are dropped: one flavor name can appear twice in a zone under different ids,
// and a package listing repeats a package per engine group.
func ExtractIDValues(payload interface{}, field string) []string {
	items, ok := payload.([]interface{})
	if !ok {
		return nil
	}

	var out []string
	seen := map[string]bool{}
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var value string
		switch typed := row[field].(type) {
		case string:
			value = typed
		case float64:
			value = strconv.FormatFloat(typed, 'f', -1, 64)
		default:
			continue
		}
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
