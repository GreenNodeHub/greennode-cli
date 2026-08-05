// Package output provides table and JSON output formatting for CLI commands.
package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

// Format represents the output format requested by the user.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatID    Format = "id"
)

var currentFormat Format = FormatTable

// SetFormat sets the active output format. Called once from rootCmd.PersistentPreRun.
func SetFormat(f Format) { currentFormat = f }

// GetFormat returns the active output format.
func GetFormat() Format { return currentFormat }

// ParseFormat parses an output format string, defaulting to table.
func ParseFormat(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "id":
		return FormatID
	case "table":
		return FormatTable
	default:
		return FormatTable
	}
}

// PrintResource renders v as JSON, prints the ID via extractID, or calls humanFn for any other format.
// extractID must not be nil. humanFn must not be nil.
func PrintResource(v interface{}, extractID func() string, humanFn func() error) error {
	switch GetFormat() {
	case FormatJSON:
		return JSON(v)
	case FormatID:
		fmt.Fprintln(os.Stdout, extractID())
		return nil
	default:
		return humanFn()
	}
}

// PrintDeletedID renders a deletion result. JSON emits {"id":"<id>"},
// id emits the bare ID, table emits nothing (silence is golden).
func PrintDeletedID(id string) error {
	switch GetFormat() {
	case FormatJSON:
		return JSON(map[string]string{"id": id})
	case FormatID:
		fmt.Fprintln(os.Stdout, id)
		return nil
	default:
		return nil
	}
}

// JSON prints any value as indented JSON to stdout.
func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Table renders a table to stdout with the given headers and rows.
func Table(headers []string, rows [][]string) {
	t := tablewriter.NewWriter(os.Stdout)
	// Convert headers to interface{} slice for the new API.
	headerIface := make([]interface{}, len(headers))
	for i, h := range headers {
		headerIface[i] = h
	}
	t.Header(headerIface...)
	for _, row := range rows {
		rowIface := make([]interface{}, len(row))
		for i, cell := range row {
			rowIface[i] = cell
		}
		_ = t.Append(rowIface...)
	}
	_ = t.Render()
}

// Success prints a success message to stdout.
func Success(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}

// Successf prints a formatted success message to stdout.
func Successf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Error prints an error message to stderr and exits with code 1.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}

// Errorf prints a formatted error message to stderr and exits with code 1.
func Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// Warn prints a warning to stderr.
func Warn(msg string) {
	fmt.Fprintln(os.Stderr, "Warning:", msg)
}

// PrintID prints just the ID to stdout (for -o id flag).
func PrintID(id string) {
	fmt.Fprintln(os.Stdout, id)
}

// FormatTime returns a human-readable time string or "-" if nil.
func FormatTime(t interface{ String() string }) string {
	if t == nil {
		return "-"
	}
	return t.String()
}

// StrOrDash returns the string if non-empty, otherwise "-".
func StrOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
