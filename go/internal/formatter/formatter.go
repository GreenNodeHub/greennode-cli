package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jmespath/go-jmespath"
)

// knownListKeys defines deterministic list-wrapper precedence shared with completion.
var knownListKeys = []string{"items", "listData", "data", "volumeTypes"}

// KnownListKeys returns a copy of the ordered list-wrapper keys.
func KnownListKeys() []string {
	out := make([]string, len(knownListKeys))
	copy(out, knownListKeys)
	return out
}

// scalar renders numbers without exponent notation in text and table output.
func scalar(v any) string {
	switch n := v.(type) {
	case json.Number:
		// Exact source digits: normalize decodes with UseNumber precisely so a
		// value beyond float64's exact integer range survives to here.
		return n.String()
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// normalize gives every output format the same JSON keys, including struct tags.
// UseNumber preserves exact digits outside JMESPath queries.
func normalize(data any) (any, error) {
	if data == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("internal error: output value is not JSON-serializable: %w", err)
	}
	// Preserve integers beyond float64's exact range.
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var out any
	if err := decoder.Decode(&out); err != nil {
		return nil, fmt.Errorf("internal error: re-decoding output value: %w", err)
	}
	return out, nil
}

// queryable converts numbers to float64 for JMESPath; queries may round integers beyond 2^53.
func queryable(data any) any {
	switch v := data.(type) {
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
		return v.String()
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			out[key] = queryable(value)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, value := range v {
			out[i] = queryable(value)
		}
		return out
	}
	return data
}

// Format formats and outputs the response data (no coloring).
func Format(data any, outputFormat, query string, w io.Writer) error {
	return FormatColor(data, outputFormat, query, w, false)
}

// FormatColor colors text/table output, never JSON. Callers validate the output format.
func FormatColor(data any, outputFormat, query string, w io.Writer, color bool) error {
	if w == nil {
		w = os.Stdout
	}

	data, err := normalize(data)
	if err != nil {
		return err
	}

	// Apply JMESPath query if specified
	if query != "" {
		result, err := jmespath.Search(query, queryable(data))
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
		return formatJSON(data, w)
	case "text":
		return formatText(data, w, color)
	case "table":
		return formatTable(data, w, color)
	default:
		return formatJSON(data, w)
	}
}

func formatJSON(data any, w io.Writer) error {
	if isEmptyMap(data) {
		return nil
	}
	out, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("marshaling JSON output: %w", err)
	}
	if _, err := fmt.Fprintf(w, "%s\n", string(out)); err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}
	return nil
}

func formatText(data any, w io.Writer, color bool) error {
	if data == nil || isEmptyMap(data) {
		return nil
	}

	if rows, ok := listRows(data); ok {
		for _, item := range rows {
			var line string
			if m, ok := item.(map[string]any); ok {
				line = strings.Join(colorValues(mapValues(m), color), "\t")
			} else {
				line = scalar(item)
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return fmt.Errorf("writing text output: %w", err)
			}
		}
		return nil
	}

	if m, ok := data.(map[string]any); ok {
		if _, err := fmt.Fprintln(w, strings.Join(colorValues(mapValues(m), color), "\t")); err != nil {
			return fmt.Errorf("writing text output: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintln(w, scalar(data)); err != nil {
		return fmt.Errorf("writing text output: %w", err)
	}
	return nil
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

func formatTable(data any, w io.Writer, color bool) error {
	if data == nil || isEmptyMap(data) {
		return nil
	}

	// List response (top-level array, or object wrapping items/listData/data):
	// render as a multi-column table.
	if rows, ok := listRows(data); ok {
		return formatRowsTable(rows, w, color)
	}

	// Detail response (a single object): render as a two-column key/value table
	// so its scalar fields are visible instead of being hijacked by a nested array.
	if m, ok := data.(map[string]any); ok {
		return formatKeyValueTable(m, w, color)
	}

	if _, err := fmt.Fprintln(w, scalar(data)); err != nil {
		return fmt.Errorf("writing table output: %w", err)
	}
	return nil
}

// formatRowsTable renders a slice of object rows as a multi-column table with a
// header row. Columns are the sorted keys of the first row. Non-map rows are
// printed one per line.
func formatRowsTable(rows []any, w io.Writer, color bool) error {
	if len(rows) == 0 {
		return nil
	}
	firstMap, ok := rows[0].(map[string]any)
	if !ok {
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, scalar(row)); err != nil {
				return fmt.Errorf("writing table output: %w", err)
			}
		}
		return nil
	}

	headers := mapKeys(firstMap)
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}

	strRows := make([][]string, len(rows))
	for i, row := range rows {
		m, _ := row.(map[string]any)
		strRows[i] = make([]string, len(headers))
		for j, h := range headers {
			val := scalar(m[h])
			strRows[i][j] = val
			if n := utf8.RuneCountInString(val); n > colWidths[j] {
				colWidths[j] = n
			}
		}
	}

	if err := printRow(w, headers, colWidths); err != nil {
		return err
	}
	if err := printSeparator(w, colWidths); err != nil {
		return err
	}
	for _, row := range strRows {
		if err := printColoredRow(w, row, colWidths, color); err != nil {
			return err
		}
	}
	return nil
}

