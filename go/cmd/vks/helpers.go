package vks

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/cli"
	"github.com/vngcloud/greennode-cli/internal/client"
)

// parseToggle converts an "enabled"/"disabled" flag value to a bool, erroring on
// any other value. Used by --private-cluster/--private-nodes/--*-plugin flags.
func parseToggle(name, value string) (bool, error) {
	switch value {
	case "enabled":
		return true, nil
	case "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("--%s must be 'enabled' or 'disabled', got %q", name, value)
	}
}

// createClient builds a GreennodeClient for the VKS service from command flags.
func createClient(cmd *cobra.Command) (*client.GreennodeClient, error) {
	return cli.NewClient(cmd, "vks")
}

// outputResult formats and prints the API response.
func outputResult(cmd *cobra.Command, data interface{}) error {
	return cli.Output(cmd, data)
}

// parseCommaSeparated splits a comma-separated string into a trimmed slice.
func parseCommaSeparated(s string) []string {
	return cli.ParseCommaSeparated(s)
}

// buildEventsQuery builds query params for events endpoints (see cli.BuildEventsQuery).
func buildEventsQuery(action, eventType string, page, pageSize int, changed map[string]bool) map[string]string {
	return cli.BuildEventsQuery(action, eventType, page, pageSize, changed)
}

// parseLabels parses "key1=val1,key2=val2" into a map.
func parseLabels(labelsStr string) map[string]string {
	result := make(map[string]string)
	if labelsStr == "" {
		return result
	}
	for _, pair := range strings.Split(labelsStr, ",") {
		pair = strings.TrimSpace(pair)
		if idx := strings.Index(pair, "="); idx > 0 {
			result[strings.TrimSpace(pair[:idx])] = strings.TrimSpace(pair[idx+1:])
		}
	}
	return result
}

// Taint represents a Kubernetes taint.
type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// parseTaints parses "key=value:effect,..." into a slice of Taints.
func parseTaints(taintsStr string) []Taint {
	var result []Taint
	if taintsStr == "" {
		return result
	}
	for _, t := range strings.Split(taintsStr, ",") {
		t = strings.TrimSpace(t)
		if colonIdx := strings.LastIndex(t, ":"); colonIdx > 0 {
			kv := t[:colonIdx]
			effect := strings.TrimSpace(t[colonIdx+1:])
			key, value := kv, ""
			if eqIdx := strings.Index(kv, "="); eqIdx > 0 {
				key = strings.TrimSpace(kv[:eqIdx])
				value = strings.TrimSpace(kv[eqIdx+1:])
			}
			result = append(result, Taint{Key: key, Value: value, Effect: effect})
		}
	}
	return result
}
