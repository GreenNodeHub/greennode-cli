package login

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"golang.org/x/term"
)

// isInteractive reports whether the given file is a terminal (TTY). Used to
// decide whether to show the login method menu and the ENTER-to-open-browser
// prompt; non-interactive stdin (piped/CI) skips both and runs pkce silently.
// golang.org/x/term is already a direct dependency (internal/agentbase uses it).
func isInteractive(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// isStdinInteractive is the cmd->TTY seam: production wraps isInteractive(os.Stdin);
// tests override it to drive the interactive login path without a real TTY.
// Mirrors the newFlow/openBrowserFn seam pattern in flow_seam.go.
var isStdinInteractive = func() bool { return isInteractive(os.Stdin) }

// flagMethod/flagSso are bound to the `grn login` flags --method/--sso in
// login.go init(). Defaults are empty so an unset --method falls through to the
// interactive menu (when TTY) or silent pkce (when not), and --sso is opt-in.
var (
	flagMethod string
	flagSso    string
)

// resolveMethod resolves the login method + SSO authorize URL (if any) from the
// --method/--sso flags plus an interactive menu/prompt. Precedence:
//  1. --sso <https-url>      → "sso" + that url (regardless of --method)
//  2. --method pkce          → "pkce"
//  3. --method sso:
//     interactive         → promptSSOLink for the url
//     non-interactive     → error (needs --sso or a TTY)
//  4. empty --method:
//     interactive         → promptMenu (default pkce; choice 2 prompts for link)
//     non-interactive     → "pkce" silently
//
// Returns (method, ssoURL, err). For method=="pkce", ssoURL=="". Pure w.r.t.
// the injected reader + interactive flag so it table-tests cleanly.
func resolveMethod(method, ssoFlag string, reader *bufio.Reader, interactive bool) (string, string, error) {
	if ssoFlag != "" {
		if !strings.HasPrefix(ssoFlag, "https://") {
			return "", "", fmt.Errorf("--sso requires an https URL")
		}
		if _, err := url.Parse(ssoFlag); err != nil {
			return "", "", fmt.Errorf("--sso URL is not valid: %w", err)
		}
		return "sso", ssoFlag, nil
	}
	switch method {
	case "":
		if !interactive {
			return "pkce", "", nil
		}
		idx, err := promptMenu(reader, "Choose a login method:", []string{"pkce (browser + local callback)", "sso (paste a login link)"})
		if err != nil {
			return "", "", err
		}
		if idx == 0 {
			return "pkce", "", nil
		}
		link, err := promptSSOLink(reader)
		if err != nil {
			return "", "", err
		}
		return "sso", link, nil
	case "pkce":
		return "pkce", "", nil
	case "sso":
		if !interactive {
			return "", "", fmt.Errorf("--method sso requires --sso <url> or a TTY to prompt")
		}
		link, err := promptSSOLink(reader)
		if err != nil {
			return "", "", err
		}
		return "sso", link, nil
	default:
		return "", "", fmt.Errorf("invalid --method %q (valid: pkce, sso)", method)
	}
}
