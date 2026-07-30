package configure

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vngcloud/greennode-cli/internal/config"
)

// captureOutput swaps os.Stdout for a pipe for the duration of fn and returns
// whatever was printed. Non-parallel: it mutates process-global os.Stdout. The
// configure commands print via fmt to os.Stdout (not cobra's Out), so the swap
// is what captures them.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	return <-done
}

// Unit: resolveCredEntry masks refresh_token (secret-at-rest), while
// resolveCredEntryPlain shows the non-secret login context as-is.
func TestResolveCredEntry_MasksRefreshToken(t *testing.T) {
	e := resolveCredEntry("refresh_token", "rt-SUPERSECRET-0001", "/home/u/.greennode/credentials")
	if strings.Contains(e.value, "rt-SUPERSECRET") {
		t.Errorf("refresh_token not masked: %q", e.value)
	}
	if !strings.HasSuffix(e.value, "0001") {
		t.Errorf("masked refresh_token should keep last 4, got %q", e.value)
	}
}

func TestResolveCredEntryPlain_ShowsLoginContextAsIs(t *testing.T) {
	for _, tc := range []struct{ name, val string }{
		{"auth_mode", "user"},
		{"login_client_id", "cid-login"},
		{"iam_env", "dev"},
	} {
		e := resolveCredEntryPlain(tc.name, tc.val, "/home/u/.greennode/credentials")
		if e.value != tc.val {
			t.Errorf("%s: got %q, want %q as-is (non-secret)", tc.name, e.value, tc.val)
		}
	}
	// Empty → <not set>, not a masked empty string.
	if e := resolveCredEntryPlain("auth_mode", "", "/h/.greennode/credentials"); e.value != "<not set>" {
		t.Errorf("empty auth_mode = %q, want <not set>", e.value)
	}
}

// End-to-end: `configure list` on a login profile masks refresh_token and shows
// the non-secret login context in plaintext.
func TestListMasksRefreshTokenShowsLoginContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GRN_PROFILE", "")
	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := config.NewConfigFileWriter().WriteLoginToken("default", "rt-SUPERSECRET-0001", exp, "user", "cid-login", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}

	root := newConfigureTestCmd()
	root.SetArgs([]string{"configure", "list", "--profile", "default"})
	out := captureOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("execute list: %v", err)
		}
	})

	if strings.Contains(out, "rt-SUPERSECRET") {
		t.Errorf("list leaked refresh_token plaintext: %q", out)
	}
	if !strings.Contains(out, "0001") {
		t.Errorf("list should show masked refresh_token ending in 0001: %q", out)
	}
	// Non-secret login context is shown as-is so a user can see the auth mode.
	for _, want := range []string{"user", "cid-login", "dev"} {
		if !strings.Contains(out, want) {
			t.Errorf("list missing non-secret login context %q: %q", want, out)
		}
	}
}

// `configure get refresh_token` masks; `get auth_mode`/`login_client_id`/
// `iam_env` return the plaintext non-secret value.
func TestGetRefreshTokenMasked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GRN_PROFILE", "")
	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := config.NewConfigFileWriter().WriteLoginToken("default", "rt-SUPERSECRET-0001", exp, "user", "cid-login", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}

	root := newConfigureTestCmd()
	root.SetArgs([]string{"configure", "get", "refresh_token", "--profile", "default"})
	out := strings.TrimSpace(captureOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("get refresh_token: %v", err)
		}
	}))
	if strings.Contains(out, "rt-SUPERSECRET") {
		t.Errorf("get refresh_token leaked plaintext: %q", out)
	}
	if !strings.HasSuffix(out, "0001") {
		t.Errorf("get refresh_token should be masked ending in 0001, got %q", out)
	}
}

func TestGetLoginContextPlain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GRN_PROFILE", "")
	exp := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	if err := config.NewConfigFileWriter().WriteLoginToken("default", "rt", exp, "user", "cid-login", "dev"); err != nil {
		t.Fatalf("WriteLoginToken: %v", err)
	}
	for _, tc := range []struct{ key, want string }{
		{"auth_mode", "user"},
		{"login_client_id", "cid-login"},
		{"iam_env", "dev"},
	} {
		root := newConfigureTestCmd()
		root.SetArgs([]string{"configure", "get", tc.key, "--profile", "default"})
		out := strings.TrimSpace(captureOutput(t, func() {
			if err := root.Execute(); err != nil {
				t.Fatalf("get %s: %v", tc.key, err)
			}
		}))
		if out != tc.want {
			t.Errorf("get %s = %q, want %q", tc.key, out, tc.want)
		}
	}
}
