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

// resolveOutput returns the effective output format, mirroring vserverclient.Output:
// the --output flag, falling back to the configured default, then "json".
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

// Preview widths (in runes) for long table fields.
const (
	keyPreviewLen = 40
	idPreviewLen  = 20
)

// sshKeyTableColumns is the column order shown in table output. The long public key
// is placed last so it never pushes the other columns out of alignment.
var sshKeyTableColumns = []string{"id", "name", "status", "createdAt", "pubKey"}

// keyFieldsToTruncate are the long key fields shortened in non-JSON output.
var keyFieldsToTruncate = map[string]bool{
	"publicKey":  true,
	"pubKey":     true,
	"privateKey": true,
}

// truncateKeyString collapses newlines and shortens a long key to a readable preview.
func truncateKeyString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return formatter.Truncate(s, keyPreviewLen)
}

// truncateKeys returns a deep copy of the response with long key fields shortened,
// so that table/text output stays readable. Full keys remain available via JSON.
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

// transformKeyTable adapts an SSH key list for table output: it shortens the id and
// key fields and formats the timestamp compactly.
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

// outputKeyList prints an SSH key list. Table output uses a fixed column order with
// shortened id/key/date fields; text output shortens long keys; JSON shows full data.
func outputKeyList(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	switch resolveOutput(cmd, cfg) {
	case "table":
		return vserverclient.OutputWithColumns(cmd, cfg, transformKeyTable(result), sshKeyTableColumns)
	case "json":
		return outputResult(cmd, cfg, result)
	default:
		return outputResult(cmd, cfg, truncateKeys(result))
	}
}

// outputKeyMutation prints the response of create/import. Key fields are always
// shortened to a preview — the full private key lives in the saved .pem file, and
// the full public key can be retrieved with 'sshkey list --output json'.
func outputKeyMutation(cmd *cobra.Command, cfg *config.Config, result interface{}) error {
	return outputResult(cmd, cfg, truncateKeys(result))
}

// keyData unwraps the SSH key object from a response envelope.
// Handles both {"data": {...}} and a plain object.
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

// findStringField returns the first non-empty string value among the given keys.
func findStringField(obj map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := obj[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// downloadsDir returns the user's Downloads directory, creating it if needed.
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

// savePrivateKey writes the private key to "<name>.pem". It is saved in destDir,
// or in the Downloads directory when destDir is empty. If a file with that name
// already exists, a " (n)" suffix is added (Chrome-style), so an existing key file
// is never overwritten. The file is created with 0600 permissions. It returns the
// absolute path of the written file.
func savePrivateKey(name, content, destDir string) (string, error) {
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

	path := filepath.Join(dir, name+".pem")
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s (%d).pem", name, i))
	}

	// Ensure the key ends with a trailing newline so the .pem is well-formed.
	data := []byte(content)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("could not write private key to %s: %w", path, err)
	}
	return path, nil
}
