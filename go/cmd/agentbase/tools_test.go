package agentbase

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	toolspkg "github.com/greennodehub/greennode-cli/internal/agentbase/tools"
)

// newInvokeArgsCmd builds a command carrying 'session invoke's argument
// flags (mirroring how cobra parses --content a=b --content-b64 c=d) and sets
// them from values.
func newInvokeArgsCmd(t *testing.T, values map[string][]string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "invoke"}
	cmd.Flags().String("language", "", "")
	cmd.Flags().String("code", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("directory-path", "", "")
	cmd.Flags().StringSlice("content", nil, "")
	cmd.Flags().StringSlice("content-b64", nil, "")
	cmd.Flags().StringSlice("path", nil, "")
	cmd.Flags().String("task-id", "", "")
	for name, items := range values {
		for _, v := range items {
			if err := cmd.Flags().Set(name, v); err != nil {
				t.Fatalf("setting --%s=%s: %v", name, v, err)
			}
		}
	}
	return cmd
}

// TestToolsInvokeToolArgs_WriteFiles_EncodesContent: --content carries plain
// text; the CLI must base64-encode it into the writeFiles blob.
func TestToolsInvokeToolArgs_WriteFiles_EncodesContent(t *testing.T) {
	cmd := newInvokeArgsCmd(t, map[string][]string{"content": {"hello.txt=hello world"}})
	args, err := toolsInvokeToolArgs(cmd, "writeFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args.Content) != 1 {
		t.Fatalf("expected 1 file, got %d", len(args.Content))
	}
	if args.Content[0].Path != "hello.txt" {
		t.Errorf("path: got %q, want hello.txt", args.Content[0].Path)
	}
	if want := "aGVsbG8gd29ybGQ="; args.Content[0].Blob != want { // base64("hello world")
		t.Errorf("blob: got %q, want %q", args.Content[0].Blob, want)
	}
}

// TestToolsInvokeToolArgs_WriteFiles_EncodesOnlyAfterFirstEquals: the split
// is on the first '=', so '=' inside the content stays part of the payload
// (it is encoded, not treated as another separator).
func TestToolsInvokeToolArgs_WriteFiles_EncodesOnlyAfterFirstEquals(t *testing.T) {
	cmd := newInvokeArgsCmd(t, map[string][]string{"content": {"env=KEY=value"}})
	args, err := toolsInvokeToolArgs(cmd, "writeFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := args.Content[0]
	if f.Path != "env" {
		t.Errorf("path: got %q, want env", f.Path)
	}
	if want := "S0VZPXZhbHVl"; f.Blob != want { // base64("KEY=value")
		t.Errorf("blob: got %q, want %q", f.Blob, want)
	}
}

// TestToolsInvokeToolArgs_WriteFiles_ContentB64Passthrough: pre-encoded
// blobs are passed through untouched (no double encoding).
func TestToolsInvokeToolArgs_WriteFiles_ContentB64Passthrough(t *testing.T) {
	blob := "AAECAwQ="
	cmd := newInvokeArgsCmd(t, map[string][]string{"content-b64": {"bin.dat=" + blob}})
	args, err := toolsInvokeToolArgs(cmd, "writeFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args.Content) != 1 || args.Content[0].Path != "bin.dat" {
		t.Fatalf("unexpected files: %+v", args.Content)
	}
	if args.Content[0].Blob != blob {
		t.Errorf("blob must pass through: got %q, want %q", args.Content[0].Blob, blob)
	}
}

// TestToolsInvokeToolArgs_WriteFiles_MixedSources: --content and
// --content-b64 combine, each handled by its own rule.
func TestToolsInvokeToolArgs_WriteFiles_MixedSources(t *testing.T) {
	cmd := newInvokeArgsCmd(t, map[string][]string{
		"content":     {"a.txt=hi"},
		"content-b64": {"b.bin=AAE="},
	})
	args, err := toolsInvokeToolArgs(cmd, "writeFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args.Content) != 2 {
		t.Fatalf("expected 2 files, got %d", len(args.Content))
	}
	if args.Content[0].Blob != "aGk=" { // base64("hi")
		t.Errorf("plain content must be encoded, got %q", args.Content[0].Blob)
	}
	if args.Content[1].Blob != "AAE=" {
		t.Errorf("pre-encoded content must pass through, got %q", args.Content[1].Blob)
	}
}

// TestToolsInvokeToolArgs_WriteFiles_RejectsInvalid: every malformed
// --content/--content-b64 value fails client-side with a message naming the
// offending flag and the expected format.
func TestToolsInvokeToolArgs_WriteFiles_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name   string
		values map[string][]string
		want   string
	}{
		{"no content", map[string][]string{}, "--tool writeFiles requires"},
		{"missing equals", map[string][]string{"content": {"justpath"}}, `invalid --content "justpath"`},
		{"empty path", map[string][]string{"content": {"=x"}}, `invalid --content "=x"`},
		{"empty text", map[string][]string{"content": {"f.txt="}}, `invalid --content "f.txt="`},
		{"b64 missing equals", map[string][]string{"content-b64": {"justpath"}}, `invalid --content-b64 "justpath"`},
		{"invalid base64", map[string][]string{"content-b64": {"f.txt=@@@@"}}, "blob is not valid base64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := toolsInvokeToolArgs(newInvokeArgsCmd(t, tc.values), "writeFiles")
			if err == nil {
				t.Fatalf("expected an error mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q must mention %q", err, tc.want)
			}
		})
	}
}

