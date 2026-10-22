package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFormatCoversPublicFormatsAndQueries(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		format string
		query  string
		want   []string
	}{
		{name: "json query", data: map[string]any{"data": map[string]any{"id": "server-1"}}, format: "json", query: "data", want: []string{`"id": "server-1"`}},
		{name: "unknown format falls back to json", data: map[string]any{"id": "server-1"}, format: "yaml", want: []string{`"id": "server-1"`}},
		{name: "text detail is deterministic", data: map[string]any{"status": "ACTIVE", "id": "server-1"}, format: "text", want: []string{"server-1\tACTIVE"}},
		{name: "text list maps and scalars", data: map[string]any{"items": []any{map[string]any{"name": "alpha", "id": "one"}, "raw"}}, format: "text", want: []string{"one\talpha", "raw"}},
		{name: "text scalar", data: 42, format: "text", want: []string{"42"}},
		{name: "table scalar rows", data: []any{"one", "two"}, format: "table", want: []string{"one", "two"}},
		{name: "table scalar", data: true, format: "table", want: []string{"true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := Format(tt.data, tt.format, tt.query, &output); err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(output.String(), want) {
					t.Errorf("output %q does not contain %q", output.String(), want)
				}
			}
		})
	}

	for _, format := range []string{"json", "text", "table"} {
		var output bytes.Buffer
		if err := Format(map[string]any{}, format, "", &output); err != nil {
			t.Fatalf("Format(empty, %s) error = %v", format, err)
		}
		if output.Len() != 0 {
			t.Errorf("Format(empty, %s) output = %q, want empty", format, output.String())
		}
	}
	var output bytes.Buffer
	if err := Format(nil, "json", "", &output); err != nil || output.Len() != 0 {
		t.Fatalf("Format(nil) = output %q, error %v", output.String(), err)
	}
	if err := Format(map[string]any{"id": "x"}, "json", "[", &output); err == nil || !strings.Contains(err.Error(), "JMESPath query error") {
		t.Fatalf("invalid query error = %v", err)
	}
}

func TestFormatJSONReturnsErrorForUnsupportedValue(t *testing.T) {
	var output bytes.Buffer
	err := formatJSON(make(chan int), &output)
	if err == nil {
		t.Fatal("formatJSON(chan) error = nil, want an error")
	}
	if output.Len() != 0 {
		t.Errorf("formatJSON(chan) wrote %q on failure, want nothing written", output.String())
	}
}

// errWriter always fails, so it exercises the very first Write in a render
// path.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type failAfterWriter struct{ n int }

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, errors.New("write failed")
	}
	w.n--
	return len(p), nil
}

func TestFormatPropagatesWriteErrors(t *testing.T) {
	cases := []struct {
		name   string
		data   any
		format string
	}{
		{"json detail", map[string]any{"id": "x"}, "json"},
		{"json list", map[string]any{"items": []any{map[string]any{"id": "x"}}}, "json"},
		{"text detail", map[string]any{"id": "x"}, "text"},
		{"text list", map[string]any{"items": []any{map[string]any{"id": "x"}}}, "text"},
		{"text scalar", 42, "text"},
		{"table detail", map[string]any{"id": "x"}, "table"},
		{"table list", map[string]any{"items": []any{map[string]any{"id": "x"}}}, "table"},
		{"table scalar rows", []any{"one"}, "table"},
		{"table scalar", true, "table"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Format(tc.data, tc.format, "", errWriter{}); err == nil {
				t.Fatalf("Format(%s) with a failing writer: want error, got nil", tc.name)
			}
		})
	}
}

func TestFormatTablePropagatesErrorsAtEveryWriteStep(t *testing.T) {
	rowsData := map[string]any{"items": []any{
		map[string]any{"id": "a"},
		map[string]any{"id": "b"},
	}}
	for n := range 3 { // 0: header fails, 1: separator fails, 2: first row fails
		w := &failAfterWriter{n: n}
		if err := Format(rowsData, "table", "", w); err == nil {
			t.Errorf("rows table with failAfterWriter{%d}: want error, got nil", n)
		}
	}

	detailData := map[string]any{"id": "a", "name": "b"}
	for n := range 3 {
		w := &failAfterWriter{n: n}
		if err := Format(detailData, "table", "", w); err == nil {
			t.Errorf("key/value table with failAfterWriter{%d}: want error, got nil", n)
		}
	}

	if err := Format(rowsData, "text", "", &failAfterWriter{n: 1}); err == nil {
		t.Error("text list with failAfterWriter{1}: want error on the second item, got nil")
	}
}

