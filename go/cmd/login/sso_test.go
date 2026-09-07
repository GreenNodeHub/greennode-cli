package login

import (
	"strings"
	"testing"
)

func TestStripRedirectURI_ReplacesRedirectUriAndPreservesOthers(t *testing.T) {
	in := "https://sso.example/ap/auth?clientId=foreign&redirectUri=https://old.example/cb&scope=openid"
	got, err := stripRedirectURI(in, "http://127.0.0.1:54321/callback", "baked-cid")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// q.Encode() percent-encodes the redirectUri value (":" -> "%3A"), so accept
	// either the literal or encoded colon. Mirrors the brief's original robust
	// dual-form check; the controller's corrected test only checked the literal
	// form, which cannot pass against the mandated q.Encode() implementation.
	if !strings.Contains(got, "127.0.0.1:54321") && !strings.Contains(got, "127.0.0.1%3A54321") {
		t.Errorf("redirectUri not rewritten to loopback: %q", got)
	}
	if strings.Contains(got, "old.example") {
		t.Errorf("old redirectUri leaked: %q", got)
	}
	if strings.Contains(got, "foreign") {
		t.Errorf("link's clientId 'foreign' leaked (must be discarded): %q", got)
	}
	if !strings.Contains(got, "clientId=baked-cid") && !strings.Contains(got, "clientId=baked") {
		t.Errorf("baked clientId not set: %q", got)
	}
	if !strings.Contains(got, "scope") {
		t.Errorf("scope not preserved: %q", got)
	}
}

func TestStripRedirectURI_DiscardsLinkClientId_UsesBaked(t *testing.T) {
	in := "https://sso.example/ap/auth?clientId=link-client&redirectUri=https://old/cb"
	got, err := stripRedirectURI(in, "http://127.0.0.1:1/callback", "baked-cid")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if strings.Contains(got, "link-client") {
		t.Errorf("link clientId leaked: %q", got)
	}
	if !strings.Contains(got, "clientId=baked") {
		t.Errorf("baked clientId not set: %q", got)
	}
}

func TestStripRedirectURI_EmptyDefaultClientId_LeavesClientIdUnset(t *testing.T) {
	in := "https://sso.example/ap/auth?clientId=link-client&redirectUri=https://old/cb"
	got, err := stripRedirectURI(in, "http://127.0.0.1:1/callback", "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if strings.Contains(got, "link-client") {
		t.Errorf("link clientId leaked even with empty default: %q", got)
	}
	if strings.Contains(got, "clientId") {
		t.Errorf("clientId should be unset when defaultClientID is empty: %q", got)
	}
}

func TestStripRedirectURI_ErrorsOnMalformed(t *testing.T) {
	cases := []string{"", "not a url", "https://", "://nohost"}
	for _, in := range cases {
		if _, err := stripRedirectURI(in, "http://127.0.0.1:1/callback", "cid"); err == nil {
			t.Errorf("expected error for %q, got nil", in)
		}
	}
}
