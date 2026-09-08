package formatter

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// A detail (single-object) response with nested array fields must render as a
// key/value table — not be hijacked into an empty table by a nested array.
func TestFormatTableDetailObject(t *testing.T) {
	data := map[string]interface{}{
		"id":            "k8s-123",
		"name":          "demo",
		"status":        "ACTIVE",
		"listSubnetIds": []interface{}{}, // empty nested array — must NOT blank the output
	}
	var buf bytes.Buffer
	formatTable(data, &buf, false)
	out := buf.String()

	if strings.TrimSpace(out) == "" {
		t.Fatalf("detail object rendered empty table")
	}
	for _, want := range []string{"id", "k8s-123", "name", "demo", "status", "ACTIVE"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\n%s", want, out)
		}
	}
}

// A list response ({items:[...]}) must render as a multi-column table with a
// header row and one row per item.
func TestFormatTableListResponse(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": "c1", "name": "alpha"},
			map[string]interface{}{"id": "c2", "name": "beta"},
		},
		"total": float64(2),
	}
	var buf bytes.Buffer
	formatTable(data, &buf, false)
	out := buf.String()

	for _, want := range []string{"id", "name", "c1", "alpha", "c2", "beta"} {
		if !strings.Contains(out, want) {
			t.Errorf("list table missing %q\n%s", want, out)
		}
	}
	// header order must be deterministic (sorted): "id" column before "name"
	if strings.Index(out, "id") > strings.Index(out, "name") {
		t.Errorf("headers not sorted deterministically:\n%s", out)
	}
}

// Top-level array renders as a multi-column table too.
func TestFormatTableTopLevelArray(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"uuid": "s1"},
		map[string]interface{}{"uuid": "s2"},
	}
	var buf bytes.Buffer
	formatTable(data, &buf, false)
	out := buf.String()
	for _, want := range []string{"uuid", "s1", "s2"} {
		if !strings.Contains(out, want) {
			t.Errorf("top-array table missing %q\n%s", want, out)
		}
	}
}

// TestExtractRowsIgnoresUnrelatedArrayFields is a regression test for a detail
// object whose own array field was mistaken for the row set, producing empty
// --output table. Found against the live VLB API: `certificate get` returns a
// bare object carrying subjectAlternativeNames, and the table came out blank.
func TestExtractRowsIgnoresUnrelatedArrayFields(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantRows int
		wantKey  string // a key the first row must have
	}{
		{
			name: "bare detail object with an array field is one row",
			data: map[string]interface{}{
				"uuid":                    "secret-1",
				"name":                    "web-cert",
				"subjectAlternativeNames": []interface{}{"app.example.com"},
			},
			wantRows: 1,
			wantKey:  "uuid",
		},
		{
			name: "data-wrapped detail object with an array field is one row",
			data: map[string]interface{}{"data": map[string]interface{}{
				"uuid":    "pool-1",
				"members": []interface{}{map[string]interface{}{"uuid": "m-1"}},
			}},
			wantRows: 1,
			wantKey:  "uuid",
		},
		{
			name:     "listData envelope yields its items",
			data:     map[string]interface{}{"listData": []interface{}{map[string]interface{}{"uuid": "lb-1"}, map[string]interface{}{"uuid": "lb-2"}}},
			wantRows: 2,
			wantKey:  "uuid",
		},
		{
			name:     "data array envelope yields its items",
			data:     map[string]interface{}{"data": []interface{}{map[string]interface{}{"uuid": "li-1"}}},
			wantRows: 1,
			wantKey:  "uuid",
		},
		{
			name:     "top-level array is the row set",
			data:     []interface{}{map[string]interface{}{"id": float64(1)}},
			wantRows: 1,
			wantKey:  "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := extractRows(tt.data)
			if len(rows) != tt.wantRows {
				t.Fatalf("extractRows returned %d rows, want %d: %#v", len(rows), tt.wantRows, rows)
			}
			m, ok := rows[0].(map[string]interface{})
			if !ok {
				t.Fatalf("row 0 is %T, want an object — a non-object row makes the table render nothing", rows[0])
			}
			if _, ok := m[tt.wantKey]; !ok {
				t.Errorf("row 0 has no %q; got keys %v", tt.wantKey, mapKeys(m))
			}
		})
	}
}

// TestExtractRowsIsDeterministic guards against the old "first array field
// found" scan, whose result depended on Go's randomised map iteration order.
func TestExtractRowsIsDeterministic(t *testing.T) {
	data := map[string]interface{}{
		"uuid": "secret-1",
		"a":    []interface{}{"x"},
		"b":    []interface{}{"y"},
		"c":    []interface{}{map[string]interface{}{"uuid": "nested"}},
	}
	first := extractRows(data)
	for i := 0; i < 200; i++ {
		got := extractRows(data)
		if len(got) != len(first) {
			t.Fatalf("iteration %d returned %d rows, first call returned %d", i, len(got), len(first))
		}
		if !reflect.DeepEqual(got[0], first[0]) {
			t.Fatalf("iteration %d returned a different row than the first call", i)
		}
	}
	if len(first) != 1 {
		t.Errorf("a detail object should be a single row, got %d", len(first))
	}
}

// TestFormatTableWithColumnsRendersBareObject is the end-to-end form of the
// regression: the command-level symptom was no output at all.
func TestFormatTableWithColumnsRendersBareObject(t *testing.T) {
	data := map[string]interface{}{
		"uuid":                    "secret-1",
		"name":                    "web-cert",
		"certificateType":         "TLS/SSL",
		"inUse":                   false,
		"subjectAlternativeNames": []interface{}{"app.example.com"},
	}
	var buf bytes.Buffer
	if err := FormatTableWithColumnsColor(data, []string{"uuid", "name", "certificateType", "inUse"}, "", &buf, false); err != nil {
		t.Fatalf("FormatTableWithColumnsColor: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("rendered nothing for a bare detail object")
	}
	for _, want := range []string{"UUID", "NAME", "secret-1", "web-cert", "TLS/SSL", "false"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// JSON numbers arrive as float64, and fmt.Sprint switches to scientific
// notation for large magnitudes — an epoch-millisecond timestamp such as a
// certificate's notAfter rendered as "1.781232534e+12" in a table cell.
func TestCellStringDoesNotUseScientificNotation(t *testing.T) {
	for _, tc := range []struct {
		in   interface{}
		want string
	}{
		{float64(1781232534000), "1781232534000"},
		{float64(2000), "2000"},
		{float64(0), "0"},
		{float64(-1781232534000), "-1781232534000"},
		{float64(1.5), "1.5"},
		{"already a string", "already a string"},
		{true, "true"},
		{nil, "<nil>"},
	} {
		if got := cellString(tc.in); got != tc.want {
			t.Errorf("cellString(%#v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
