package login

import (
	"net/url"
	"strings"
	"testing"
)

func cfgForTest() Config {
	return Config{
		AuthorizeURL: "https://signin.vngcloud.vn/ap/auth",
		TokenURL:     "https://iam.api.vngcloud.vn/accounts-api/v2/auth/token",
		ClientID:     "cli-client-id",
		Scopes:       []string{"openid"},
	}
}

func TestBuildAuthorizeURL_BaselineCamelCase(t *testing.T) {
	t.Parallel()
	c := cfgForTest()
	got := c.BuildAuthorizeURL("http://127.0.0.1:8080/callback", "Z0iBB67yj6V", "state-nonce")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v\nurl=%s", err, got)
	}
	q := u.Query()
	cases := map[string]string{
		"clientId":            "cli-client-id",
		"responseType":        "code",
		"codeChallenge":       "Z0iBB67yj6V",
		"codeChallengeMethod": "S256",
		"appState":            "state-nonce",
		"redirectUri":         "http://127.0.0.1:8080/callback",
		"scope":               "openid",
	}
	for k, want := range cases {
		if got := q.Get(k); got != want {
			t.Errorf("query[%s]=%q, want %q", k, got, want)
		}
	}
	if u.Scheme != "https" || u.Host != "signin.vngcloud.vn" || u.Path != "/ap/auth" {
		t.Errorf("base URL changed: %s", got)
	}
}

func TestBuildAuthorizeURL_ScopeOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	c := cfgForTest()
	c.Scopes = nil
	got := c.BuildAuthorizeURL("http://127.0.0.1:8080/callback", "ch", "st")
	u, _ := url.Parse(got)
	if _, ok := u.Query()["scope"]; ok {
		t.Errorf("scope must be omitted when Scopes is empty; url=%s", got)
	}
}

func TestBuildAuthorizeURL_RejectsSnakeCaseSynonyms(t *testing.T) {
	t.Parallel()
	c := cfgForTest()
	got := c.BuildAuthorizeURL("http://127.0.0.1:8080/callback", "ch", "st")
	// IAM uses camelCase; snake_case params must NOT appear.
	for _, bad := range []string{"client_id", "response_type", "code_challenge", "code_challenge_method", "state", "redirect_uri"} {
		if strings.Contains(got, bad+"=") || strings.Contains(got, bad+"&") {
			t.Errorf("snake_case param %q must not appear; url=%s", bad, got)
		}
	}
}

func TestAccessTokenFrom_ReturnsToken(t *testing.T) {
	t.Parallel()
	c := cfgForTest()
	body := []byte(`{"access_token":"abc.def.ghi","token_type":"Bearer","expires_in":3600}`)
	got, err := c.AccessTokenFrom(body)
	if err != nil {
		t.Fatalf("AccessTokenFrom: %v", err)
	}
	if got != "abc.def.ghi" {
		t.Errorf("got %q, want abc.def.ghi", got)
	}
}

func TestAccessTokenFrom_FailLoudOnMissing(t *testing.T) {
	t.Parallel()
	c := cfgForTest()
	for _, body := range [][]byte{
		[]byte(`{"token_type":"Bearer"}`), // no access_token
		[]byte(`{"access_token":""}`),     // empty access_token
		[]byte(`{}`),                      // empty object
		[]byte(`not json at all`),         // not JSON
	} {
		if _, err := c.AccessTokenFrom(body); err == nil {
			t.Errorf("expected error for body %q, got nil", string(body))
		}
	}
}