// formatKeyValueTable renders a single object as a two-column FIELD | VALUE
// table, with fields sorted for deterministic output.
func formatKeyValueTable(m map[string]any, w io.Writer, color bool) error {
	if len(m) == 0 {
		return nil
	}
	keys := mapKeys(m)
	fieldWidth := utf8.RuneCountInString("FIELD")
	for _, k := range keys {
		if n := utf8.RuneCountInString(k); n > fieldWidth {
			fieldWidth = n
		}
	}
	valWidth := utf8.RuneCountInString("VALUE")
	vals := make([]string, len(keys))
	for i, k := range keys {
		vals[i] = scalar(m[k])
		if n := utf8.RuneCountInString(vals[i]); n > valWidth {
			valWidth = n
		}
	}

	widths := []int{fieldWidth, valWidth}
	if err := printRow(w, []string{"FIELD", "VALUE"}, widths); err != nil {
		return err
	}
	if err := printSeparator(w, widths); err != nil {
		return err
	}
	for i, k := range keys {
		if err := printColoredRow(w, []string{k, vals[i]}, widths, color); err != nil {
			return err
		}
	}
	return nil
}

func printRow(w io.Writer, cells []string, widths []int) error {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = padRight(c, widths[i])
	}
	if _, err := fmt.Fprintln(w, strings.Join(parts, " | ")); err != nil {
		return fmt.Errorf("writing table row: %w", err)
	}
	return nil
}

// printColoredRow is printRow that colors any cell whose value is a recognized
// status. Cells are padded on their raw text first, so alignment is unaffected.
func printColoredRow(w io.Writer, cells []string, widths []int, color bool) error {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = colorCell(padRight(c, widths[i]), c, color)
	}
	if _, err := fmt.Fprintln(w, strings.Join(parts, " | ")); err != nil {
		return fmt.Errorf("writing table row: %w", err)
	}
	return nil
}

func printSeparator(w io.Writer, widths []int) error {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width)
	}
	if _, err := fmt.Fprintln(w, strings.Join(parts, "-+-")); err != nil {
		return fmt.Errorf("writing table separator: %w", err)
	}
	return nil
}

// listRows returns the item slice if data is a list response (a top-level array,
// or an object with a known list key); ok=false for a detail (single) object.
func listRows(data any) ([]any, bool) {
	switch v := data.(type) {
	case []any:
		return v, true
	case map[string]any:
		for _, k := range knownListKeys {
			if items, ok := v[k].([]any); ok {
				return items, true
			}
		}
	}
	return nil, false
}

