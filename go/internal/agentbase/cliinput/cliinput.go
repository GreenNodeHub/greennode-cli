// Package cliinput provides dual-mode input helpers for CLI commands.
// In non-interactive mode (the default), missing required values produce errors.
// In interactive mode (--interactive / -i), missing values trigger terminal prompts.
package cliinput

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	interactive bool
	sc          *bufio.Scanner
)

func init() {
	sc = bufio.NewScanner(os.Stdin)
}

// SetInteractive enables or disables interactive prompt mode.
func SetInteractive(v bool) { interactive = v }

// IsInteractive reports whether interactive mode is active.
func IsInteractive() bool { return interactive }

// SetReader replaces the underlying scanner reader. Use in tests to inject fake stdin.
func SetReader(r io.Reader) { sc = bufio.NewScanner(r) }

// RequireOrPromptString returns value if non-empty. When value is empty and the
// mode is non-interactive, it returns an error naming flagName. When interactive,
// it prompts with label and requires a non-empty answer.
func RequireOrPromptString(value, flagName, label string) (string, error) {
	return RequireOrPromptStringWithPlaceholder(value, flagName, label, "")
}

// RequireOrPromptStringWithPlaceholder is like RequireOrPromptString but when
// placeholder is non-empty, the interactive prompt is "Label (placeholder): ".
func RequireOrPromptStringWithPlaceholder(value, flagName, label, placeholder string) (string, error) {
	if value != "" {
		return value, nil
	}
	if !interactive {
		return "", fmt.Errorf("required flag %q not set", flagName)
	}
	s, err := promptLineWithPlaceholder(label, placeholder)
	if err != nil {
		return "", err
	}
	if s == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return s, nil
}

// RequireOrPromptSecret returns value if non-empty. When value is empty and the
// mode is non-interactive, it returns an error. When interactive, it reads from
// the terminal without echoing (falling back to plain input when not a TTY).
func RequireOrPromptSecret(value, flagName, label string) (string, error) {
	if value != "" {
		return value, nil
	}
	if !interactive {
		return "", fmt.Errorf("required flag %q not set", flagName)
	}
	fmt.Fprintf(os.Stdout, "%s: ", label)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stdout)
	if err != nil {
		// Fallback to plain-text read (e.g. when stdin is not a TTY).
		if !sc.Scan() {
			return "", fmt.Errorf("%s is required", label)
		}
		s := strings.TrimSpace(sc.Text())
		if s == "" {
			return "", fmt.Errorf("%s is required", label)
		}
		return s, nil
	}
	s := strings.TrimSpace(string(pw))
	if s == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return s, nil
}

// RequireOrPromptStringSlice returns values if non-empty. When empty and
// non-interactive, it returns an error. When interactive, it prompts for a
// comma-separated list.
func RequireOrPromptStringSlice(values []string, flagName, label string) ([]string, error) {
	if len(values) > 0 {
		return values, nil
	}
	if !interactive {
		return nil, fmt.Errorf("required flag %q not set", flagName)
	}
	fmt.Fprintf(os.Stdout, "%s (comma-separated): ", label)
	if !sc.Scan() {
		return nil, fmt.Errorf("%s is required", label)
	}
	raw := strings.TrimSpace(sc.Text())
	if raw == "" {
		return nil, fmt.Errorf("%s is required", label)
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

// PromptString reads a labelled line from stdin. Callers should only call this
// in interactive mode.
func PromptString(label string) (string, error) {
	return promptLine(label)
}

// PromptIntDefault reads an integer with a displayed default. Returns def if the
// user presses Enter without input or enters an invalid value.
func PromptIntDefault(label string, def int) int {
	fmt.Fprintf(os.Stdout, "%s [%d]: ", label, def)
	if !sc.Scan() {
		return def
	}
	s := strings.TrimSpace(sc.Text())
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}

// PromptChoice presents a numbered list of items and returns the selected 0-based
// index. Defaults to the first item if the user presses Enter or enters an invalid
// selection.
func PromptChoice(label string, items []string) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("no items to choose from")
	}
	fmt.Fprintln(os.Stdout, label+":")
	for i, item := range items {
		fmt.Fprintf(os.Stdout, "  [%d] %s\n", i+1, item)
	}
	fmt.Fprint(os.Stdout, "Select [1]: ")
	if !sc.Scan() {
		return 0, nil
	}
	s := strings.TrimSpace(sc.Text())
	if s == "" {
		return 0, nil
	}
	var idx int
	if _, err := fmt.Sscanf(s, "%d", &idx); err != nil || idx < 1 || idx > len(items) {
		return 0, nil
	}
	return idx - 1, nil
}

// Confirm asks a yes/no question. Returns true only when the user answers "y" or "Y".
func Confirm(prompt string) bool {
	fmt.Fprintf(os.Stdout, "%s [y/N] ", prompt)
	if !sc.Scan() {
		return false
	}
	return strings.ToLower(strings.TrimSpace(sc.Text())) == "y"
}

func promptLine(label string) (string, error) {
	return promptLineWithPlaceholder(label, "")
}

func promptLineWithPlaceholder(label, placeholder string) (string, error) {
	if label != "" {
		if p := strings.TrimSpace(placeholder); p != "" {
			fmt.Fprintf(os.Stdout, "%s (%s): ", label, p)
		} else {
			fmt.Fprintf(os.Stdout, "%s: ", label)
		}
	}
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(sc.Text()), nil
}
