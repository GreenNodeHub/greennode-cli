package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jmespath/go-jmespath"
)

// knownListKeys are the wrapper keys list responses use for their item slice.
// Checking these by name (instead of "first array field found") prevents a
// detail object's nested array field (e.g. a cluster's listSubnetIds) from being
// mistaken for the rows — which previously produced empty table/text output.
var knownListKeys = []string{"items", "listData", "data"}

// Format formats and outputs the response data (no coloring).
func Format(data interface{}, outputFormat, query string, w io.Writer) error {
	return FormatColor(data, outputFormat, query, w, false)
}

// FormatColor formats and outputs the response data, coloring status values in
// text/table output when color is true. JSON output is never colored so it
// stays valid.
func FormatColor(data interface{}, outputFormat, query string, w io.Writer, color bool) error {
	if w == nil {
		w = os.Stdout
	}

	// Apply JMESPath query if specified
	if query != "" {
		result, err := jmespath.Search(query, data)
		if err != nil {
			return fmt.Errorf("JMESPath query error: %w", err)
		}
		data = result
	}

	if data == nil {
		return nil
	}

	switch outputFormat {
	case "json":
		formatJSON(data, w)
	case "text":
		formatText(data, w, color)
	case "table":
		formatTable(data, w, color)
	default:
		formatJSON(data, w)
	}
	return nil
}

func formatJSON(data interface{}, w io.Writer) {
	if isEmptyMap(data) {
		return
	}
	out, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintf(w, "%v\n", data)
		return
	}
	fmt.Fprintf(w, "%s\n", string(out))
}

func formatText(data interface{}, w io.Writer, color bool) {
	if data == nil || isEmptyMap(data) {
		return
	}

	if rows, ok := listRows(data); ok {
		for _, item := range rows {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Fprintln(w, strings.Join(colorValues(mapValues(m), color), "\t"))
			} else {
				fmt.Fprintln(w, fmt.Sprint(item))
			}
		}
		return
	}

	if m, ok := data.(map[string]interface{}); ok {
		fmt.Fprintln(w, strings.Join(colorValues(mapValues(m), color), "\t"))
		return
	}
	fmt.Fprintln(w, fmt.Sprint(data))
}

// colorValues colors each value that is a recognized status. Values are not
// padded here, so the raw value doubles as both the padded and raw argument.
func colorValues(vals []string, color bool) []string {
	if !color {
		return vals
	}
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = colorCell(v, v, color)
	}
	return out
}

func formatTable(data interface{}, w io.Writer, color bool) {
	if data == nil || isEmptyMap(data) {
		return
	}

	// List response (top-level array, or object wrapping items/listData/data):
	// render as a multi-column table.
	if rows, ok := listRows(data); ok {
		formatRowsTable(rows, w, color)
		return
	}

	// Detail response (a single object): render as a two-column key/value table
	// so its scalar fields are visible instead of being hijacked by a nested array.
	if m, ok := data.(map[string]interface{}); ok {
		formatKeyValueTable(m, w, color)
		return
	}

	fmt.Fprintln(w, fmt.Sprint(data))
}

// formatRowsTable renders a slice of object rows as a multi-column table with a
// header row. Columns are the sorted keys of the first row. Non-map rows are
// printed one per line.
func formatRowsTable(rows []interface{}, w io.Writer, color bool) {
	if len(rows) == 0 {
		return
	}
	firstMap, ok := rows[0].(map[string]interface{})
	if !ok {
		for _, row := range rows {
			fmt.Fprintln(w, fmt.Sprint(row))
		}
		return
	}

	headers := mapKeys(firstMap)
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}

	strRows := make([][]string, len(rows))
	for i, row := range rows {
		m, _ := row.(map[string]interface{})
		strRows[i] = make([]string, len(headers))
		for j, h := range headers {
			val := fmt.Sprint(m[h])
			strRows[i][j] = val
			if len(val) > colWidths[j] {
				colWidths[j] = len(val)
			}
		}
	}

	printRow(w, headers, colWidths)
	printSeparator(w, colWidths)
	for _, row := range strRows {
		printColoredRow(w, row, colWidths, color)
	}
}

// formatKeyValueTable renders a single object as a two-column FIELD | VALUE
// table, with fields sorted for deterministic output.
func formatKeyValueTable(m map[string]interface{}, w io.Writer, color bool) {
	if len(m) == 0 {
		return
	}
	keys := mapKeys(m)
	fieldWidth := len("FIELD")
	for _, k := range keys {
		if len(k) > fieldWidth {
			fieldWidth = len(k)
		}
	}
	valWidth := len("VALUE")
	vals := make([]string, len(keys))
	for i, k := range keys {
		vals[i] = fmt.Sprint(m[k])
		if len(vals[i]) > valWidth {
			valWidth = len(vals[i])
		}
	}

	widths := []int{fieldWidth, valWidth}
	printRow(w, []string{"FIELD", "VALUE"}, widths)
	printSeparator(w, widths)
	for i, k := range keys {
		printColoredRow(w, []string{k, vals[i]}, widths, color)
	}
}

