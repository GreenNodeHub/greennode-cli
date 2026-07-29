package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrStateMismatch is returned by Login when the callback's appState does not
// equal the nonce Login generated (CSRF / stale-callback guard).
var ErrStateMismatch = errors.New("login: state mismatch")

// Token is the full in-memory result of Login: the live access token plus the
// refresh fields. Only the refresh fields are persisted (via store); the access
// token is caller-held and ephemeral.
type Token struct {
	AccessToken  string
	TokenType    string // e.g. "Bearer"; from the /v2 body, defaulted if empty
	RefreshToken string // empty if IAM returned none (partial-success)
	ExpiresAt    time.Time
}

// openBrowser is the browser-open seam. Tests replace it with a no-op. The
// default tries os/exec openers and prints the URL on failure (see below).
var openBrowser = defaultOpenBrowser

func defaultOpenBrowser(rawurl string) error {
	return browserRunner()(rawurl) // platform-specific, defined in opener.go
}

// Login runs the full PKCE authorization-code flow against IAM and returns the
// minted token. It persists the refresh token (0600) to storePath when one is
// present. ctx bounds the whole browser wait. On failure nothing is persisted.
func Login(ctx context.Context, cfg Config, storePath string) (Token, error) {
	pair, err := Generate()
	if err != nil {
		return Token{}, fmt.Errorf("login: pkce generate: %w", err)
	}
	nonce, err := randomNonce()
	if err != nil {
		return Token{}, fmt.Errorf("login: nonce: %w", err)
	}

	l, err := NewListener()
	if err != nil {
		return Token{}, fmt.Errorf("login: listener: %w", err)
	}
	defer l.Close()

	authURL := cfg.BuildAuthorizeURL(l.RedirectURI(), pair.Challenge, nonce)

	// Open the browser best-effort; on failure print the URL and keep waiting
	// (headless/SSH users can complete the flow manually).
	if oerr := openBrowser(authURL); oerr != nil {
		fmt.Fprintf(noisyStderr, "Could not open a browser automatically.\nOpen this URL to log in:\n  %s\n", authURL)
	}

	code, state, serr := l.Serve(ctx)
	if serr != nil {
		return Token{}, serr
	}
	if state != nonce {
		return Token{}, ErrStateMismatch
	}

	tc := New(30 * time.Second)
	resp, errE, err := tc.ExchangeCode(ctx, cfg.TokenURL, ExchangeParams{
		Code:         code,
		RedirectURI:  l.RedirectURI(),
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		CodeVerifier: pair.Verifier,
	})
	if err != nil {
		return Token{}, fmt.Errorf("login: token exchange: %w", err)
	}
	if errE != nil {
		return Token{}, fmt.Errorf("login: token exchange: %w", errE)
	}

	accessToken, err := cfg.AccessTokenFrom(resp.Raw)
	if err != nil {
		return Token{}, fmt.Errorf("login: %w", err)
	}

	tok, refreshToken, err := decodeTokenBody(resp.Raw, accessToken)
	if err != nil {
		// access_token parsed fine but refresh/expires decode failed — partial.
		return Token{AccessToken: accessToken, TokenType: "Bearer"}, fmt.Errorf("login: token body decode: %w", err)
	}

	if refreshToken != "" {
		if err := Save(storePath, StoredToken{RefreshToken: refreshToken, ExpiresAt: tok.ExpiresAt}); err != nil {
			return Token{}, fmt.Errorf("login: persist: %w", err)
		}
	} else {
		fmt.Fprintf(noisyStderr, "Warning: IAM returned no refresh_token — re-login needed after expiry.\n")
	}
	return tok, nil
}

// tokenBody is the subset of the /v2 response Login decodes (snake_case).
type tokenBody struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func decodeTokenBody(raw json.RawMessage, accessToken string) (Token, string, error) {
	var b tokenBody
	if err := json.Unmarshal(raw, &b); err != nil {
		return Token{}, "", fmt.Errorf("decode token body: %w", err)
	}
	tok := Token{
		AccessToken:  accessToken, // already fail-loud-validated
		TokenType:    b.TokenType,
		RefreshToken: b.RefreshToken,
	}
	if b.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	if b.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(b.ExpiresIn) * time.Second).UTC()
	}
	return tok, b.RefreshToken, nil
}

// randomNonce returns a 256-bit random base64url string for the appState.
func randomNonce() (string, error) {
	p, err := Generate() // reuse the 256-bit base64url generator
	if err != nil {
		return "", err
	}
	return p.Verifier, nil
}
