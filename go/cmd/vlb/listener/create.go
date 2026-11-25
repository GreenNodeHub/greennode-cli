package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new listener",
	RunE:  runCreate,
}

func init() {
	addLBFlag(createCmd)
	f := createCmd.Flags()
	f.String("name", "", "Listener name (required)")
	f.String("protocol", "", "Listener protocol: HTTP, HTTPS, TCP or UDP (required)")
	f.Int("protocol-port", 0, "Listener port, 1-65535 (required)")
	f.Int("timeout-client", 50, "Client idle timeout in seconds (1-3600)")
	f.Int("timeout-member", 50, "Member idle timeout in seconds (1-3600)")
	f.Int("timeout-connection", 5, "Connection idle timeout in seconds (1-3600)")
	f.String("default-pool-id", "", "Pool this listener forwards to by default")
	f.String("allowed-cidrs", "", "Allowed CIDRs")
	f.String("blocked-cidrs", "", "Blocked CIDRs")
	f.String("default-action", "", "Default action: accept or drop")
	f.String("certificate-authorities", "", "Certificate IDs, comma-separated (HTTPS only)")
	f.String("default-certificate-authority", "", "Default certificate ID (HTTPS only)")
	f.String("client-certificate", "", "TLS client authentication mode")
	f.StringArray("insert-header", nil,
		"Header to insert, shorthand (headerName=X-Forwarded-For,headerValue=true) or JSON (repeatable)")
	f.String("alpn-protocols", "", "ALPN protocols, comma-separated: http/1.1, http/1.0, h2")
	f.String("tls-security-policy", "", "TLS security policy: COMPATIBLE, MODERN or STRICT")
	f.Bool("dry-run", false, "Validate parameters and print the request body without creating")

	for _, name := range []string{"name", "protocol", "protocol-port"} {
		if err := createCmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("BUG: MarkFlagRequired(%q): %v", name, err))
		}
	}
}

// listenerOptions carries the optional listener fields shared by create and the
// flag-reading code, keeping buildCreateBody's signature manageable.
type listenerOptions struct {
	DefaultPoolID    string
	AllowedCidrs     string
	BlockedCidrs     string
	DefaultAction    string
	CertAuthorities  string
	DefaultCertAuth  string
	ClientCert       string
	InsertHeaders    []string
	ALPNProtocols    string
	TLSSecurityPolcy string
}

// buildCreateBody assembles the CreateListenerRequest. The three timeouts are
// required by the API, so they are always sent; everything else is omitted when
// unset so the backend applies its own defaults.
func buildCreateBody(name, protocol string, port, timeoutClient, timeoutMember, timeoutConnection int,
	opts listenerOptions) (map[string]interface{}, error) {

	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("--protocol-port must be between 1 and 65535, got %d", port)
	}
	for _, t := range []struct {
		flag  string
		value int
	}{
		{"timeout-client", timeoutClient},
		{"timeout-member", timeoutMember},
		{"timeout-connection", timeoutConnection},
	} {
		if t.value < 1 || t.value > 3600 {
			return nil, fmt.Errorf("--%s must be between 1 and 3600 seconds, got %d", t.flag, t.value)
		}
	}

	body := map[string]interface{}{
		"listenerName":         name,
		"listenerProtocol":     protocol,
		"listenerProtocolPort": port,
		"timeoutClient":        timeoutClient,
		"timeoutMember":        timeoutMember,
		"timeoutConnection":    timeoutConnection,
	}

	for key, value := range map[string]string{
		"defaultPoolId":               opts.DefaultPoolID,
		"allowedCidrs":                opts.AllowedCidrs,
		"blockedCidrs":                opts.BlockedCidrs,
		"defaultAction":               opts.DefaultAction,
		"defaultCertificateAuthority": opts.DefaultCertAuth,
		"clientCertificate":           opts.ClientCert,
		"tlsSecurityPolicy":           opts.TLSSecurityPolcy,
	} {
		if value != "" {
			body[key] = value
		}
	}

	if cas := cli.ParseCommaSeparated(opts.CertAuthorities); len(cas) > 0 {
		body["certificateAuthorities"] = cas
	}
	if alpn := cli.ParseCommaSeparated(opts.ALPNProtocols); len(alpn) > 0 {
		body["alpnProtocols"] = alpn
	}

	headers, err := cli.ParseStructFlags("insert-header", opts.InsertHeaders, nil, nil)
	if err != nil {
		return nil, err
	}
	if headers != nil {
		body["insertHeaders"] = headers
	}

	return body, nil
}

// readListenerOptions collects the optional listener flags from a command.
func readListenerOptions(cmd *cobra.Command) listenerOptions {
	f := cmd.Flags()
	var o listenerOptions
	o.DefaultPoolID, _ = f.GetString("default-pool-id")
	o.AllowedCidrs, _ = f.GetString("allowed-cidrs")
	o.BlockedCidrs, _ = f.GetString("blocked-cidrs")
	o.DefaultAction, _ = f.GetString("default-action")
	o.CertAuthorities, _ = f.GetString("certificate-authorities")
	o.DefaultCertAuth, _ = f.GetString("default-certificate-authority")
	o.ClientCert, _ = f.GetString("client-certificate")
	o.InsertHeaders, _ = f.GetStringArray("insert-header")
	o.ALPNProtocols, _ = f.GetString("alpn-protocols")
	o.TLSSecurityPolcy, _ = f.GetString("tls-security-policy")
	return o
}

func runCreate(cmd *cobra.Command, args []string) error {
	lbID, _, err := idsOf(cmd, false)
	if err != nil {
		return err
	}
	f := cmd.Flags()
	name, _ := f.GetString("name")
	protocol, _ := f.GetString("protocol")
	port, _ := f.GetInt("protocol-port")
	timeoutClient, _ := f.GetInt("timeout-client")
	timeoutMember, _ := f.GetInt("timeout-member")
	timeoutConnection, _ := f.GetInt("timeout-connection")
	dryRun, _ := f.GetBool("dry-run")

	body, err := buildCreateBody(name, protocol, port, timeoutClient, timeoutMember, timeoutConnection,
		readListenerOptions(cmd))
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("create", "listener "+name+" on load balancer "+lbID, body)
		return nil
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	result, err := apiClient.Post(vlbclient.Listeners(projectID, lbID), body)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	return outputRaw(cmd, cfg, result)
}
