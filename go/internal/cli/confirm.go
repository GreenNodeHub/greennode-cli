package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

var nonInteractive bool
var confirmationError error

// SetNonInteractive disables stdin prompts.
func SetNonInteractive(enabled bool) {
	nonInteractive = enabled
	confirmationError = nil
}

// IsNonInteractive reports whether commands must fail fast instead of prompting.
func IsNonInteractive() bool {
	return nonInteractive
}

// ConfirmationError fails unforced non-interactive calls; interactive refusals are no-ops.
func ConfirmationError() error {
	return confirmationError
}

// DryRunNotice prints how to execute the previewed action.
func DryRunNotice(verb string) {
	fmt.Printf("\nRun without --dry-run to %s.\n", verb)
}

// PrintDryRun prints a stable, credential-redacted request preview.
func PrintDryRun(verb, target string, body map[string]any) {
	body = RedactJSON(body).(map[string]any)
	fmt.Println("=== DRY RUN ===")
	if target != "" {
		fmt.Printf("Would %s %s:\n", verb, target)
	}
	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %v\n", k, body[k])
	}
	DryRunNotice(verb)
}

// Confirm accepts force or explicit yes; non-interactive calls never read stdin.
func Confirm(force bool, prompt string) bool {
	if force {
		return true
	}
	if IsNonInteractive() {
		confirmationError = fmt.Errorf("confirmation required in non-interactive mode; rerun with --force")
		fmt.Fprintln(os.Stderr, confirmationError)
		return false
	}
	fmt.Printf("\n%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
