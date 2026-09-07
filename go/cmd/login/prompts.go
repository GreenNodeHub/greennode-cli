package login

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// errLoginCanceled is returned by confirmOpen when the user sends EOF/Ctrl-C at
// the ENTER-to-open-browser prompt. runLogin treats it as a non-zero abort: it
// prints "Login canceled." and returns, never calling Complete.
var errLoginCanceled = errors.New("login canceled")

// maxSSOPrompts bounds the re-prompt loop in promptSSOLink.
const maxSSOPrompts = 3

// promptMenu prints a numbered menu and returns the chosen 0-based index.
// Enter / EOF on a non-interactive reader returns index 0 (the default, pkce)
// without error so headless callers fall through cleanly. Out-of-range input
// re-prompts; a piped invalid line followed by EOF also falls back to 0.
func promptMenu(reader *bufio.Reader, label string, options []string) (int, error) {
	for {
		fmt.Printf("%s\n", label)
		for i, o := range options {
			fmt.Printf("  %d) %s\n", i+1, o)
		}
		fmt.Printf("Select [1]: ")
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			// Enter, or EOF with nothing typed → default.
			return 0, nil
		}
		if err != nil && line != "" {
			// Got a typed line then EOF (no newline); still try to parse it.
		}
		n, perr := strconv.Atoi(line)
		if perr != nil || n < 1 || n > len(options) {
			fmt.Fprintf(os.Stderr, "Invalid choice %q; enter 1-%d.\n", line, len(options))
			continue
		}
		return n - 1, nil
	}
}

// promptSSOLink prompts for an https SSO authorize URL, re-prompting on invalid
// input up to maxSSOPrompts times. Returns the trimmed URL or an error if the
// user exhausts retries or EOFs without input.
func promptSSOLink(reader *bufio.Reader) (string, error) {
	for i := 0; i < maxSSOPrompts; i++ {
		fmt.Print("Paste your SSO login link: ")
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			if err != nil {
				return "", fmt.Errorf("no SSO link provided")
			}
			continue
		}
		if !strings.HasPrefix(line, "https://") {
			fmt.Fprintf(os.Stderr, "The link must start with https:// (got %q). Try again.\n", line)
			continue
		}
		if _, perr := url.Parse(line); perr != nil {
			fmt.Fprintf(os.Stderr, "That link is not a valid URL. Try again.\n")
			continue
		}
		return line, nil
	}
	return "", fmt.Errorf("no valid SSO link after %d attempts", maxSSOPrompts)
}

// confirmOpen prompts the user to press ENTER to open the browser, or 's'/any
// other input to skip (the link is already printed, so the user can open it
// manually on the same machine). EOF/Ctrl-C returns (false, errLoginCanceled) so
// runLogin aborts without calling Complete. The cmd only calls this when
// isStdinInteractive() is true; non-interactive callers skip it entirely.
func confirmOpen(reader *bufio.Reader) (bool, error) {
	fmt.Print("Press ENTER to open the browser, or 's' to skip (open the link yourself): ")
	line, err := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		// EOF / Ctrl-C at the prompt → cancel.
		return false, errLoginCanceled
	}
	switch strings.ToLower(line) {
	case "", "y", "yes":
		return true, nil
	default:
		// 's', 'skip', or any unrecognized input → skip (link is shown).
		return false, nil
	}
}
