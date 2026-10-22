package sshkey

import (
	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/redact"
	"github.com/spf13/cobra"
)

func requestKey(c *client.GreennodeClient, method, path string, query map[string]string, body any) (any, error) {
	response, err := c.RequestWithStatusNoRetrySensitive(method, path, query, body)
	return response.Data, err
}

func keyOutput(cmd *cobra.Command, data any) any {
	show, _ := cmd.Flags().GetBool("show-secret")
	if show {
		return data
	}
	return redactKeyOutput(data)
}

func redactKeyOutput(data any) any {
	switch value := data.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if key != "publicKey" && key != "pubKey" && redact.IsSensitiveKey(key) {
				out[key] = redact.Value
			} else {
				out[key] = redactKeyOutput(item)
			}
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = redactKeyOutput(item)
		}
		return out
	default:
		return data
	}
}
