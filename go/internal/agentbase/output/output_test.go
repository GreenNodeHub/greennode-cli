package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout to a buffer for the duration of f.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestParseFormat_JSON(t *testing.T) {
	if ParseFormat("json") != FormatJSON {
		t.Error("expected FormatJSON")
	}
}

func TestParseFormat_Default(t *testing.T) {
	if ParseFormat("") != FormatTable {
		t.Error("expected FormatTable for empty string")
	}
	if ParseFormat("something") != FormatTable {
		t.Error("expected FormatTable for unknown string")
	}
}

func TestJSON_Output(t *testing.T) {
	type payload struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := captureStdout(t, func() {
		if err := JSON(payload{ID: "1", Name: "test"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	var decoded payload
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v (got %q)", err, out)
	}
	if decoded.ID != "1" || decoded.Name != "test" {
		t.Errorf("unexpected decoded value: %+v", decoded)
	}
}

func TestSuccess(t *testing.T) {
	out := captureStdout(t, func() {
		Success("it worked")
	})
	if !strings.Contains(out, "it worked") {
		t.Errorf("expected 'it worked' in output, got %q", out)
	}
}

func TestSuccessf(t *testing.T) {
	out := captureStdout(t, func() {
		Successf("value: %d", 42)
	})
	if !strings.Contains(out, "42") {
		t.Errorf("expected '42' in output, got %q", out)
	}
}

func TestPrintID(t *testing.T) {
	out := captureStdout(t, func() {
		PrintID("abc-123")
	})
	if !strings.Contains(out, "abc-123") {
		t.Errorf("expected 'abc-123' in output, got %q", out)
	}
}

func TestStrOrDash_NonEmpty(t *testing.T) {
	if got := StrOrDash("hello"); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStrOrDash_Empty(t *testing.T) {
	if got := StrOrDash(""); got != "-" {
		t.Errorf("expected '-', got %q", got)
	}
}

func TestTable_Renders(t *testing.T) {
	out := captureStdout(t, func() {
		Table([]string{"ID", "Name"}, [][]string{
			{"1", "alpha"},
			{"2", "beta"},
		})
	})
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("expected table rows in output, got %q", out)
	}
}

func TestWarn_WritesToStderr(t *testing.T) {
	// Warn writes to stderr; we just call it to ensure it doesn't panic.
	Warn("test warning")
}

type mockStringer struct{ val string }

func (m mockStringer) String() string { return m.val }

func TestFormatTime_NonNil(t *testing.T) {
	got := FormatTime(mockStringer{val: "2024-01-01"})
	if got != "2024-01-01" {
		t.Errorf("expected '2024-01-01', got %q", got)
	}
}

func TestParseFormat_ID(t *testing.T) {
	if ParseFormat("id") != FormatID {
		t.Errorf("expected FormatID, got %q", ParseFormat("id"))
	}
}

func TestParseFormat_Table(t *testing.T) {
	for _, s := range []string{"table", "", "unknown"} {
		if ParseFormat(s) != FormatTable {
			t.Errorf("ParseFormat(%q) = %q, want FormatTable", s, ParseFormat(s))
		}
	}
}

func TestSetGetFormat(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	cases := []struct {
		input string
		want  Format
	}{
		{"json", FormatJSON},
		{"id", FormatID},
		{"table", FormatTable},
		{"", FormatTable},
	}
	for _, tc := range cases {
		SetFormat(ParseFormat(tc.input))
		if got := GetFormat(); got != tc.want {
			t.Errorf("after SetFormat(ParseFormat(%q)): GetFormat() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestPrintResource_JSON(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatJSON)
	type res struct {
		ID string `json:"id"`
	}
	humanCalled := false
	out := captureStdout(t, func() {
		err := PrintResource(res{ID: "abc"}, func() string { return "abc" }, func() error {
			humanCalled = true
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if humanCalled {
		t.Error("humanFn should not be called for JSON format")
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v (got %q)", err, out)
	}
	if decoded["id"] != "abc" {
		t.Errorf("expected id=abc, got %+v", decoded)
	}
}

func TestPrintResource_ID(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatID)
	humanCalled := false
	out := captureStdout(t, func() {
		err := PrintResource(struct{}{}, func() string { return "xyz-123" }, func() error {
			humanCalled = true
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if humanCalled {
		t.Error("humanFn should not be called for ID format")
	}
	if strings.TrimSpace(out) != "xyz-123" {
		t.Errorf("expected 'xyz-123', got %q", out)
	}
}

func TestPrintResource_Table(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatTable)
	humanCalled := false
	captureStdout(t, func() {
		err := PrintResource(struct{}{}, func() string { return "ignored" }, func() error {
			humanCalled = true
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !humanCalled {
		t.Error("humanFn must be called for table format")
	}
}

func TestPrintDeletedID_JSON(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatJSON)
	out := captureStdout(t, func() {
		if err := PrintDeletedID("del-456"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	var decoded map[string]string
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v (got %q)", err, out)
	}
	if decoded["id"] != "del-456" {
		t.Errorf("expected id=del-456, got %+v", decoded)
	}
}

func TestPrintDeletedID_ID(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatID)
	out := captureStdout(t, func() {
		if err := PrintDeletedID("del-456"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(out) != "del-456" {
		t.Errorf("expected 'del-456', got %q", out)
	}
}

func TestPrintDeletedID_Table(t *testing.T) {
	t.Cleanup(func() { SetFormat(FormatTable) })
	SetFormat(FormatTable)
	out := captureStdout(t, func() {
		if err := PrintDeletedID("del-456"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if out != "" {
		t.Errorf("expected empty stdout in table mode, got %q", out)
	}
}

func TestSetGetQuery(t *testing.T) {
	t.Cleanup(func() { SetQuery("") })
	SetQuery("items[0].id")
	if got := GetQuery(); got != "items[0].id" {
		t.Errorf("GetQuery() = %q, want items[0].id", got)
	}
	SetQuery("")
	if got := GetQuery(); got != "" {
		t.Errorf("GetQuery() after reset = %q, want empty", got)
	}
}

// TestJSON_WithQuery: a JMESPath query filters JSON output.
func TestJSON_WithQuery(t *testing.T) {
	t.Cleanup(func() { SetQuery("") })
	SetQuery("items[*].id")
	out := captureStdout(t, func() {
		err := JSON(map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"id": "a"},
				map[string]interface{}{"id": "b"},
			},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	var ids []string
	if err := json.Unmarshal([]byte(out), &ids); err != nil {
		t.Fatalf("output is not a JSON array of ids: %v (got %q)", err, out)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("expected [a b], got %v", ids)
	}
}

// TestJSON_WithQueryOnStruct proves the marshal→decode round-trip: the query
// is written against JSON field names, not Go struct fields.
func TestJSON_WithQueryOnStruct(t *testing.T) {
	t.Cleanup(func() { SetQuery("") })
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	SetQuery("items[0].name")
	out := captureStdout(t, func() {
		err := JSON(map[string]interface{}{"items": []item{{ID: "1", Name: "alpha"}}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(out) != `"alpha"` {
		t.Errorf(`expected "alpha", got %q`, out)
	}
}

// TestJSON_WithQueryInvalid: a bad JMESPath expression fails with an error
// naming the --query flag.
func TestJSON_WithQueryInvalid(t *testing.T) {
	t.Cleanup(func() { SetQuery("") })
	SetQuery("items[")
	err := JSON(map[string]interface{}{"id": "a"})
	if err == nil {
		t.Fatal("expected an error for an invalid JMESPath expression")
	}
	if !strings.Contains(err.Error(), "--query") {
		t.Errorf("error must name the offending flag, got %q", err)
	}
}