func printRow(w io.Writer, cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = padRight(c, widths[i])
	}
	fmt.Fprintln(w, strings.Join(parts, " | "))
}

// printColoredRow is printRow that colors any cell whose value is a recognized
// status. Cells are padded on their raw text first, so alignment is unaffected.
func printColoredRow(w io.Writer, cells []string, widths []int, color bool) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = colorCell(padRight(c, widths[i]), c, color)
	}
	fmt.Fprintln(w, strings.Join(parts, " | "))
}

func printSeparator(w io.Writer, widths []int) {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width)
	}
	fmt.Fprintln(w, strings.Join(parts, "-+-"))
}

// listRows returns the item slice if data is a list response (a top-level array,
// or an object with a known list key); ok=false for a detail (single) object.
func listRows(data interface{}) ([]interface{}, bool) {
	switch v := data.(type) {
	case []interface{}:
		return v, true
	case map[string]interface{}:
		for _, k := range knownListKeys {
			if items, ok := v[k].([]interface{}); ok {
				return items, true
			}
		}
	}
	return nil, false
}

// mapKeys returns the map's keys sorted, for deterministic column/row order.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// mapValues returns the map's values ordered by sorted key, matching mapKeys.
func mapValues(m map[string]interface{}) []string {
	keys := mapKeys(m)
	vals := make([]string, len(keys))
	for i, k := range keys {
		vals[i] = fmt.Sprint(m[k])
	}
	return vals
}

func isEmptyMap(data interface{}) bool {
	m, ok := data.(map[string]interface{})
	return ok && len(m) == 0
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// Truncate shortens s to at most n runes, appending "..." when truncated.
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

// ShortDate reformats an RFC3339 (or similar) timestamp string to "2006-01-02".
// If the string cannot be parsed it returns the first 10 characters, or the
// original string when it is shorter than 10.
func ShortDate(s string) string {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC().Format("2006-01-02")
		}
	}
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
func FormatTableWithColumns(data interface{}, columns []string, query string, w io.Writer) error {
	return FormatTableWithColumnsColor(data, columns, query, w, false)
}

// FormatTableWithColumnsColor is FormatTableWithColumns that colors status
// values when color is true.
func FormatTableWithColumnsColor(data interface{}, columns []string, query string, w io.Writer, color bool) error {
	if w == nil {
		w = os.Stdout
	}
	if query != "" {
		result, err := jmespath.Search(query, data)
		if err != nil {
			return fmt.Errorf("JMESPath query error: %w", err)
		}
		data = result
	}
	if data == nil {
		return nil
	}
	formatTableColumns(data, columns, w, color)
	return nil
}

func formatTableColumns(data interface{}, columns []string, w io.Writer, color bool) {
	if data == nil || isEmptyMap(data) {
		return
	}
	rows := extractRows(data)
	if len(rows) == 0 {
		return
	}
	if _, ok := rows[0].(map[string]interface{}); !ok {
		return
	}

	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = formatHeader(c)
	}

	colWidths := make([]int, len(columns))
	for i, h := range headers {
		colWidths[i] = len(h)
	}

	strRows := make([][]string, len(rows))
	for i, row := range rows {
		m, _ := row.(map[string]interface{})
		strRows[i] = make([]string, len(columns))
		for j, col := range columns {
			val := fmt.Sprint(m[col])
			if val == "<nil>" {
				val = ""
			}
			strRows[i][j] = val
			if len(val) > colWidths[j] {
				colWidths[j] = len(val)
			}
		}
	}

	headerParts := make([]string, len(headers))
	sepParts := make([]string, len(headers))
	for i, h := range headers {
		headerParts[i] = padRight(h, colWidths[i])
		sepParts[i] = strings.Repeat("-", colWidths[i])
	}
	fmt.Fprintln(w, strings.Join(headerParts, " | "))
	fmt.Fprintln(w, strings.Join(sepParts, "-+-"))

	for _, row := range strRows {
		printColoredRow(w, row, colWidths, color)
	}
}

func extractRows(data interface{}) []interface{} {
	switch v := data.(type) {
	case []interface{}:
		return v
	case map[string]interface{}:
		// List envelope: {"listData": [...]} or any key holding a slice
		for _, value := range v {
			if items, ok := value.([]interface{}); ok {
				return items
			}
		}
		// Single-object envelope: {"data": {...}} — unwrap the inner object
		if inner, ok := v["data"].(map[string]interface{}); ok {
			return []interface{}{inner}
		}
		return []interface{}{v}
	default:
		return []interface{}{v}
	}
}
func formatHeader(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i > 0 && c >= 'A' && c <= 'Z' {
			b.WriteByte(' ')
		}
		if c >= 'a' && c <= 'z' {
			b.WriteByte(c - 32)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}
