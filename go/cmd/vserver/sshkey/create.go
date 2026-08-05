package sshkey

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SSH key pair",
	Long: `Create a new SSH key pair. The server generates both the public and
private key. The private key is returned only once and is saved as a
"<name>.pem" file in your Downloads directory (it is never overwritten — a
" (n)" suffix is added if a file with that name already exists).`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "SSH key name (required)")
	f.String("output-dir", "", "Directory to save the <name>.pem private key (default: Downloads)")
	if err := createCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", "name", err))
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	if name == "" {
		return fmt.Errorf("flag --name is required")
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
		fmt.Sprintf("/v2/%s/sshKeys", projectID),
		map[string]interface{}{"name": name},
	)
	if err != nil {
		return fmt.Errorf("failed to create SSH key %q: %w", name, err)
	}

	// Persist the private key to a .pem file (Downloads by default).
	if data := keyData(result); data != nil {
		privateKey := findStringField(data, "privateKey", "private_key", "privatekey", "priKey")
		if privateKey != "" {
			path, saveErr := savePrivateKey(name, privateKey, outputDir)
			if saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: SSH key created but failed to save private key: %v\n", saveErr)
			} else {
				fmt.Fprintf(os.Stderr, "Private key saved to: %s\n", path)
				fmt.Fprintf(os.Stderr, "Keep this file safe — the private key cannot be retrieved again.\n")
				fmt.Fprintf(os.Stderr, "Tip: restrict its permissions with 'chmod 600 %q'.\n", path)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Warning: no private key was returned by the API; nothing was saved.")
		}
	}

	fmt.Fprintln(os.Stderr, "Run 'grn vserver sshkey list --output json' to see the full public key.")

	return outputKeyMutation(cmd, cfg, result)
}
