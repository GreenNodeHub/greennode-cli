package certificate

import (
	"fmt"
	"os"
	"strings"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

const (
	typeCA     = "CA"
	typeTLSSSL = "TLS/SSL"
)

var createCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"import"},
	Short:   "Import a certificate",
	Long: "Import a CA or TLS/SSL certificate for use by HTTPS listeners. Prefer the " +
		"--*-file flags: passing PEM material on the command line leaves it in the " +
		"shell history.",
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.String("name", "", "Certificate name (required)")
	f.String("type", "", "Certificate type: CA or TLS/SSL (required)")
	f.String("certificate-file", "", "Path to the PEM certificate")
	f.String("certificate", "", "PEM certificate (prefer --certificate-file)")
	f.String("private-key-file", "", "Path to the PEM private key")
	f.String("private-key", "", "PEM private key (prefer --private-key-file)")
	f.String("certificate-chain-file", "", "Path to the PEM certificate chain")
	f.String("certificate-chain", "", "PEM certificate chain (prefer --certificate-chain-file)")
	f.String("passphrase", "", "Passphrase protecting the private key")
	f.Bool("dry-run", false, "Validate parameters and preview the request without importing")

	for _, name := range []string{"name", "type"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
	for _, pair := range [][2]string{
		{"certificate", "certificate-file"},
		{"private-key", "private-key-file"},
		{"certificate-chain", "certificate-chain-file"},
	} {
		createCmd.MarkFlagsMutuallyExclusive(pair[0], pair[1])
	}
}

// readMaterial returns the inline value, or the contents of path when given.
// Exactly one of the two is set — the flags are mutually exclusive.
func readMaterial(flagName, inline, path string) (string, error) {
	if path == "" {
		return inline, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("--%s-file: %w", flagName, err)
	}
	return string(data), nil
}

// buildCreateBody assembles the ImportCARequest. Empty optional fields are
// omitted rather than sent as empty strings.
func buildCreateBody(name, certType, certificate, privateKey, chain, passphrase string) (map[string]interface{}, error) {
	if certType != typeCA && certType != typeTLSSSL {
		return nil, fmt.Errorf("--type must be %q or %q, got %q", typeCA, typeTLSSSL, certType)
	}
	if strings.TrimSpace(certificate) == "" {
		return nil, fmt.Errorf("a certificate is required: pass --certificate-file or --certificate")
	}
	if certType == typeTLSSSL && strings.TrimSpace(privateKey) == "" {
		return nil, fmt.Errorf("a private key is required for a %s certificate: pass --private-key-file or --private-key", typeTLSSSL)
	}

	body := map[string]interface{}{
		"name":        name,
		"type":        certType,
		"certificate": certificate,
	}
	for key, value := range map[string]string{
		"privateKey":       privateKey,
		"certificateChain": chain,
		"passphrase":       passphrase,
	} {
		if strings.TrimSpace(value) != "" {
			body[key] = value
		}
	}
	return body, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	name, _ := f.GetString("name")
	certType, _ := f.GetString("type")
	passphrase, _ := f.GetString("passphrase")
	dryRun, _ := f.GetBool("dry-run")

	inlineCert, _ := f.GetString("certificate")
	certPath, _ := f.GetString("certificate-file")
	certificate, err := readMaterial("certificate", inlineCert, certPath)
	if err != nil {
		return err
	}

	inlineKey, _ := f.GetString("private-key")
	keyPath, _ := f.GetString("private-key-file")
	privateKey, err := readMaterial("private-key", inlineKey, keyPath)
	if err != nil {
		return err
	}

	inlineChain, _ := f.GetString("certificate-chain")
	chainPath, _ := f.GetString("certificate-chain-file")
	chain, err := readMaterial("certificate-chain", inlineChain, chainPath)
	if err != nil {
		return err
	}

	body, err := buildCreateBody(name, certType, certificate, privateKey, chain, passphrase)
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("import", "certificate "+name, redactedPreview(body))
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.Certificates(projectID), body)
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}

// redactedPreview replaces the secret material with a size summary so a
// --dry-run never prints a private key or passphrase to the terminal.
func redactedPreview(body map[string]interface{}) map[string]interface{} {
	preview := map[string]interface{}{}
	for k, v := range body {
		switch k {
		case "privateKey", "passphrase":
			preview[k] = "<redacted>"
		case "certificate", "certificateChain":
			s, _ := v.(string)
			preview[k] = fmt.Sprintf("<%d bytes of PEM>", len(s))
		default:
			preview[k] = v
		}
	}
	return preview
}
