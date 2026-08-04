// Package login implements a native PKCE authorization-code login against VNG
// IAM for the greennode CLI. The CLI is the OAuth client: it generates the PKCE
// verifier/challenge, binds a loopback redirect listener, opens the browser to
// IAM's signin page, exchanges the authorization code at IAM's /v2 token
// endpoint (sending client_secret via Basic only when configured), and persists
// the refresh token (0600) for reuse across subsequent CLI invocations.
//
// The access token is never persisted; only the refresh token and its expiry.
//
// The IAM-leg code (camelCase authorize URL, client_secret_basic token
// exchange, fail-loud access_token extraction) is lifted from
// agent-core-gateway's internal/oauth/iamidp and internal/clients/idpoauth; the
// authorization-server machinery (sessions, cookies, signed state, discovery)
// is intentionally not lifted — the CLI needs only the client leg.
package login
