package listener

import (
	"fmt"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/vlbclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a listener",
	Long: "Update a listener's timeouts, default pool, CIDR rules or TLS settings. " +
		"The API requires the full object, so unspecified fields are read from the " +
		"current listener and sent back unchanged.",
	RunE: runUpdate,
}

// updatableStringFields maps a listener flag to its UpdateListenerRequest key
// and the ListenerDto key its current value is read from. The listener's name,
// protocol and port cannot be changed after creation and are absent here.
var updatableStringFields = []struct {
	flag    string
	bodyKey string
	dtoKey  string
}{
	{"default-pool-id", "defaultPoolId", "defaultPoolId"},
	{"allowed-cidrs", "allowedCidrs", "allowedCidrs"},
	{"blocked-cidrs", "blockedCidrs", "blockedCidrs"},
	{"default-action", "defaultAction", "defaultAction"},
	{"default-certificate-authority", "defaultCertificateAuthority", "defaultCertificateAuthority"},
	{"client-certificate", "clientCertificate", "clientCertificateAuthentication"},
	{"tls-security-policy", "tlsSecurityPolicy", "tlsSecurityPolicy"},
}

// tlsOnlyFields are rejected on a listener that is not HTTPS.
var tlsOnlyFields = map[string]bool{
	"certificateAuthorities":      true,
	"defaultCertificateAuthority": true,
	"clientCertificate":           true,
	"tlsSecurityPolicy":           true,
	"alpnProtocols":               true,
}

// currentProtocol reads the listener's protocol, which cannot be changed after
// creation and therefore decides which fields may be sent.
func currentProtocol(current map[string]interface{}) string {
	p, _ := current["protocol"].(string)
	return p
}

var updatableTimeouts = []struct {
	flag string
	key  string
}{
	{"timeout-client", "timeoutClient"},
	{"timeout-member", "timeoutMember"},
	{"timeout-connection", "timeoutConnection"},
}

func init() {
	addLBFlag(updateCmd)
	addListenerFlag(updateCmd)
	addUpdateFlags(updateCmd.Flags())
	updateCmd.Flags().Bool("dry-run", false, "Print the request body without applying the update")
}

// addUpdateFlags declares the body-shaping flags. It is separate from init() so
// tests can build an identical flag set without mutating the shared command.
func addUpdateFlags(f *pflag.FlagSet) {
	f.Int("timeout-client", 0, "Client idle timeout in seconds (1-3600)")
	f.Int("timeout-member", 0, "Member idle timeout in seconds (1-3600)")
	f.Int("timeout-connection", 0, "Connection idle timeout in seconds (1-3600)")
	f.String("default-pool-id", "", "Pool this listener forwards to by default")
	f.String("allowed-cidrs", "", "Allowed CIDRs")
	f.String("blocked-cidrs", "", "Blocked CIDRs")
	f.String("default-action", "", "Default action: accept or drop")
	f.String("certificate-authorities", "", "Certificate IDs, comma-separated (HTTPS only)")
	f.String("default-certificate-authority", "", "Default certificate ID (HTTPS only)")
	f.String("client-certificate", "", "TLS client authentication mode")
	f.StringArray("insert-header", nil,
		"Header to insert, shorthand (headerName=X-Forwarded-For,headerValue=true) or JSON (repeatable); replaces the current list")
	f.String("alpn-protocols", "", "ALPN protocols, comma-separated: http/1.1, http/1.0, h2")
	f.String("tls-security-policy", "", "TLS security policy: COMPATIBLE, MODERN or STRICT")
}

// updateFlagNames are every flag that contributes to the request body; the
// command errors when the user set none of them.
func updateFlagNames() []string {
	names := []string{"certificate-authorities", "insert-header", "alpn-protocols"}
	for _, t := range updatableTimeouts {
		names = append(names, t.flag)
	}
	for _, s := range updatableStringFields {
		names = append(names, s.flag)
	}
	return names
}

