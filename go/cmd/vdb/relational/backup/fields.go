package backup

import (
	"strings"

	"github.com/spf13/pflag"
)

// Readers for an API payload, where every JSON number decodes to float64 and any
// field may be null.

func stringField(payload map[string]interface{}, key string) string {
	value, _ := payload[key].(string)
	return value
}

func intField(payload map[string]interface{}, key string) int {
	value, _ := payload[key].(float64)
	return int(value)
}

func stringSlice(payload map[string]interface{}, key string) []string {
	items, _ := payload[key].([]interface{})
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok && value != "" {
			out = append(out, value)
		}
	}
	return out
}

// stringOrDefault and friends return the flag when the user set it, otherwise the
// fallback. Changed() rather than emptiness, so "--public-access=false" reads as a
// deliberate override.

func stringOrDefault(flags *pflag.FlagSet, name, fallback string) string {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetString(name)
	return value
}

func intOrDefault(flags *pflag.FlagSet, name string, fallback int) int {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetInt(name)
	return value
}

func boolOrDefault(flags *pflag.FlagSet, name string, fallback bool) bool {
	if !flags.Changed(name) {
		return fallback
	}
	value, _ := flags.GetBool(name)
	return value
}

func mustString(flags *pflag.FlagSet, name string) string {
	value, _ := flags.GetString(name)
	return value
}

// subnetIDPrefix is what a restore request needs in netIds. A backup record may
// carry a network ("net-") there instead, which the API rejects.
const subnetIDPrefix = "sub-"

func subnetIDsOnly(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, subnetIDPrefix) {
			out = append(out, value)
		}
	}
	return out
}

func toInterfaces(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
