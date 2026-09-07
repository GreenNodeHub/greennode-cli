package login

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// simulateBrowser fires an async GET at the flow's AuthURL so the fake IAM's
// 302 redirect lands on the loopback listener — the same shape stubBrowser uses
// for the Login path, but driven by the caller (Prepare/Complete does not open a
// browser itself; the cmd does, or the test simulates it).
func simulateBrowser(t *testing.T, authURL string) {
	t.Helper()
	go func() {
		resp, err := http.Get(authURL)
		if err == nil {
			resp.Body.Close()
		}
	}()
}

func TestPrepare_BindsListenerAndArmsNonce(t *testing.T) {
	cfg := Config{AuthorizeURL: "https://x.example/auth", TokenURL: "https://x.example/token", ClientID: "cid"}
	flow, err := Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer flow.Close()

	if !strings.HasPrefix(flow.RedirectURI(), "http://127.0.0.1:") {
		t.Errorf("RedirectURI=%q, want 127.0.0.1 loopback", flow.RedirectURI())
	}
	u, err := url.Parse(flow.AuthURL())
	if err != nil {
		t.Fatalf("AuthURL not parseable: %v", err)
	}
	q := u.Query()
	if q.Get("codeChallenge") == "" {
		t.Error("AuthURL missing codeChallenge")
	}
	if q.Get("codeChallengeMethod") != "S256" {
		t.Errorf("codeChallengeMethod=%q, want S256", q.Get("codeChallengeMethod"))
	}
	if q.Get("appState") == "" {
		t.Error("AuthURL missing appState (nonce)")
	}
	if q.Get("clientId") != "cid" {
		t.Errorf("clientId=%q, want cid", q.Get("clientId"))
	}
	if q.Get("redirectUri") != flow.RedirectURI() {
		t.Errorf("redirectUri=%q, want %s", q.Get("redirectUri"), flow.RedirectURI())
	}
}

// TestPrepareComplete_EndToEnd asserts Prepare + a caller-driven browser
// redirect + Complete mints the same Token the Login path mints. This is the
// back-compat/regression guarantee for the staged split.
func TestPrepareComplete_EndToEnd(t *testing.T) {
	f := &fakeIAM{
		authCode:  "the-code",
		tokenBody: `{"access_token":"at-123","token_type":"Bearer","refresh_token":"rt-456","expires_in":3600}`,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()
	cfg := newLoginTestCfg(srv.URL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	flow, err := Prepare(ctx, cfg)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer flow.Close()
	simulateBrowser(t, flow.AuthURL())

	tok, err := flow.Complete(ctx)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("AccessToken=%q, want at-123", tok.AccessToken)
	}
	if tok.RefreshToken != "rt-456" {
		t.Errorf("RefreshToken=%q, want rt-456", tok.RefreshToken)
	}
	if tok.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set from expires_in")
	}
}

func TestPrepareComplete_SetAuthorizeURL_OverridesAuthorize(t *testing.T) {
	f := &fakeIAM{
		authCode:  "c",
		tokenBody: `{"access_token":"at","token_type":"Bearer","refresh_token":"rt","expires_in":3600}`,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()
	cfg := newLoginTestCfg(srv.URL, "")

	flow, err := Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer flow.Close()
	flow.SetAuthorizeURL("https://sso.example/auth?clientId=sso-client")
	got, err := url.Parse(flow.AuthURL())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Host != "sso.example" {
		t.Errorf("host=%q, want sso.example (override did not take effect)", got.Host)
	}
	if got.Query().Get("clientId") != "sso-client" {
		t.Errorf("clientId=%q, want sso-client", got.Query().Get("clientId"))
	}
	if got.Query().Get("redirectUri") != flow.RedirectURI() {
		t.Errorf("redirectUri not set to loopback after override")
	}
}

func TestFlow_Close_ReleasesPortWithoutServe(t *testing.T) {
	cfg := Config{AuthorizeURL: "https://x/auth", TokenURL: "https://x/token", ClientID: "cid"}
	flow, err := Prepare(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := flow.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Listener.Close is idempotent; a second Close must be a no-op.
	if err := flow.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestComplete_CtxDeadlinePropagates(t *testing.T) {
	f := &fakeIAM{authCode: "c", tokenBody: `{"access_token":"at","token_type":"Bearer"}`}
	srv := startFakeIAM(t, f)
	defer srv.Close()
	cfg := newLoginTestCfg(srv.URL, "")

	// No browser redirect is ever fired, so Serve blocks until ctx expires.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	flow, err := Prepare(ctx, cfg)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer flow.Close()
	_, err = flow.Complete(ctx)
	if err == nil {
		t.Fatal("expected ctx-deadline error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err=%v, want ctx.DeadlineExceeded", err)
	}
}