func TestFormatTableWithColumnsPropagatesWriteErrors(t *testing.T) {
	data := map[string]any{"items": []any{
		map[string]any{"id": "a"},
		map[string]any{"id": "b"},
	}}
	if err := FormatTableWithColumns(data, []string{"id"}, "", errWriter{}); err == nil {
		t.Fatal("FormatTableWithColumns with a failing writer: want error, got nil")
	}
	for n := range 3 {
		w := &failAfterWriter{n: n}
		if err := FormatTableWithColumns(data, []string{"id"}, "", w); err == nil {
			t.Errorf("FormatTableWithColumns with failAfterWriter{%d}: want error, got nil", n)
		}
	}
}

func TestFormatTableWithColumnsHandlesEnvelopesQueriesAndColor(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"displayName": "api", "status": "ACTIVE", "optional": nil},
			map[string]any{"displayName": "worker", "status": "ERROR", "optional": "set"},
		},
	}
	var output bytes.Buffer
	if err := FormatTableWithColumnsColor(data, []string{"displayName", "status", "optional"}, "items", &output, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"DISPLAY NAME", "api", "worker", ansiGreen + "ACTIVE", ansiRed + "ERROR", "set"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("column output missing %q:\n%s", want, output.String())
		}
	}

	output.Reset()
	detail := map[string]any{"data": map[string]any{"id": "server-1", "name": "api"}}
	if err := FormatTableWithColumns(detail, []string{"id", "name"}, "", &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "server-1") {
		t.Errorf("single-object envelope output = %q", output.String())
	}

	for _, value := range []any{nil, map[string]any{}, []any{}, []any{"not-a-map"}} {
		output.Reset()
		if err := FormatTableWithColumns(value, []string{"id"}, "", &output); err != nil {
			t.Fatalf("FormatTableWithColumns(%#v) error = %v", value, err)
		}
		if output.Len() != 0 {
			t.Errorf("FormatTableWithColumns(%#v) output = %q, want empty", value, output.String())
		}
	}
	if err := FormatTableWithColumns(data, []string{"id"}, "[", &output); err == nil {
		t.Fatal("invalid column query succeeded")
	}
}

