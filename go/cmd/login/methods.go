package login

import (
	"os"

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
