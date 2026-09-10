package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// trustedEndpointDomains limits reusable bearer-token exposure.
var trustedEndpointDomains = []string{"vngcloud.vn", "greennode.ai"}

// IsTrustedEndpoint accepts built-in endpoints and trusted domains.
func IsTrustedEndpoint(endpointURL string) bool {
	if endpointURL == "" {
		return true
	}
	u, err := url.Parse(endpointURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	for _, d := range trustedEndpointDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// CheckEndpoint warns on untrusted TLS hosts; insecure untrusted hosts require opt-in.
func CheckEndpoint(endpointURL string, noVerifySSL, allowUntrusted bool) error {
	if endpointURL == "" {
		return nil
	}
	u, err := url.Parse(endpointURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("endpoint must be an absolute HTTP(S) URL without user info or a fragment")
	}
	if IsTrustedEndpoint(endpointURL) {
		return nil
	}
	host := u.Hostname()

	noTLS := noVerifySSL || strings.EqualFold(u.Scheme, "http")
	if noTLS && !allowUntrusted {
		reason := "plain HTTP"
		if noVerifySSL {
			reason = "--no-verify-ssl"
		}
		return fmt.Errorf(
			"refusing to send your IAM bearer token to untrusted host %q over an unprotected connection (%s): the token could be captured and replayed. Re-run with --allow-untrusted-endpoint if you really intend this",
			host, reason)
	}

	fmt.Fprintf(os.Stderr,
		"Warning: --endpoint-url %q is outside the trusted domains (%s). grn will send your IAM bearer token to this host, and a bearer token can be replayed. Only use endpoints you trust.\n",
		host, strings.Join(trustedEndpointDomains, ", "))
	return nil
}