// TestValidateToolsMetadata: name/description pass at their limits and fail
// one char over, with errors naming the offending flag.
func TestValidateToolsMetadata(t *testing.T) {
	if err := validateToolsMetadata(strings.Repeat("n", 255), strings.Repeat("d", 1000)); err != nil {
		t.Errorf("at-limit values must pass: %v", err)
	}
	if err := validateToolsMetadata("", ""); err != nil {
		t.Errorf("empty values must pass: %v", err)
	}
	if err := validateToolsMetadata(strings.Repeat("n", 256), ""); err == nil || !strings.Contains(err.Error(), "--name") {
		t.Errorf("expected an --name error, got %v", err)
	}
	if err := validateToolsMetadata("", strings.Repeat("d", 1001)); err == nil || !strings.Contains(err.Error(), "--description") {
		t.Errorf("expected an --description error, got %v", err)
	}
}

// TestValidatePageParams: pages are 1-based; zero/negative values fail with
// errors naming the offending flag.
func TestValidatePageParams(t *testing.T) {
	if err := validatePageParams(1, 10); err != nil {
		t.Errorf("defaults must pass: %v", err)
	}
	cases := []struct {
		name       string
		page, size int
		want       string
	}{
		{"zero page", 0, 10, "--page 0"},
		{"negative page", -1, 10, "--page -1"},
		{"zero size", 1, 0, "--size 0"},
		{"negative size", 1, -5, "--size -5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePageParams(tc.page, tc.size)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error mentioning %q, got %v", tc.want, err)
			}
		})
	}
}

// TestFlagEndUserID: --end-user-id is optional but, when set, must match the
// server's charset (1-128 chars of A-Za-z0-9._:@-).
func TestFlagEndUserID(t *testing.T) {
	newCmd := func(v string) *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("end-user-id", "", "")
		if v != "" {
			if err := cmd.Flags().Set("end-user-id", v); err != nil {
				t.Fatalf("setting --end-user-id: %v", err)
			}
		}
		return cmd
	}
	for _, valid := range []string{"", "user-1", "user@tenant.io", "u.1:_a-b@x", strings.Repeat("u", 128)} {
		got, err := flagEndUserID(newCmd(valid))
		if err != nil {
			t.Errorf("%q must be valid: %v", valid, err)
		}
		if got != valid {
			t.Errorf("got %q, want %q", got, valid)
		}
	}
	for _, invalid := range []string{"user 1", "user/1", strings.Repeat("u", 129)} {
		_, err := flagEndUserID(newCmd(invalid))
		if err == nil || !strings.Contains(err.Error(), "--end-user-id") {
			t.Errorf("%q must fail with an --end-user-id error, got %v", invalid, err)
		}
	}
}

// TestToolsTaskHelpers: task-wait's readers pull taskId/taskStatus/exitCode
// out of structuredContent and tolerate their absence.
func TestToolsTaskHelpers(t *testing.T) {
	sc := map[string]interface{}{
		"taskId":     "t-1",
		"taskStatus": "completed",
		"exitCode":   float64(2),
	}
	r := &toolspkg.InvokeResponse{StructuredContent: &sc}
	if got := toolsTaskStatus(r); got != "completed" {
		t.Errorf("toolsTaskStatus: got %q, want completed", got)
	}
	if got := toolsInvokeTaskID(r); got != "t-1" {
		t.Errorf("toolsInvokeTaskID: got %q, want t-1", got)
	}
	if code, ok := toolsTaskExitCode(r); !ok || code != 2 {
		t.Errorf("toolsTaskExitCode: got (%d, %v), want (2, true)", code, ok)
	}

	empty := &toolspkg.InvokeResponse{}
	if got := toolsTaskStatus(empty); got != "" {
		t.Errorf("toolsTaskStatus on empty: got %q, want empty", got)
	}
	if got := toolsInvokeTaskID(empty); got != "" {
		t.Errorf("toolsInvokeTaskID on empty: got %q, want empty", got)
	}
	if code, ok := toolsTaskExitCode(empty); ok || code != 0 {
		t.Errorf("toolsTaskExitCode on empty: got (%d, %v), want (0, false)", code, ok)
	}

	wrongTypes := map[string]interface{}{"taskStatus": 42, "exitCode": "1", "taskId": true}
	r2 := &toolspkg.InvokeResponse{StructuredContent: &wrongTypes}
	if got := toolsTaskStatus(r2); got != "" {
		t.Errorf("toolsTaskStatus on wrong type: got %q, want empty", got)
	}
	if code, ok := toolsTaskExitCode(r2); ok || code != 0 {
		t.Errorf("toolsTaskExitCode on wrong type: got (%d, %v), want (0, false)", code, ok)
	}
	if got := toolsInvokeTaskID(r2); got != "" {
		t.Errorf("toolsInvokeTaskID on wrong type: got %q, want empty", got)
	}
}

