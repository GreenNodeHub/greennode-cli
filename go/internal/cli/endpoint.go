package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// trustedEndpointDomains are the domains grn's own services live under. A
// request to a host outside these is flagged because grn sends a reusable IAM
// bearer token with every request (see CheckEndpoint / SEC-08).
var trustedEndpointDomains = []string{"vngcloud.vn", "greennode.ai"}

// IsTrustedEndpoint reports whether endpointURL targets a host within a trusted
// domain. An empty value means no --endpoint-url override was given (the
// built-in region endpoint is used), which is trusted.
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

// CheckEndpoint enforces the endpoint-safety policy for --endpoint-url. grn
// authenticates against the real IAM and sends the resulting reusable bearer
// token to whatever host --endpoint-url names, so:
//   - trusted host (or no override): allowed silently.
//   - untrusted host over TLS (https, cert verified): a warning is printed.
//   - untrusted host without TLS protection (plain http, or --no-verify-ssl):
//     blocked with an error unless allowUntrusted is set, because the token can
//     be captured (MITM) and replayed.
//
// It returns a non-nil error only for the blocked case.
func CheckEndpoint(endpointURL string, noVerifySSL, allowUntrusted bool) error {
	if IsTrustedEndpoint(endpointURL) {
		return nil
	}
	u, _ := url.Parse(endpointURL)
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
