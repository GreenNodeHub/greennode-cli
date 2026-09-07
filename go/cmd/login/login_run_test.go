package login

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/config"
	loginpkg "github.com/greennodehub/greennode-cli/internal/login"
)

// fakeFlow satisfies loginFlow for runLogin tests, avoiding a real loopback bind.
type fakeFlow struct {
	redirectURI   string
	authURL       string
	setAuthorize  string
	tok           loginpkg.Token
	completeSleep time.Duration // optional, to let the spinner draw a frame
	closed        bool
}

func (f *fakeFlow) AuthURL() string          { return f.authURL }
func (f *fakeFlow) RedirectURI() string      { return f.redirectURI }
func (f *fakeFlow) SetAuthorizeURL(u string) { f.setAuthorize = u }
func (f *fakeFlow) Complete(ctx context.Context) (loginpkg.Token, error) {
	if f.completeSleep > 0 {
		select {
		case <-time.After(f.completeSleep):
		case <-ctx.Done():
		}
	}
	return f.tok, nil
}
func (f *fakeFlow) Close() error { f.closed = true; return nil }

// withStdinPipe replaces os.Stdin with a pipe whose write end is pre-closed
// (EOF / non-interactive). Restored on cleanup.
func withStdinPipe(t *testing.T) {
	t.Helper()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	pw.Close()
	old := os.Stdin
	os.Stdin = pr
	t.Cleanup(func() { os.Stdin = old; pr.Close() })
}

func newRunLoginCmd() *cobra.Command {
	c := &cobra.Command{}
	c.SetContext(context.Background()) // mirror Execute(), which sets a non-nil ctx; runLogin calls cmd.Context()
	c.Flags().Bool("debug", false, "")
	return c
}

func TestRunLogin_Pkce_NonInteractive_PersistsToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	t.Setenv("GRN_IAM_ENV", "prod")
	flagMethod, flagSso = "", ""
	withStdinPipe(t)

	origFlow, origOpen := newFlow, openBrowserFn
	fake := &fakeFlow{
		redirectURI:   "http://127.0.0.1:59999/callback",
		authURL:       "https://example.test/auth",
		tok:           loginpkg.Token{AccessToken: "at", RefreshToken: "rt-xyz", ExpiresAt: time.Now().Add(time.Hour)},
		completeSleep: 5 * time.Millisecond, // let the spinner draw at least one frame
	}
	newFlow = func(ctx context.Context, cfg loginpkg.Config) (loginFlow, error) { return fake, nil }
	openBrowserFn = func(string) error { return nil }
	t.Cleanup(func() { newFlow, openBrowserFn = origFlow, origOpen })

	if err := runLogin(newRunLoginCmd(), nil); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	if !fake.closed {
		t.Error("flow.Close() not called (port leak)")
	}
	cfg, err := config.LoadConfig("default")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.RefreshToken != "rt-xyz" {
		t.Errorf("RefreshToken=%q, want rt-xyz", cfg.RefreshToken)
	}
	if cfg.AuthMode != "user" {
		t.Errorf("AuthMode=%q, want user", cfg.AuthMode)
	}
	if cfg.IamEnv != "prod" {
		t.Errorf("IamEnv=%q, want prod", cfg.IamEnv)
	}
}

func TestRunLogin_Sso_NonInteractive_UsesStrippedURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	t.Setenv("GRN_IAM_ENV", "prod")
	flagMethod, flagSso = "", "https://sso.example/ap/auth?clientId=sso-client"
	withStdinPipe(t)

	origFlow, origOpen := newFlow, openBrowserFn
	fake := &fakeFlow{
		redirectURI: "http://127.0.0.1:59998/callback",
		authURL:     "https://will-be-overridden/auth",
		tok:         loginpkg.Token{AccessToken: "at", RefreshToken: "rt-sso", ExpiresAt: time.Now().Add(time.Hour)},
	}
	newFlow = func(ctx context.Context, cfg loginpkg.Config) (loginFlow, error) { return fake, nil }
	openBrowserFn = func(string) error { return nil }
	t.Cleanup(func() { newFlow, openBrowserFn = origFlow, origOpen })

	if err := runLogin(newRunLoginCmd(), nil); err != nil {
		t.Fatalf("runLogin: %v", err)
	}
	// SetAuthorizeURL must have been called with the loopback redirect forced in.
	if !strings.Contains(fake.setAuthorize, "127.0.0.1:59998") && !strings.Contains(strings.ReplaceAll(fake.setAuthorize, "%3A", ":"), "127.0.0.1:59998") {
		t.Errorf("SetAuthorizeURL=%q does not contain the loopback redirect", fake.setAuthorize)
	}
	cfg, _ := config.LoadConfig("default")
	if cfg.RefreshToken != "rt-sso" {
		t.Errorf("RefreshToken=%q, want rt-sso", cfg.RefreshToken)
	}
	if cfg.AuthMode != "user" {
		t.Errorf("AuthMode=%q, want user", cfg.AuthMode)
	}
}

func TestRunLogin_CancelAtConfirm_WritesNoCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GRN_PROFILE", "")
	t.Setenv("GRN_IAM_ENV", "prod")
	flagMethod, flagSso = "", ""
	withStdinPipe(t) // EOF at the confirm prompt → errLoginCanceled

	origInteractive := isStdinInteractive
	isStdinInteractive = func() bool { return true } // force the interactive path
	t.Cleanup(func() { isStdinInteractive = origInteractive })

	origFlow, origOpen := newFlow, openBrowserFn
	fake := &fakeFlow{redirectURI: "http://127.0.0.1:59997/callback", authURL: "https://example.test/auth"}
	newFlow = func(ctx context.Context, cfg loginpkg.Config) (loginFlow, error) { return fake, nil }
	openBrowserFn = func(string) error { return nil }
	t.Cleanup(func() { newFlow, openBrowserFn = origFlow, origOpen })

	err := runLogin(newRunLoginCmd(), nil)
	if err == nil {
		t.Fatal("expected cancel error, got nil")
	}
	if !fake.closed {
		t.Error("flow.Close() not called on cancel (port leak)")
	}
	// No credentials must be written on cancel (Complete was never reached).
	if _, statErr := os.Stat(credentialsFilePath("default")); statErr == nil {
		t.Error("credentials file was written on cancel (should not be)")
	}
}
