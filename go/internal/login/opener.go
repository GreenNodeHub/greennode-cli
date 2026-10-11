package login

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// errNoOpener is returned by defaultOpenBrowser when no platform opener is known.
var errNoOpener = errors.New("no browser opener available for this platform")

// noisyStderr is the destination for fallback URL prints and warnings. It is a
// package var so tests can redirect it (e.g. to a *bytes.Buffer or io.Discard)
// to keep test output pristine; the default is os.Stderr.
var noisyStderr io.Writer = os.Stderr

// browserRunner returns a function that opens rawurl in the user's default
// browser, or nil if no platform opener is known.
func browserRunner() func(string) error {
	switch runtime.GOOS {
	case "darwin":
		return func(u string) error { return exec.Command("open", u).Start() }
	case "windows":
		return func(u string) error {
			// 'start' is a cmd builtin; run through cmd /c.
			quotedURL, err := quoteWindowsURL(u)
			if err != nil {
				return err
			}
			return exec.Command("cmd", "/c", "start", "", quotedURL).Start()
		}
	default: // linux / *bsd
		return func(u string) error {
			for _, name := range []string{"xdg-open", "x-www-browser", "www-browser"} {
				if _, err := exec.LookPath(name); err == nil {
					return exec.Command(name, u).Start()
				}
			}
			return errNoOpener
		}
	}
}

// quoteWindowsURL wraps a URL in double quotes for cmd.exe. Double quotes
// prevent cmd.exe from treating URL query separators as command operators.
// URLs should not contain embedded double quotes, so they are rejected.
func quoteWindowsURL(rawurl string) (string, error) {
	if strings.Contains(rawurl, `"`) {
		return "", errors.New("login: URL contains an invalid double quote")
	}
	return `"` + rawurl + `"`, nil
}