func TestFormatterHelpersPreserveUnicodeAndDeterminism(t *testing.T) {
	if got := mapKeys(map[string]any{"z": 1, "a": 2}); !reflect.DeepEqual(got, []string{"a", "z"}) {
		t.Errorf("mapKeys() = %#v", got)
	}
	if got := mapValues(map[string]any{"z": 1, "a": 2}); !reflect.DeepEqual(got, []string{"2", "1"}) {
		t.Errorf("mapValues() = %#v", got)
	}
	truncateCases := map[string]string{
		"short":        "short",
		"long value":   "long...",
		"việt nam dài": "việt...",
	}
	for input, want := range truncateCases {
		if got := Truncate(input, 7); got != want {
			t.Errorf("Truncate(%q, 7) = %q, want %q", input, got, want)
		}
	}
	if got := Truncate("abcdef", 3); got != "abc" {
		t.Errorf("Truncate short limit = %q", got)
	}
	for input, want := range map[string]string{
		"2026-09-04T12:30:00+07:00": "2026-09-04",
		"2026-09-04 12:30:00":       "2026-09-04",
		"not-a-date-but-long":       "not-a-date",
		"short":                     "short",
	} {
		if got := ShortDate(input); got != want {
			t.Errorf("ShortDate(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatHeaderPinsASCIIWordSplitting(t *testing.T) {
	cases := map[string]string{
		"id":               "ID",
		"displayName":      "DISPLAY NAME",
		"status":           "STATUS",
		"zoneID":           "ZONE I D",
		"createdAt":        "CREATED AT",
		"ipv4Address":      "IPV4 ADDRESS",
		"name":             "NAME",
		"vpcID":            "VPC I D",
		"createdBy":        "CREATED BY",
		"a":                "A",
		"ABC":              "A B C",
		"snake_case_field": "SNAKE_CASE_FIELD",
	}
	for input, want := range cases {
		if got := formatHeader(input); got != want {
			t.Errorf("formatHeader(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatHeaderUppercasesNonASCII(t *testing.T) {
	cases := map[string]string{
		"café":            "CAFÉ",
		"displayNameViệt": "DISPLAY NAME VIỆT",
		"北京Name":          "北京 NAME",
	}
	for input, want := range cases {
		if got := formatHeader(input); got != want {
			t.Errorf("formatHeader(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPadRightIsRuneBased(t *testing.T) {
	if got, want := padRight("北京", 5), "北京   "; got != want {
		t.Errorf("padRight(CJK) = %q, want %q", got, want)
	}
	if got, want := padRight("ab", 5), "ab   "; got != want {
		t.Errorf("padRight(ascii) = %q, want %q", got, want)
	}
	// Already at/over width: unchanged either way.
	if got := padRight("北京市朝阳区", 3); got != "北京市朝阳区" {
		t.Errorf("padRight(over-width CJK) = %q", got)
	}
}

func TestFormatTableColumnsRuneWidthAlignment(t *testing.T) {
	data := map[string]any{"items": []any{
		map[string]any{"name": "北京", "status": "OK"},              // 2 runes / 6 bytes
		map[string]any{"name": "NewYorkCity", "status": "ACTIVE"}, // 11 runes
		map[string]any{"name": "café", "status": "OK"},            // 4 runes / 5 bytes
	}}
	var buf bytes.Buffer
	if err := FormatTableWithColumns(data, []string{"name", "status"}, "", &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 5 { // header, separator, 3 rows
		t.Fatalf("got %d lines, want 5:\n%q", len(lines), lines)
	}

	const nameWidth, statusWidth = 11, 6
	const wantWidth = nameWidth + 3 + statusWidth // name col + " | " + status col
	for i, l := range lines {
		if n := utf8.RuneCountInString(l); n != wantWidth {
			t.Errorf("line %d = %q (rune width %d), want %d", i, l, n, wantWidth)
		}
		if i == 1 {
			continue // separator line joins with "-+-", not " | "
		}

		parts := strings.SplitN(l, " | ", 2)
		if len(parts) != 2 {
			t.Fatalf("line %d = %q: missing ' | ' column separator", i, l)
		}
		if n := utf8.RuneCountInString(parts[0]); n != nameWidth {
			t.Errorf("line %d name column %q has rune width %d, want %d", i, parts[0], n, nameWidth)
		}
		if n := utf8.RuneCountInString(parts[1]); n != statusWidth {
			t.Errorf("line %d status column %q has rune width %d, want %d", i, parts[1], n, statusWidth)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(strings.SplitN(lines[2], " | ", 2)[0]), "北京") {
		t.Errorf("CJK row missing expected content: %q", lines[2])
	}
}

func TestExtractRowsUnchangedForSingleListResponse(t *testing.T) {
	data := map[string]any{
		"listData": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}},
		"total":    float64(2),
	}
	want := data["listData"]
	for range 20 {
		got := extractRows(data)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("extractRows = %#v, want %#v", got, want)
		}
	}
}

func TestExtractRowsFallsBackToSmallestArrayKeyDeterministically(t *testing.T) {
	data := map[string]any{
		"zeta":  []any{map[string]any{"id": "z"}},
		"alpha": []any{map[string]any{"id": "a"}},
		"mid":   []any{map[string]any{"id": "m"}},
	}
	want := data["alpha"]
	for range 20 {
		got := extractRows(data)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("extractRows = %#v, want the lexicographically-smallest key's value %#v", got, want)
		}
	}
}

func TestExtractRowsPrefersKnownKeyOverFallback(t *testing.T) {
	data := map[string]any{
		"aaaaa": []any{map[string]any{"id": "unknown-earlier"}},
		"items": []any{map[string]any{"id": "known"}},
	}
	want := data["items"]
	for range 20 {
		if got := extractRows(data); !reflect.DeepEqual(got, want) {
			t.Fatalf("extractRows = %#v, want known key's value %#v", got, want)
		}
	}
}

func TestKnownListKeysCanonicalOrder(t *testing.T) {
	want := []string{"items", "listData", "data", "volumeTypes"}
	if got := KnownListKeys(); !reflect.DeepEqual(got, want) {
		t.Errorf("KnownListKeys() = %#v, want %#v", got, want)
	}

	got := KnownListKeys()
	got[0] = "mutated"
	if again := KnownListKeys(); reflect.DeepEqual(again, got) {
		t.Error("KnownListKeys() returned a shared slice; mutation leaked into the next call")
	}
}

type normalizeRow struct {
	Group   string `json:"group"`
	BaseURL string `json:"baseUrl"`
	Count   int    `json:"count"`
}

func TestFormatQueryUsesJSONTagsNotFieldNames(t *testing.T) {
	rows := []normalizeRow{
		{Group: "access", BaseURL: "https://example.test/identity", Count: 1},
		{Group: "memory", BaseURL: "https://example.test/memory", Count: 2},
	}

	for _, tc := range []struct {
		query string
		want  string
	}{
		{"[].baseUrl", "https://example.test/identity"},
		{"[0].baseUrl", "https://example.test/identity"},
		{"[?group=='memory'].baseUrl | [0]", "https://example.test/memory"},
		{"[].group", "access"},
	} {
		var sb strings.Builder
		if err := Format(rows, "json", tc.query, &sb); err != nil {
			t.Fatalf("Format(query=%q) error = %v", tc.query, err)
		}
		if got := sb.String(); !strings.Contains(got, tc.want) {
			t.Errorf("Format(query=%q) = %q, want it to contain %q", tc.query, got, tc.want)
		}
	}
}

func TestFormatRendersStructRowsAsTable(t *testing.T) {
	rows := []normalizeRow{{Group: "access", BaseURL: "https://example.test/identity", Count: 1}}

	var table strings.Builder
	if err := Format(rows, "table", "", &table); err != nil {
		t.Fatalf("Format(table) error = %v", err)
	}
	got := table.String()
	if strings.Contains(got, "{access") {
		t.Errorf("table output is a Go value dump:\n%s", got)
	}
	for _, header := range []string{"baseUrl", "count", "group"} {
		if !strings.Contains(got, header) {
			t.Errorf("table output is missing the %q column:\n%s", header, got)
		}
	}

	var text strings.Builder
	if err := Format(rows, "text", "", &text); err != nil {
		t.Fatalf("Format(text) error = %v", err)
	}
	if strings.Contains(text.String(), "{access") {
		t.Errorf("text output is a Go value dump:\n%s", text.String())
	}
}

func TestFormatNormalizesIntegersWithoutExponent(t *testing.T) {
	for _, format := range []string{"json", "text", "table"} {
		var sb strings.Builder
		if err := Format(map[string]any{"bytes": 1048576}, format, "", &sb); err != nil {
			t.Fatalf("Format(%s) error = %v", format, err)
		}
		if got := sb.String(); !strings.Contains(got, "1048576") {
			t.Errorf("Format(%s) = %q, want the exact integer 1048576", format, got)
		}
	}
}

func TestFormatRejectsUnserializableValue(t *testing.T) {
	var sb strings.Builder
	if err := Format(map[string]any{"ch": make(chan int)}, "table", "", &sb); err == nil {
		t.Fatal("Format accepted an unserializable value, want an error")
	}
}

func TestFormatTableWithColumnsNormalizes(t *testing.T) {
	rows := []normalizeRow{{Group: "access", BaseURL: "https://example.test/identity", Count: 1}}
	var sb strings.Builder
	if err := FormatTableWithColumns(rows, []string{"group", "baseUrl"}, "", &sb); err != nil {
		t.Fatalf("FormatTableWithColumns error = %v", err)
	}
	got := sb.String()
	if !strings.Contains(got, "access") || !strings.Contains(got, "https://example.test/identity") {
		t.Errorf("column table did not render struct rows:\n%s", got)
	}
}

func TestFormatRendersDecodedJSONNumbersWithoutExponent(t *testing.T) {
	var decoded any
	if err := json.Unmarshal([]byte(`[{"name":"vol-1","sizeBytes":1099511627776,"port":8080,"ratio":1.5}]`), &decoded); err != nil {
		t.Fatal(err)
	}

	for _, format := range []string{"text", "table"} {
		var sb strings.Builder
		if err := Format(decoded, format, "", &sb); err != nil {
			t.Fatalf("Format(%s) error = %v", format, err)
		}
		got := sb.String()
		if strings.Contains(got, "e+") {
			t.Errorf("Format(%s) used exponent notation:\n%s", format, got)
		}
		for _, want := range []string{"1099511627776", "8080", "1.5"} {
			if !strings.Contains(got, want) {
				t.Errorf("Format(%s) = %q, want it to contain %q", format, got, want)
			}
		}
	}
}

func TestFormatPreservesLargeIntegerPrecision(t *testing.T) {
	const exact int64 = 9007199254740993 // 2^53 + 1: the first int64 float64 cannot represent
	var rounded strings.Builder
	if err := Format(map[string]any{"id": float64(exact)}, "json", "", &rounded); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rounded.String(), "9007199254740993") {
		t.Skip("float64 represents 2^53+1 exactly on this platform; the test premise does not hold")
	}

	for _, format := range []string{"json", "text", "table"} {
		var sb strings.Builder
		if err := Format(map[string]any{"id": exact}, format, "", &sb); err != nil {
			t.Fatalf("Format(%s) error = %v", format, err)
		}
		if got := sb.String(); !strings.Contains(got, "9007199254740993") {
			t.Errorf("Format(%s) = %q, want the exact value 9007199254740993", format, got)
		}
	}

	var uns strings.Builder
	if err := Format(map[string]any{"id": uint64(18446744073709551615)}, "json", "", &uns); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uns.String(), "18446744073709551615") {
		t.Errorf("uint64 max = %q, want the exact value", uns.String())
	}
}

func TestQueryStillComparesNumbers(t *testing.T) {
	var decoded any
	if err := json.Unmarshal([]byte(`[{"name":"a","size":10},{"name":"b","size":300}]`), &decoded); err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	if err := Format(decoded, "json", "[?size > `100`].name | [0]", &sb); err != nil {
		t.Fatalf("numeric filter error = %v", err)
	}
	if !strings.Contains(sb.String(), "b") {
		t.Errorf("numeric filter = %q, want it to select \"b\"", sb.String())
	}
}
