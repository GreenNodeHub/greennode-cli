package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ini.v1"

	"github.com/vngcloud/greennode-cli/internal/auth"
	"github.com/vngcloud/greennode-cli/internal/config"
)

func TestNewTokenProvider_UserMode_BuildsLoginTokenProvider(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Profile:      "default",
		AuthMode:     "user",
		RefreshToken: "rt-123",
		IamEnv:       "dev",
	}
	tp, err := NewTokenProvider(cfg)
	if err != nil {
		t.Fatalf("NewTokenProvider: %v", err)
	}
	if _, ok := tp.(*auth.LoginTokenProvider); !ok {
		t.Errorf("tp=%T, want *auth.LoginTokenProvider", tp)
	}
}

func TestNewTokenProvider_MachineMode_BuildsTokenManager(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Profile: "default",
		// AuthMode unset → machine (today's behavior).
		ClientID:     "cid",
		ClientSecret: "cs",
	}
	tp, err := NewTokenProvider(cfg)
	if err != nil {
		t.Fatalf("NewTokenProvider: %v", err)
	}
	if _, ok := tp.(*auth.TokenManager); !ok {
		t.Errorf("tp=%T, want *auth.TokenManager", tp)
	}
}

func TestNewTokenProvider_UserModeMissingRefreshToken_Errors(t *testing.T) {
	t.Parallel()
	// Refresh token missing → the only hard-error condition in the user branch
	// (the client_id is resolved from iam_env, so a missing client_id is no
	// longer an error). The error must point to `grn login`.
	cfg := &config.Config{Profile: "default", AuthMode: "user", RefreshToken: "", IamEnv: "dev"}
	_, err := NewTokenProvider(cfg)
	if err == nil {
		t.Fatal("NewTokenProvider succeeded, want error")
	}
	if !strings.Contains(err.Error(), "grn login") {
		t.Errorf("err=%q, want guidance mentioning `grn login`", err)
	}
}

func TestNewTokenProvider_UserModeBadIamEnv_Errors(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Profile:      "default",
		AuthMode:     "user",
		RefreshToken: "rt",
		IamEnv:       "staging",
	}
	_, err := NewTokenProvider(cfg)
	if err == nil {
		t.Fatal("NewTokenProvider succeeded, want error for unknown iam_env")
	}
}

func TestNewTokenProvider_MachineModeMissingCreds_Errors(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Profile: "default"} // no AuthMode, no machine creds
	_, err := NewTokenProvider(cfg)
	if err == nil {
		t.Fatal("NewTokenProvider succeeded, want error")
	}
	if !strings.Contains(err.Error(), "grn configure") {
		t.Errorf("err=%q, want guidance mentioning `grn configure`", err)
	}
}

// TestNewTokenProvider_UserMode_RotationPersistsToResolvedProfile is the
// regression for the silent "failed to create section ”: empty section name"
// warning: the rotation persist callback must close over cfg.Profile (the
// RESOLVED profile from LoadConfig) — not the raw --profile flag, which is ""
// when unset. Pre-fix, rotation persisted nowhere and the next invocation
// refreshed against a STALE refresh token. This drives a refresh-grant
// rotation through NewTokenProvider against a httptest token server (via the
// tokenURLForEnv seam) and asserts the rotated token lands in [default].
func TestNewTokenProvider_UserMode_RotationPersistsToResolvedProfile(t *testing.T) {
	// NOT parallel: mutates process HOME and the package-level tokenURLForEnv.
	body := `{"access_token":"at-1","token_type":"Bearer","refresh_token":"rt-ROTATED","expires_in":3600}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	prev := tokenURLForEnv
	tokenURLForEnv = func(string) (string, error) { return srv.URL, nil }
	t.Cleanup(func() { tokenURLForEnv = prev })

	// Isolated HOME so the rotation write lands in a temp credentials INI.
	t.Setenv("HOME", t.TempDir())

	cfg := &config.Config{
		Profile:      "default", // resolved profile — the section WriteLoginToken must target
		AuthMode:     "user",
		RefreshToken: "rt-orig",
		IamEnv:       "dev",
	}
	tp, err := NewTokenProvider(cfg)
	if err != nil {
		t.Fatalf("NewTokenProvider: %v", err)
	}
	if _, err := tp.GetToken(); err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	creds, err := ini.Load(filepath.Join(config.DefaultConfigDir(), "credentials"))
	if err != nil {
		t.Fatalf("load credentials: %v (rotation persist did not write?)", err)
	}
	sec, err := creds.GetSection("default")
	if err != nil {
		t.Fatalf("no [default] section (empty-profile regression); sections=%v", creds.SectionStrings())
	}
	if got := sec.Key("refresh_token").String(); got != "rt-ROTATED" {
		t.Errorf("persisted refresh_token=%q, want rt-ROTATED", got)
	}
}