// TestConfirmToolsDelete: confirmation accepts y/yes (any case) and nothing
// else; EOF without --force must fail closed.
func TestConfirmToolsDelete(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase y", "y\n", true},
		{"uppercase Y", "Y", true},
		{"yes", "YES\n", true},
		{"no", "n\n", false},
		{"empty line", "\n", false},
		{"random text", "sure?\n", false},
		{"eof", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "grn-stdin-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			if _, err := f.WriteString(tc.input); err != nil {
				t.Fatal(err)
			}
			if _, err := f.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			orig := os.Stdin
			os.Stdin = f
			got := confirmToolsDelete("Delete?")
			os.Stdin = orig
			f.Close()
			if got != tc.want {
				t.Errorf("input %q: got %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestToolsDeleteCmds_SafetyFlags: both delete commands must register the
// confirm/dry-run safety flags their RunE reads (a missing registration
// panics at runtime).
func TestToolsDeleteCmds_SafetyFlags(t *testing.T) {
	for name, cmd := range map[string]*cobra.Command{
		"toolsBrowserDeleteCmd":         toolsBrowserDeleteCmd,
		"toolsCodeInterpreterDeleteCmd": toolsCodeInterpreterDeleteCmd,
	} {
		for _, flag := range []string{"force", "dry-run"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Errorf("%s: missing --%s flag", name, flag)
			}
		}
	}
}

// TestToolsCISessionTaskWaitCmd_Registered: task-wait must be mounted under
// 'code-interpreter session' with a required --task-id and --timeout/--interval.
func TestToolsCISessionTaskWaitCmd_Registered(t *testing.T) {
	if !subCmdExists(toolsCISessionCmd, "task-wait") {
		t.Fatal("code-interpreter session missing 'task-wait'")
	}
	taskIDFlag := toolsCISessionTaskWaitCmd.Flags().Lookup("task-id")
	if taskIDFlag == nil {
		t.Fatal("task-wait missing --task-id")
	}
	if ann := taskIDFlag.Annotations["cobra_annotation_bash_completion_one_required_flag"]; len(ann) == 0 || ann[0] != "true" {
		t.Error("task-wait --task-id must be required")
	}
	for _, flag := range []string{"timeout", "interval"} {
		if toolsCISessionTaskWaitCmd.Flags().Lookup(flag) == nil {
			t.Errorf("task-wait missing --%s", flag)
		}
	}
}

// TestApplyAgentbaseQuery covers agentbase.go: grn's global --query (JMESPath)
// is passed to the output stack for JSON and rejected for table/id (it would
// otherwise be silently ignored).
func TestApplyAgentbaseQuery(t *testing.T) {
	t.Cleanup(func() {
		output.SetFormat(output.FormatTable)
		output.SetQuery("")
	})
	newCmd := func(query string) *cobra.Command {
		cmd := &cobra.Command{Use: "probe"}
		cmd.Flags().String("query", "", "")
		if query != "" {
			if err := cmd.Flags().Set("query", query); err != nil {
				t.Fatalf("setting --query: %v", err)
			}
		}
		return cmd
	}

	output.SetFormat(output.FormatJSON)
	if err := applyAgentbaseQuery(newCmd("items[0].id")); err != nil {
		t.Errorf("json format must accept --query: %v", err)
	}
	if got := output.GetQuery(); got != "items[0].id" {
		t.Errorf("query not stored: %q", got)
	}

	for _, f := range []output.Format{output.FormatTable, output.FormatID} {
		output.SetFormat(f)
		err := applyAgentbaseQuery(newCmd("id"))
		if err == nil || !strings.Contains(err.Error(), "-o json") {
			t.Errorf("format %v with --query must fail with an -o json hint, got %v", f, err)
		}
	}

	output.SetFormat(output.FormatTable)
	if err := applyAgentbaseQuery(newCmd("")); err != nil {
		t.Errorf("no query must always pass: %v", err)
	}
	if got := output.GetQuery(); got != "" {
		t.Errorf("empty query must reset, got %q", got)
	}
}

// TestToolsInvokeToolArgs_ListFiles_DefaultsDirectoryPath: the backend
// requires directoryPath (400 with -32602 when omitted) and the platform MCP
// server defaults it to "." — a bare listFiles must list the session working
// directory instead of erroring (34837).
func TestToolsInvokeToolArgs_ListFiles_DefaultsDirectoryPath(t *testing.T) {
	cmd := newInvokeArgsCmd(t, nil) // no --directory-path at all
	args, err := toolsInvokeToolArgs(cmd, "listFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.DirectoryPath != "." {
		t.Errorf("directoryPath: got %q, want \".\"", args.DirectoryPath)
	}

	cmd = newInvokeArgsCmd(t, map[string][]string{"directory-path": {"/tmp/work"}})
	args, err = toolsInvokeToolArgs(cmd, "listFiles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.DirectoryPath != "/tmp/work" {
		t.Errorf("explicit directoryPath: got %q, want /tmp/work", args.DirectoryPath)
	}
}