// buildUpdateBody assembles the UpdateListenerRequest by seeding every field
// from the listener's current state and overlaying only what the user set.
// `current` is the unwrapped ListenerDto from GET.
func buildUpdateBody(current map[string]interface{}, f *pflag.FlagSet) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	// The three timeouts are required by the API; carry the current values
	// through whenever the user did not override them.
	for _, t := range updatableTimeouts {
		if f.Changed(t.flag) {
			v, _ := f.GetInt(t.flag)
			if v < 1 || v > 3600 {
				return nil, fmt.Errorf("--%s must be between 1 and 3600 seconds, got %d", t.flag, v)
			}
			body[t.key] = v
			continue
		}
		if v, ok := current[t.key]; ok && v != nil {
			body[t.key] = v
		}
	}
	for _, t := range updatableTimeouts {
		if _, ok := body[t.key]; !ok {
			return nil, fmt.Errorf("could not determine the listener's current %s; pass --%s explicitly", t.key, t.flag)
		}
	}

	// TLS settings are rejected outright on a non-HTTPS listener ("Cannot
	// specify client certificate, certificate authorities, default certificate
	// authority, or tls security policy for http listener"). The service still
	// reports them on such a listener — as nulls and empty arrays — so echoing
	// its own state back is what trips the check. Skip them entirely unless the
	// listener speaks TLS, or the user explicitly asked for one.
	isTLS := currentProtocol(current) == "HTTPS"

	for _, s := range updatableStringFields {
		if f.Changed(s.flag) {
			v, _ := f.GetString(s.flag)
			body[s.bodyKey] = v
			continue
		}
		if tlsOnlyFields[s.bodyKey] && !isTLS {
			continue
		}
		if v, ok := current[s.dtoKey]; ok && v != nil && v != "" {
			body[s.bodyKey] = v
		}
	}

	for _, l := range []struct {
		flag, bodyKey, dtoKey string
	}{
		{"certificate-authorities", "certificateAuthorities", "certificateAuthorities"},
		{"alpn-protocols", "alpnProtocols", "alpnProtocols"},
	} {
		if f.Changed(l.flag) {
			raw, _ := f.GetString(l.flag)
			body[l.bodyKey] = cli.ParseCommaSeparated(raw)
			continue
		}
		if !isTLS {
			continue
		}
		// An empty list is the service's way of saying "none configured";
		// sending it back adds nothing and reads as specifying the field.
		if v, ok := current[l.dtoKey].([]interface{}); ok && len(v) > 0 {
			body[l.bodyKey] = v
		}
	}

	if f.Changed("insert-header") {
		raw, _ := f.GetStringArray("insert-header")
		headers, err := cli.ParseStructFlags("insert-header", raw, nil, nil)
		if err != nil {
			return nil, err
		}
		if headers == nil {
			headers = []map[string]interface{}{}
		}
		body["insertHeaders"] = headers
	} else if v, ok := current["insertHeaders"]; ok && v != nil {
		body["insertHeaders"] = v
	}

	return body, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	lbID, listenerID, err := idsOf(cmd, true)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	anyChanged := false
	for _, name := range updateFlagNames() {
		if cmd.Flags().Changed(name) {
			anyChanged = true
			break
		}
	}
	if !anyChanged {
		return fmt.Errorf("nothing to update: provide at least one of --%s",
			joinFlags(updateFlagNames()))
	}

	apiClient, cfg, projectID, err := resolve(cmd)
	if err != nil {
		return err
	}

	response, err := apiClient.Get(vlbclient.Listener(projectID, lbID, listenerID), nil)
	if err != nil {
		return fmt.Errorf("failed to fetch listener %s: %w", listenerID, err)
	}

	body, err := buildUpdateBody(unwrapData(response), cmd.Flags())
	if err != nil {
		return err
	}

	if dryRun {
		cli.PrintDryRun("update", "listener "+listenerID, body)
		return nil
	}

	result, err := apiClient.Put(vlbclient.Listener(projectID, lbID, listenerID), body)
	if err != nil {
		return fmt.Errorf("failed to update listener %s: %w", listenerID, err)
	}

	return outputRaw(cmd, cfg, result)
}

func joinFlags(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", --"
		}
		out += n
	}
	return out
}
