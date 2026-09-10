package sshkey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/client"
	"github.com/greennodehub/greennode-cli/internal/config"
	"github.com/greennodehub/greennode-cli/internal/formatter"
	"github.com/greennodehub/greennode-cli/internal/vserverclient"
	"github.com/spf13/cobra"
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

func resolveOutput(cmd *cobra.Command, cfg *config.Config) string {
	output, _ := cmd.Flags().GetString("output")
	if output == "" && cfg != nil {
		output = cfg.Output
	}
	if output == "" {
		output = "json"
	}
	return output
}

const (
	keyPreviewLen = 40
	idPreviewLen  = 20
)

var sshKeyTableColumns = []string{"id", "name", "status", "createdAt", "pubKey"}

var keyFieldsToTruncate = map[string]bool{
	"publicKey":  true,
	"pubKey":     true,
	"privateKey": true,
}

func truncateKeyString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return formatter.Truncate(s, keyPreviewLen)
}

func truncateKeys(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if keyFieldsToTruncate[k] {
				if s, ok := val.(string); ok {
					out[k] = truncateKeyString(s)
					continue
				}
			}
			out[k] = truncateKeys(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = truncateKeys(item)
		}
		return out
	default:
		return v
	}
}

func transformKeyTable(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			switch {
			case keyFieldsToTruncate[k]:
				if s, ok := val.(string); ok {
					out[k] = truncateKeyString(s)
					continue
				}
				out[k] = val
			case k == "id" || k == "uuid":
				if s, ok := val.(string); ok {
					out[k] = formatter.Truncate(s, idPreviewLen)
					continue
				}
				out[k] = val
			case k == "createdAt" || k == "updatedAt":
				if s, ok := val.(string); ok {
					out[k] = formatter.ShortDate(s)
					continue
				}
				out[k] = val
			default:
				out[k] = transformKeyTable(val)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = transformKeyTable(item)
		}
		return out
	default:
		return v
	}
}

func outputKeyList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	result = keyOutput(cmd, result)
	switch resolveOutput(cmd, cfg) {
	case "table":
		return vserverclient.OutputWithColumns(cmd, cfg, transformKeyTable(result), sshKeyTableColumns)
	case "json":
		return outputResult(cmd, cfg, result)
	default:
		return outputResult(cmd, cfg, truncateKeys(result))
	}
}

func outputKeyMutation(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	return outputResult(cmd, cfg, keyOutput(cmd, result))
}

func keyData(result interface{}) map[string]interface{} {
	m, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	if d, ok := m["data"].(map[string]interface{}); ok {
		return d
	}
	return m
}

func findStringField(obj map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := obj[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}
	dir := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create downloads directory %s: %w", dir, err)
	}
	return dir, nil
}

func savePrivateKey(name, content, destDir string) (string, error) {
	if err := validateKeyFileName(name); err != nil {
		return "", err
	}
	dir := destDir
	if dir == "" {
		d, err := downloadsDir()
		if err != nil {
			return "", err
		}
		dir = d
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create output directory %s: %w", dir, err)
	}

	data := []byte(content)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	temporary, err := os.CreateTemp(dir, ".grn-key-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	for i := 0; ; i++ {
		fileName := name + ".pem"
		if i > 0 {
			fileName = fmt.Sprintf("%s (%d).pem", name, i)
		}
		path := filepath.Join(dir, fileName)
		if err := os.Link(temporary.Name(), path); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", fmt.Errorf("could not save private key: %w", err)
		}
		return path, nil
	}
}

func validateKeyFileName(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("SSH key name must be a file name, not a path")
	}
	return nil
}
