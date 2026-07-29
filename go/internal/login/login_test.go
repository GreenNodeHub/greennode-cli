package login

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeIAM stands up an httptest.Server that speaks the authorize redirect and
// the token endpoint. On GET to /authorize it 302-redirects to the redirectUri
// with code+appState; on POST to /token it returns a token JSON body configured
// by the test.
type fakeIAM struct {
	authCode     string
	tokenBody    string // returned by /token
	tokenStatus  int
	omitBasic    bool // when true, test asserts no Basic header here
	sawBasicAuth bool
}

func startFakeIAM(t *testing.T, f *fakeIAM) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			// Redirect back to the listener's callback with code + the state we
			// were given on the authorize URL.
			q := r.URL.Query()
			state := q.Get("appState")
			ru := q.Get("redirectUri")
			http.Redirect(w, r, ru+"?code="+f.authCode+"&appState="+state, http.StatusFound)
		case "/token":
			f.sawBasicAuth = r.Header.Get("Authorization") != ""
			w.Header().Set("Content-Type", "application/json")
			status := f.tokenStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(f.tokenBody))
		default:
			http.NotFound(w, r)
		}
	}))
	return s
}

func newLoginTestCfg(serverURL string, clientSecret string) Config {
	return Config{
		AuthorizeURL: serverURL + "/authorize",
		TokenURL:     serverURL + "/token",
		ClientID:     "cid",
		ClientSecret: clientSecret,
		Scopes:       []string{"openid"},
	}
}

// stubBrowser replaces openBrowser with a fake that simulates the browser
// following the authorize URL's 302 redirect to the loopback callback. It
// fetches the authorize URL with http.Get (the default client follows
// redirects, landing on the listener) in a goroutine so openBrowser returns
// immediately, mirroring a real async browser launch. No real browser opens.
func stubBrowser() func() {
	orig := openBrowser
	openBrowser = func(u string) error {
		go func() {
			resp, err := http.Get(u)
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
	return func() { openBrowser = orig }
}

func TestLogin_EndToEnd_RefreshTokenPersisted(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	f := &fakeIAM{
		authCode:  "the-code",
		tokenBody: `{"access_token":"at-123","token_type":"Bearer","refresh_token":"rt-456","expires_in":3600}`,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tok, err := Login(ctx, cfg, storePath)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("AccessToken=%q, want at-123", tok.AccessToken)
	}
	if tok.RefreshToken != "rt-456" {
		t.Errorf("RefreshToken=%q, want rt-456", tok.RefreshToken)
	}
	if tok.TokenType == "" {
		t.Error("TokenType should be set")
	}
	if tok.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set from expires_in")
	}

	// File persisted at 0600 with the refresh token.
	stored, err := Load(storePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.RefreshToken != "rt-456" {
		t.Errorf("stored RefreshToken=%q, want rt-456", stored.RefreshToken)
	}
	if fi, _ := os.Stat(storePath); fi != nil && fi.Mode().Perm() != 0600 {
		t.Errorf("file perm=%o, want 0600", fi.Mode().Perm())
	}
}

func TestLogin_ConfidentialClient_SendsBasic(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	f := &fakeIAM{
		authCode:  "c",
		tokenBody: `{"access_token":"at","token_type":"Bearer","refresh_token":"rt","expires_in":3600}`,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "the-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := Login(ctx, cfg, storePath); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !f.sawBasicAuth {
		t.Error("expected Basic auth header at /token when ClientSecret is set")
	}
}

func TestLogin_PublicClient_NoBasic(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	f := &fakeIAM{
		authCode:  "c",
		tokenBody: `{"access_token":"at","token_type":"Bearer","refresh_token":"rt","expires_in":3600}`,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "") // public client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := Login(ctx, cfg, storePath); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if f.sawBasicAuth {
		t.Error("expected NO Basic auth at /token for a public client")
	}
}

func TestLogin_StateMismatchReturnsErrStateMismatch(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	// fakeIAM redirects back with appState=<state received on authorize URL>, so
	// to force a mismatch we make the listener's held nonce differ from what
	// comes back. We do that by having the IAM redirect override appState.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authorize" {
			ru := r.URL.Query().Get("redirectUri")
			// Send a DIFFERENT appState than the one Login generated.
			http.Redirect(w, r, ru+"?code=c&appState=WRONG", http.StatusFound)
			return
		}
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer"}`))
		}
	}))
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Login(ctx, cfg, storePath)
	if !strings.Contains(err.Error(), "state") && err != ErrStateMismatch {
		t.Errorf("err=%v, want ErrStateMismatch", err)
	}
	// Nothing persisted on mismatch.
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Errorf("expected no token file on state mismatch; stat err=%v", err)
	}
}

func TestLogin_PartialSuccessWhenNoRefreshToken(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	// Capture the spec'd no-refresh-token warning so test output stays pristine
	// AND verify Login actually emits it.
	origStderr := noisyStderr
	var warnBuf bytes.Buffer
	noisyStderr = &warnBuf
	defer func() { noisyStderr = origStderr }()

	f := &fakeIAM{
		authCode:  "c",
		tokenBody: `{"access_token":"at-only","token_type":"Bearer","expires_in":3600}`, // no refresh_token
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tok, err := Login(ctx, cfg, storePath)
	if err != nil {
		t.Fatalf("Login (partial): %v", err)
	}
	if tok.AccessToken != "at-only" {
		t.Errorf("AccessToken=%q, want at-only", tok.AccessToken)
	}
	if tok.RefreshToken != "" {
		t.Errorf("RefreshToken=%q, want empty", tok.RefreshToken)
	}
	if !strings.Contains(warnBuf.String(), "refresh_token") {
		t.Errorf("expected a no-refresh-token warning on stderr, got %q", warnBuf.String())
	}
	// No file written when there's nothing to persist.
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Errorf("expected no token file when no refresh_token; stat err=%v", err)
	}
}

func TestLogin_TokenExchangeNon2xxFails(t *testing.T) {
	restore := stubBrowser()
	defer restore()

	f := &fakeIAM{
		authCode:    "c",
		tokenBody:   `{"error":"invalid_grant"}`,
		tokenStatus: http.StatusBadRequest,
	}
	srv := startFakeIAM(t, f)
	defer srv.Close()

	storePath := filepath.Join(t.TempDir(), "token.json")
	cfg := newLoginTestCfg(srv.URL, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Login(ctx, cfg, storePath)
	if err == nil {
		t.Fatal("expected error for 400 token response, got nil")
	}
	if !strings.Contains(err.Error(), "token exchange") {
		t.Errorf("err=%v, want a 'token exchange' wrap", err)
	}
}
