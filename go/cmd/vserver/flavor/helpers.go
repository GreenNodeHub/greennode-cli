package flavor

import (
	"github.com/spf13/cobra"
	"github.com/vngcloud/greennode-cli/internal/client"
	"github.com/vngcloud/greennode-cli/internal/config"
	"github.com/vngcloud/greennode-cli/internal/vserverclient"
)

func createClient(cmd *cobra.Command) (*client.GreennodeClient, *config.Config, error) {
	return vserverclient.BuildClient(cmd)
}

func getProjectID(cfg *config.Config) (string, error) {
	return vserverclient.ProjectID(cfg)
}

func outputResult(cmd *cobra.Command, cfg *config.Config, data interface{}) error {
	return vserverclient.Output(cmd, cfg, data)
}

func transformFamilies(result interface{}) interface{} {
	transform := func(item interface{}) interface{} {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return item
		}
		out := map[string]interface{}{
			"name":        obj["key"],
			"description": obj["description"],
		}
		if cond, ok := obj["condition"].(map[string]interface{}); ok {
			out["codes"] = cond["codes"]
		} else {
			out["codes"] = []interface{}{}
		}
		return out
	}

	switch v := result.(type) {
	case []interface{}:
		transformed := make([]interface{}, len(v))
		for i, item := range v {
			transformed[i] = transform(item)
		}
		return transformed
	case map[string]interface{}:
		if items, ok := v["data"].([]interface{}); ok {
			transformed := make([]interface{}, len(items))
			for i, item := range items {
				transformed[i] = transform(item)
			}
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["data"] = transformed
			return out
		}
	}
	return result
}

func transformCodes(result interface{}) interface{} {
	transform := func(item interface{}) interface{} {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return item
		}
		return map[string]interface{}{
			"name":        obj["key"],
			"description": obj["description"],
		}
	}

	switch v := result.(type) {
	case []interface{}:
		transformed := make([]interface{}, len(v))
		for i, item := range v {
			transformed[i] = transform(item)
		}
		return transformed
	case map[string]interface{}:
		if items, ok := v["data"].([]interface{}); ok {
			transformed := make([]interface{}, len(items))
			for i, item := range items {
				transformed[i] = transform(item)
			}
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["data"] = transformed
			return out
		}
	}
	return result
}

// dropField removes a field from every object in the response (bare array or {"data":[...]} envelope).
func dropField(result interface{}, field string) interface{} {
	deleteFromItems := func(items []interface{}) {
		for _, item := range items {
			if obj, ok := item.(map[string]interface{}); ok {
				delete(obj, field)
			}
		}
	}

	switch v := result.(type) {
	case []interface{}:
		deleteFromItems(v)
	case map[string]interface{}:
		if items, ok := v["data"].([]interface{}); ok {
			deleteFromItems(items)
		} else {
			delete(v, field)
		}
	}
	return result
}

// extractStringSlice converts an API response to a []string for shell completion.
// Handles both a top-level []interface{} and a map with a "data" key.
func extractStringSlice(result interface{}) []string {
	var items []interface{}

	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		if d, ok := v["data"].([]interface{}); ok {
			items = d
		}
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		switch s := item.(type) {
		case string:
			out = append(out, s)
		case map[string]interface{}:
			// try common name fields
			for _, key := range []string{"name", "code", "id", "value"} {
				if val, ok := s[key].(string); ok {
					out = append(out, val)
					break
				}
			}
		}
	}
	return out
}
