package volume

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
	return vserverclient.Output(cmd, cfg, transformVolumeResult(data))
}

var volumeRemoveKeys = map[string]bool{
	"zone":       true,
	"product":    true,
	"bootIndex":  true,
	"updatedAt":  true,
	"volumeType": true,
}

func transformVolume(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		if !volumeRemoveKeys[k] {
			result[k] = v
		}
	}
	return result
}

func transformVolumeList(items []interface{}) []interface{} {
	out := make([]interface{}, len(items))
	for i, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			out[i] = transformVolume(obj)
		} else {
			out[i] = item
		}
	}
	return out
}

func transformVolumeResult(result interface{}) interface{} {
	switch v := result.(type) {
	case map[string]interface{}:
		if data, ok := v["data"].(map[string]interface{}); ok {
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["data"] = transformVolume(data)
			return out
		}
		if listData, ok := v["listData"].([]interface{}); ok {
			out := make(map[string]interface{}, len(v))
			for k, val := range v {
				out[k] = val
			}
			out["listData"] = transformVolumeList(listData)
			return out
		}
		return transformVolume(v)
	case []interface{}:
		return transformVolumeList(v)
	}
	return result
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func suggestZones(apiClient *client.GreennodeClient, projectID string) error {
	return vserverclient.SuggestZoneIDs(apiClient, projectID)
}
