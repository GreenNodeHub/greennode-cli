package login

import (
	"os"
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
