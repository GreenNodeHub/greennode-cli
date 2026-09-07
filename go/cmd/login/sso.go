package login

import (
	"fmt"
	"net/url"
)

// stripRedirectURI rewrites the redirectUri query param of a user-supplied SSO
// authorize URL to the loopback listener's RedirectURI (so the SSO provider
// redirects back to the CLI), and DISCARDS any clientId the link carried—the
// CLI authenticates under its own baked public client (ClientIDForEnv(iamEnv)),
// resolved by the caller, NOT the link's client. If defaultClientID is non-empty
// it is set as clientId (the baked client the token exchange + refresh both use);
// if empty, clientId is left unset and BuildAuthorizeURL will set it from
// cfg.ClientID downstream. All other query params are preserved. Returns an
// error if ssoURL is not a valid absolute URL.
func stripRedirectURI(ssoURL, ourRedirect, defaultClientID string) (string, error) {
	u, err := url.Parse(ssoURL)
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("sso URL is not valid: %q", ssoURL)
	}
	q := u.Query()
	q.Set("redirectUri", ourRedirect)
	q.Del("clientId") // discard the link's client; CLI uses its own baked public client
	if defaultClientID != "" {
		q.Set("clientId", defaultClientID)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