// mapKeys returns the map's keys sorted, for deterministic column/row order.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// mapValues returns the map's values ordered by sorted key, matching mapKeys.
func mapValues(m map[string]any) []string {
	keys := mapKeys(m)
	vals := make([]string, len(keys))
	for i, k := range keys {
		vals[i] = scalar(m[k])
	}
	return vals
}

func isEmptyMap(data any) bool {
	m, ok := data.(map[string]any)
	return ok && len(m) == 0
}

// padRight pads by rune count to match table column widths.
func padRight(s string, n int) string {
	length := utf8.RuneCountInString(s)
	if length >= n {
		return s
	}
	return s + strings.Repeat(" ", n-length)
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
func FormatTableWithColumns(data any, columns []string, query string, w io.Writer) error {
	return FormatTableWithColumnsColor(data, columns, query, w, false)
}

// FormatTableWithColumnsColor is FormatTableWithColumns that colors status
// values when color is true.
func FormatTableWithColumnsColor(data any, columns []string, query string, w io.Writer, color bool) error {
	if w == nil {
		w = os.Stdout
	}
	data, err := normalize(data)
	if err != nil {
		return err
	}
	if query != "" {
		result, err := jmespath.Search(query, queryable(data))
		if err != nil {
			return fmt.Errorf("JMESPath query error: %w", err)
		}
		data = result
	}
	if data == nil {
		return nil
	}
	return formatTableColumns(data, columns, w, color)
}

func formatTableColumns(data any, columns []string, w io.Writer, color bool) error {
	if data == nil || isEmptyMap(data) {
		return nil
	}
	rows := extractRows(data)
	if len(rows) == 0 {
		return nil
	}
	if _, ok := rows[0].(map[string]any); !ok {
		return nil
	}

	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = formatHeader(c)
	}

	colWidths := make([]int, len(columns))
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}

	strRows := make([][]string, len(rows))
	for i, row := range rows {
		m, _ := row.(map[string]any)
		strRows[i] = make([]string, len(columns))
		for j, col := range columns {
			val := scalar(m[col])
			if val == "<nil>" {
				val = ""
			}
			strRows[i][j] = val
			if n := utf8.RuneCountInString(val); n > colWidths[j] {
				colWidths[j] = n
			}
		}
	}

	headerParts := make([]string, len(headers))
	sepParts := make([]string, len(headers))
	for i, h := range headers {
		headerParts[i] = padRight(h, colWidths[i])
		sepParts[i] = strings.Repeat("-", colWidths[i])
	}
	if _, err := fmt.Fprintln(w, strings.Join(headerParts, " | ")); err != nil {
		return fmt.Errorf("writing table header: %w", err)
	}
	if _, err := fmt.Fprintln(w, strings.Join(sepParts, "-+-")); err != nil {
		return fmt.Errorf("writing table separator: %w", err)
	}

	for _, row := range strRows {
		if err := printColoredRow(w, row, colWidths, color); err != nil {
			return err
		}
	}
	return nil
}

// extractRows uses known wrapper precedence, then the lexicographically first array field.
func extractRows(data any) []any {
	switch v := data.(type) {
	case []any:
		return v
	case map[string]any:
		if items, ok := listRows(v); ok {
			return items
		}
		if items, ok := smallestArrayKey(v); ok {
			return items
		}
		// Single-object envelope: {"data": {...}} — unwrap the inner object
		if inner, ok := v["data"].(map[string]any); ok {
			return []any{inner}
		}
		return []any{v}
	default:
		return []any{v}
	}
}

// smallestArrayKey returns the lexicographically first array field, if any.
func smallestArrayKey(v map[string]any) (items []any, ok bool) {
	var bestKey string
	for k, val := range v {
		arr, isArray := val.([]any)
		if !isArray {
			continue
		}
		if !ok || k < bestKey {
			bestKey, items, ok = k, arr, true
		}
	}
	return items, ok
}

// formatHeader inserts spaces before uppercase runes and uppercases the result.
func formatHeader(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}
