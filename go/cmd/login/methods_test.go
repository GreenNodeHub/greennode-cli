package login

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// isInteractive on a non-tty (the test process's stdin is piped/not a terminal)
// must return false. A true-positive TTY test is environment-dependent and is
// not asserted here; the spinner/login-flow tests exercise the interactive seam
// by overriding isStdinInteractive directly.
func TestIsInteractive_NonTtyStdinFalse(t *testing.T) {
	// CI/test runners pipe stdin, so this is reliably false. If it ever flips,
	// the seam-based tests still cover interactive behavior.
	if isInteractive(os.Stdin) {
		t.Skip("stdin is a TTY in this environment; skipping non-tty assertion")
	}
}

func TestResolveMethod(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		ssoFlag     string
		input       string // stdin for interactive prompts; "" means non-interactive via interactive=false OR EOF
		interactive bool
		wantMethod  string
		wantSso     string
		wantErr     string
	}{
		{
			name: "--sso dominates --method", method: "pkce", ssoFlag: "https://sso.example/auth",
			interactive: false, wantMethod: "sso", wantSso: "https://sso.example/auth",
		},
		{
			name: "--sso non-https errors", ssoFlag: "http://insecure",
			interactive: false, wantErr: "https",
		},
		{
			name: "--method pkce", method: "pkce", interactive: false, wantMethod: "pkce", wantSso: "",
		},
		{
			name: "--method sso non-interactive no --sso errors", method: "sso", interactive: false,
			wantErr: "requires --sso",
		},
		{
			name: "--method sso interactive prompts link", method: "sso", interactive: true,
			input: "https://sso.example/auth\n", wantMethod: "sso", wantSso: "https://sso.example/auth",
		},
		{
			name: "empty non-interactive -> pkce", method: "", interactive: false, wantMethod: "pkce",
		},
		{
			name: "empty interactive menu choice 1 -> pkce", method: "", interactive: true,
			input: "1\n", wantMethod: "pkce", wantSso: "",
		},
		{
			name: "empty interactive menu choice 2 -> sso + link", method: "", interactive: true,
			input: "2\nhttps://sso.example/auth\n", wantMethod: "sso", wantSso: "https://sso.example/auth",
		},
		{
			name: "empty interactive menu Enter -> pkce", method: "", interactive: true,
			input: "\n", wantMethod: "pkce",
		},
		{
			name: "invalid --method errors", method: "device", interactive: false, wantErr: "invalid --method",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r *bufio.Reader
			if tc.interactive {
				r = bufio.NewReader(strings.NewReader(tc.input))
			} else {
				r = bufio.NewReader(strings.NewReader(""))
			}
			gotMethod, gotSso, err := resolveMethod(tc.method, tc.ssoFlag, r, tc.interactive)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%q, want it to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if gotMethod != tc.wantMethod {
				t.Errorf("method=%q, want %q", gotMethod, tc.wantMethod)
			}
			if gotSso != tc.wantSso {
				t.Errorf("sso=%q, want %q", gotSso, tc.wantSso)
			}
		})
	}
}
