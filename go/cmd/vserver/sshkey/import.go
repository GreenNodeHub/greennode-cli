package sshkey

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import an existing SSH public key",
	Long: `Import an existing SSH public key. The public key can be supplied
directly with --public-key, or — more conveniently — read from a file
(e.g. ~/.ssh/id_rsa.pub) with --public-key-file.`,
	RunE: runImport,
}

func init() {
	f := importCmd.Flags()
	f.String("name", "", "SSH key name (required)")
	f.String("public-key", "", "SSH public key contents (e.g. 'ssh-rsa AAAA...')")
	f.String("public-key-file", "", "Path to a public key file to read (e.g. ~/.ssh/id_rsa.pub)")

	if err := importCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "name", err))
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	publicKey, _ := cmd.Flags().GetString("public-key")
	publicKeyFile, _ := cmd.Flags().GetString("public-key-file")

	if name == "" {
		return fmt.Errorf("flag --name is required")
	}

	pubKey, err := resolvePublicKey(publicKey, publicKeyFile)
	if err != nil {
		return err
	}

	apiClient, cfg, err := createClient(cmd)
	if err != nil {
		return err
	}

	projectID, err := getProjectID(cfg)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(
		fmt.Sprintf("/v2/%s/sshKeys/import", projectID),
		map[string]interface{}{"name": name, "pubKey": pubKey},
	)
	if err != nil {
		return fmt.Errorf("failed to import SSH key %q: %w", name, err)
	}

	return outputKeyMutation(cmd, cfg, result)
}

// resolvePublicKey returns the public key contents from --public-key-file (if set)
// or --public-key. Exactly one source must be provided. The value is whitespace-trimmed
// and validated to look like an SSH public key.
func resolvePublicKey(publicKey, publicKeyFile string) (string, error) {
	if publicKey == "" && publicKeyFile == "" {
		return "", fmt.Errorf("a public key is required: pass --public-key-file <path> or --public-key <value>")
	}
	if publicKey != "" && publicKeyFile != "" {
		return "", fmt.Errorf("use only one of --public-key or --public-key-file, not both")
	}

	key := publicKey
	if publicKeyFile != "" {
		data, err := os.ReadFile(publicKeyFile)
		if err != nil {
			return "", fmt.Errorf("could not read public key file %s: %w", publicKeyFile, err)
		}
		key = string(data)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("the provided public key is empty")
	}
	if !looksLikePublicKey(key) {
		return "", fmt.Errorf("the provided value does not look like an SSH public key (expected it to start with e.g. 'ssh-rsa', 'ssh-ed25519', or 'ecdsa-sha2-...')")
	}
	return key, nil
}

func looksLikePublicKey(key string) bool {
	for _, prefix := range []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2-", "sk-ssh-", "sk-ecdsa-"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
