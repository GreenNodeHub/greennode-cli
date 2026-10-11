package login

import (
	"testing"
)

// TestExtractIdpHint_PullsIdpIdFromConsoleLink: the real user link is a console
// SSO deep-link of the form
// https://dashboard.console.vngcloud.vn/login?type=idp&idpId=<uuid>&clientId=<baked>&redirectUri=<loopback>&...
// The SSO model no longer uses that link as the authorize URL — it extracts
// idpId (the IdP selector) and discards the rest (host/path/redirectUri/clientId
// are all regenerated on the canonical IAM authorize endpoint).
func TestExtractIdpHint_PullsIdpIdFromConsoleLink(t *testing.T) {
	in := "https://dashboard.console.vngcloud.vn/login?type=idp&idpId=b1d3f049-9376-4dc6-90b0-0d5facd6b8eb&clientId=09b427e9-3b45-437b-a3a6-bf9f7fd24185&redirectUri=http://127.0.0.1:54250/callback&codeChallenge=x&responseType=code&scope=openid&appState=y"
	got, err := extractIdpHint(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "b1d3f049-9376-4dc6-90b0-0d5facd6b8eb" {
		t.Errorf("idpId=%q, want the UUID from the link", got)
	}
}

// TestExtractIdpHint_PullsIdpIdFromPlainAuthorizeURL: a user may paste a bare
// authorize URL that already carries idpId. Still extracted.
func TestExtractIdpHint_PullsIdpIdFromPlainAuthorizeURL(t *testing.T) {
	in := "https://signin.vngcloud.vn/ap/auth?idpId=abc-123"
	got, err := extractIdpHint(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "abc-123" {
		t.Errorf("idpId=%q, want abc-123", got)
	}
}

// TestExtractIdpHint_EmptyWhenNoIdpId: a link without idpId yields an empty
// hint (the caller then runs plain pkce against the canonical endpoint). No
// error — "no hint" is a valid outcome, not a failure.
func TestExtractIdpHint_EmptyWhenNoIdpId(t *testing.T) {
	in := "https://dashboard.console.vngcloud.vn/login?type=idp&clientId=x&redirectUri=http://127.0.0.1:1/callback"
	got, err := extractIdpHint(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("idpId=%q, want empty (no idpId in link)", got)
	}
}

// TestExtractIdpHint_ErrorsOnMalformed: a non-absolute / unparseable link is an
// error (the caller surfaces it). Same absolute-URL guard the old
// stripRedirectURI enforced.
func TestExtractIdpHint_ErrorsOnMalformed(t *testing.T) {
	cases := []string{"", "not a url", "https://", "://nohost"}
	for _, in := range cases {
		if _, err := extractIdpHint(in); err == nil {
			t.Errorf("expected error for %q, got nil", in)
		}
	}
}
