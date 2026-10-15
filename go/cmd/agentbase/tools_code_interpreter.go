package agentbase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	toolspkg "github.com/greennodehub/greennode-cli/internal/agentbase/tools"
)

// toolsCodeInterpreterCmd groups the code-interpreter commands. Mirrors the
// browser group minus the stream endpoints, plus 'session invoke' — the only
// built-in tool surface that executes work directly.
var toolsCodeInterpreterCmd = &cobra.Command{
	Use:   "code-interpreter",
	Short: "Manage code interpreters",
	Long: `Manage platform-managed code-execution sandboxes.

Create is asynchronous: 201 CREATING → poll with 'wait' until ACTIVE (or
FAILED). Sessions attach to a code interpreter and are the unit of execution:
'code-interpreter session invoke' runs a tool (executeCode, executeCommand,
listFiles, writeFiles, readFiles, getTask, ...) inside a session.`,
}

var toolsCodeInterpreterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List code interpreters",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		if err := validatePageParams(page, size); err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.ListCodeInterpreters(ctx, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(result)
		case output.FormatID:
			if len(result.ListData) > 0 {
				output.PrintID(result.ListData[0].ID)
			}
			return nil
		}
		if len(result.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No code interpreters found.")
			return nil
		}
		rows := make([][]string, 0, len(result.ListData))
		for i := range result.ListData {
			ci := result.ListData[i]
			rows = append(rows, []string{
				ci.ID, ci.Name, output.StrOrDash(ci.Description), ci.Status,
				output.FormatTime(ci.CreatedAt),
			})
		}
		output.Table([]string{"ID", "Name", "Description", "Status", "Created"}, rows)
		printToolsPageFooter(result.Page, result.PageSize, result.TotalItem, result.TotalPage)
		return nil
	},
}

var toolsCodeInterpreterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a code interpreter",
	Long: `Create a code-execution sandbox.

Creation is asynchronous: the API returns the code interpreter in CREATING;
run 'grn agentbase tools code-interpreter wait <id>' until it is ACTIVE before
attaching sessions (creating a session on a non-ACTIVE code interpreter fails
with 409).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Code interpreter name")
		if err != nil {
			return err
		}
		if err := validateToolsMetadata(name, description); err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		ci, err := client.CreateCodeInterpreter(ctx, &toolspkg.CreateCodeInterpreterRequest{Name: name, Description: description})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Code interpreter %q created (id %s, status %s).\n", ci.Name, ci.ID, ci.Status)
		fmt.Fprintln(os.Stderr, "Run 'grn agentbase tools code-interpreter wait "+ci.ID+"' to wait for ACTIVE.")
		return output.PrintResource(ci, func() string { return ci.ID }, func() error { return renderToolsCodeInterpreterDetail(ci) })
	},
}

var toolsCodeInterpreterGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a code interpreter",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		ci, err := client.GetCodeInterpreter(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(ci, func() string { return ci.ID }, func() error { return renderToolsCodeInterpreterDetail(ci) })
	},
}

var toolsCodeInterpreterDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a code interpreter",
	Long: `Delete a code interpreter.

Deleting a code interpreter also deletes all its sessions (cascade). The CLI
shows what will be deleted and asks for confirmation first: pass --force to
skip the prompt (required in non-interactive scripts) or --dry-run to preview
the delete without doing it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		ci, err := client.GetCodeInterpreter(ctx, args[0])
		if err != nil {
			return err
		}
		// Preview the cascade: count the sessions the delete removes (the
		// cheapest page, size 1, only reads the total). A failed count
		// degrades the preview but doesn't block the delete.
		sessions, sessionsErr := client.ListCodeInterpreterSessions(ctx, args[0], "", 1, 1)
		sessionCount := 0
		if sessionsErr == nil {
			sessionCount = sessions.TotalItem
		}
		printToolsDeletePreview("code interpreter", ci.ID, ci.Name, ci.Status, sessionCount, sessionsErr)
		if dryRun {
			toolsDryRunNotice()
			return nil
		}
		if !force && !confirmToolsDelete("Delete this code interpreter?") {
			toolsAbortedNotice()
			return nil
		}
		if err := client.DeleteCodeInterpreter(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Code interpreter %q deleted.\n", args[0])
		return output.PrintDeletedID(args[0])
	},
}

var toolsCodeInterpreterWaitCmd = &cobra.Command{
	Use:   "wait <id>",
	Short: "Wait for a code interpreter to become ACTIVE",
	Long: `Poll a code interpreter until it reaches a terminal state: ACTIVE
(success) or FAILED/DELETED/timeout (failure). Use after create — sessions
can only attach to an ACTIVE code interpreter.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		pctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if _, err := pollToolsResource(pctx, "code interpreter", args[0], interval,
			func(ctx context.Context) (string, error) {
				ci, err := client.GetCodeInterpreter(ctx, args[0])
				if err != nil {
					return "", err
				}
				return ci.Status, nil
			}); err != nil {
			return err
		}
		return output.PrintResource(struct{ ID string }{ID: args[0]}, func() string { return args[0] }, func() error {
			fmt.Fprintf(os.Stdout, "Code interpreter %q is ACTIVE.\n", args[0])
			return nil
		})
	},
}

var toolsCISessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage code interpreter sessions",
	Long: `Manage a code interpreter's sessions.

Sessions are the unit of execution: create one, then run
'code-interpreter session invoke' to execute tools inside it. Session
creation is synchronous (no wait needed).

--end-user-id is optional (the X-End-User-ID header): the end user on whose
behalf the session acts, 1-128 chars of A-Za-z0-9._:@- (validated
server-side).`,
}

var toolsCISessionListCmd = &cobra.Command{
	Use:   "list <code-interpreter-id>",
	Short: "List a code interpreter's sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		if err := validatePageParams(page, size); err != nil {
			return err
		}
		endUserID, err := flagEndUserID(cmd)
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.ListCodeInterpreterSessions(ctx, args[0], endUserID, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(result)
		case output.FormatID:
			if len(result.ListData) > 0 {
				output.PrintID(result.ListData[0].ID)
			}
			return nil
		}
		if len(result.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No code interpreter sessions found.")
			return nil
		}
		rows := make([][]string, 0, len(result.ListData))
		for i := range result.ListData {
			s := result.ListData[i]
			rows = append(rows, []string{
				s.ID, s.Name, s.Status, output.StrOrDash(s.EndUserID),
				output.FormatTime(s.CreatedAt),
			})
		}
		output.Table([]string{"ID", "Name", "Status", "End User", "Created"}, rows)
		printToolsPageFooter(result.Page, result.PageSize, result.TotalItem, result.TotalPage)
		return nil
	},
}

var toolsCISessionCreateCmd = &cobra.Command{
	Use:   "create <code-interpreter-id>",
	Short: "Create a code interpreter session",
	Long: `Create a session on a code interpreter.

The code interpreter must be ACTIVE (create is async — run
'code-interpreter wait' first); otherwise the API rejects the session with
409. Session creation itself is synchronous: the response is the ready
session.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name, _ := cmd.Flags().GetString("name")
		endUserID, err := flagEndUserID(cmd)
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		s, err := client.CreateCodeInterpreterSession(ctx, args[0], &toolspkg.CreateSessionRequest{Name: name}, endUserID)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Code interpreter session %q created (id %s, status %s).\n", s.Name, s.ID, s.Status)
		fmt.Fprintln(os.Stderr, "Run 'grn agentbase tools code-interpreter session invoke "+args[0]+" "+s.ID+" --tool ...' to execute a tool.")
		return output.PrintResource(s, func() string { return s.ID }, func() error { return renderToolsCISession(s) })
	},
}

var toolsCISessionDeleteCmd = &cobra.Command{
	Use:   "delete <code-interpreter-id> <session-id>",
	Short: "Delete a code interpreter session",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		endUserID, err := flagEndUserID(cmd)
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteCodeInterpreterSession(ctx, args[0], args[1], endUserID); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Code interpreter session %q deleted.\n", args[1])
		return output.PrintDeletedID(args[1])
	},
}

// toolsInvokeToolArgs maps the --tool value to its InvokeArguments, reading
// the per-tool flags and enforcing each tool's required/optional shape
// client-side (the API is the source of truth and rejects bad shapes with a
// 400 listing the accepted values — this catches the common typos early with
// a clearer message).
func toolsInvokeToolArgs(cmd *cobra.Command, tool string) (*toolspkg.InvokeArguments, error) {
	arguments := &toolspkg.InvokeArguments{}
	language, _ := cmd.Flags().GetString("language")
	code, _ := cmd.Flags().GetString("code")
	command, _ := cmd.Flags().GetString("command")
	directoryPath, _ := cmd.Flags().GetString("directory-path")
	content, _ := cmd.Flags().GetStringSlice("content")
	contentB64, _ := cmd.Flags().GetStringSlice("content-b64")
	paths, _ := cmd.Flags().GetStringSlice("path")
	taskID, _ := cmd.Flags().GetString("task-id")

	switch tool {
	case "executeCode":
		if language == "" || code == "" {
			return nil, fmt.Errorf("--tool executeCode requires --language and --code")
		}
		arguments.Language = language
		arguments.Code = code
	case "executeCommand", "startCommandExecution":
		if command == "" {
			return nil, fmt.Errorf("--tool %s requires --command", tool)
		}
		arguments.Command = command
	case "listFiles":
		arguments.DirectoryPath = directoryPath
	case "writeFiles":
		if len(content) == 0 && len(contentB64) == 0 {
			return nil, fmt.Errorf("--tool writeFiles requires at least one --content path=content (or --content-b64 path=<base64>)")
		}
		// --content carries raw file content: the API requires the blob
		// base64-encoded, so the CLI encodes it (otherwise every caller
		// would have to base64-encode by hand).
		files := make([]toolspkg.FileContent, 0, len(content)+len(contentB64))
		for _, item := range content {
			path, text, ok := strings.Cut(item, "=")
			if !ok || path == "" || text == "" {
				return nil, fmt.Errorf("invalid --content %q: must be path=content (split on the first '='); the CLI base64-encodes the content", item)
			}
			files = append(files, toolspkg.FileContent{Path: path, Blob: base64.StdEncoding.EncodeToString([]byte(text))})
		}
		// --content-b64 carries an already-encoded blob (e.g. piped binary
		// data): passed through as-is, validated so a typo fails here with
		// a clear message instead of a server 400.
		for _, item := range contentB64 {
			path, blob, ok := strings.Cut(item, "=")
			if !ok || path == "" || blob == "" {
				return nil, fmt.Errorf("invalid --content-b64 %q: must be path=<base64 blob> (split on the first '=')", item)
			}
			if _, err := base64.StdEncoding.DecodeString(blob); err != nil {
				return nil, fmt.Errorf("invalid --content-b64 %q: blob is not valid base64", item)
			}
			files = append(files, toolspkg.FileContent{Path: path, Blob: blob})
		}
		arguments.Content = files
	case "readFiles":
		if len(paths) == 0 {
			return nil, fmt.Errorf("--tool readFiles requires at least one --path")
		}
		arguments.Paths = paths
	case "getTask":
		if taskID == "" {
			return nil, fmt.Errorf("--tool getTask requires --task-id")
		}
		arguments.TaskID = taskID
	default:
		return nil, fmt.Errorf("unknown --tool %q: accepted values are %s", tool, strings.Join(toolspkg.InvokeTools, ", "))
	}
	return arguments, nil
}

var toolsCISessionInvokeCmd = &cobra.Command{
	Use:   "invoke <code-interpreter-id> <session-id>",
	Short: "Run a tool inside a code interpreter session",
	Long: `Run a built-in tool inside a code interpreter session and print its
result (the request is held open server-side until the command finishes).

Per-tool flags (the tool decides which are required):

  executeCode          --language, --code (both required)
  executeCommand       --command (required)
  startCommandExecution --command (required; async variant)
  listFiles            --directory-path (optional)
  writeFiles           --content path=content (repeatable, at least one;
                        the CLI base64-encodes each content) or
                        --content-b64 path=<base64 blob> (pre-encoded)
  readFiles            --path (repeatable, at least one)
  getTask              --task-id (required)

startCommandExecution returns a taskId (taskStatus submitted) and finishes
asynchronously: run 'code-interpreter session task-wait <code-interpreter-id>
<session-id> --task-id <id>' to wait for it (or poll with getTask).
readFiles returns file contents as base64 blobs in its content items;
writeFiles is the inverse — the CLI base64-encodes --content for you.

Output: table prints the result's content items (text as-is, non-text as
JSON) plus structuredContent; json prints the full response; id prints the
first text content item (script-friendly). An isError=true result prints and
exits non-zero.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		tool, _ := cmd.Flags().GetString("tool")
		arguments, err := toolsInvokeToolArgs(cmd, tool)
		if err != nil {
			return err
		}
		endUserID, err := flagEndUserID(cmd)
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.InvokeCodeInterpreter(ctx, args[0], args[1], &toolspkg.InvokeRequest{Name: tool, Arguments: arguments}, endUserID)
		if err != nil {
			return err
		}
		// startCommandExecution finishes asynchronously (taskStatus
		// submitted): hint the polling companion on stderr.
		if tool == "startCommandExecution" && (result.IsError == nil || !*result.IsError) {
			if taskID := toolsInvokeTaskID(result); taskID != "" {
				fmt.Fprintf(os.Stderr, "Run 'grn agentbase tools code-interpreter session task-wait %s %s --task-id %s' to wait for it.\n", args[0], args[1], taskID)
			}
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			if result.IsError != nil && *result.IsError {
				_ = output.JSON(result)
				return fmt.Errorf("tool %q failed (isError=true)", tool)
			}
			return output.JSON(result)
		case output.FormatID:
			if text, ok := toolsFirstTextItem(result); ok {
				fmt.Fprintln(os.Stdout, text)
			}
			if result.IsError != nil && *result.IsError {
				return fmt.Errorf("tool %q failed (isError=true)", tool)
			}
			return nil
		}
		if err := renderToolsInvokeResult(result); err != nil {
			return err
		}
		if result.IsError != nil && *result.IsError {
			return fmt.Errorf("tool %q failed (isError=true)", tool)
		}
		return nil
	},
}

// toolsFirstTextItem returns the first text content item of an invoke result,
// for -o id (script-friendly single-line output).
func toolsFirstTextItem(result *toolspkg.InvokeResponse) (string, bool) {
	for _, item := range result.Content {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok {
					return text, true
				}
			}
		}
	}
	return "", false
}

// renderToolsInvokeResult prints an invoke result: content items (text
// as-is, non-text as compact JSON), then structuredContent as indented JSON
// when present.
func renderToolsInvokeResult(result *toolspkg.InvokeResponse) error {
	if len(result.Content) == 0 && result.StructuredContent == nil {
		fmt.Fprintln(os.Stdout, "(no content)")
		return nil
	}
	for _, item := range result.Content {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok {
					fmt.Fprintln(os.Stdout, text)
					continue
				}
			}
		}
		encoded, err := json.Marshal(item)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(encoded))
	}
	if result.StructuredContent != nil {
		fmt.Fprintln(os.Stdout, "Structured content:")
		encoded, err := json.MarshalIndent(*result.StructuredContent, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(encoded))
	}
	return nil
}

// ---------------------------------------------------------------------------
// code-interpreter session task-wait
// ---------------------------------------------------------------------------

// toolsCISessionTaskWaitCmd waits for an async startCommandExecution task:
// polls getTask until the task finishes, then prints the final result — the
// polling companion 'session invoke' hints at after a submitted taskId.
var toolsCISessionTaskWaitCmd = &cobra.Command{
	Use:   "task-wait <code-interpreter-id> <session-id>",
	Short: "Wait for an async command task to finish",
	Long: `Poll a startCommandExecution task (via getTask) until it finishes.

'session invoke --tool startCommandExecution' runs a command asynchronously
and returns a taskId (taskStatus submitted). This command polls getTask until
taskStatus is completed, then prints the final result (content plus
structuredContent: taskStatus, exitCode, startedAt, finishedAt). It exits 0
only when the task completed with exit code 0; a failed task, a non-zero task
exit code, or a timeout exits non-zero.

Output mirrors 'invoke': table prints the result's content items plus
structuredContent; json prints the full getTask response; id prints the first
text content item.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		taskID, _ := cmd.Flags().GetString("task-id")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		endUserID, err := flagEndUserID(cmd)
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		pctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		result, err := pollToolsTask(pctx, client, args[0], args[1], taskID, interval, endUserID)
		if result != nil {
			// Render the last getTask result even on failure — it carries
			// the task's error output (same policy as 'invoke').
			switch output.GetFormat() {
			case output.FormatJSON:
				_ = output.JSON(result)
			case output.FormatID:
				if text, ok := toolsFirstTextItem(result); ok {
					fmt.Fprintln(os.Stdout, text)
				}
			default:
				_ = renderToolsInvokeResult(result)
			}
		}
		if err != nil {
			return err
		}
		// Translate the task's exit code into the CLI's exit status: 0 only
		// when the task completed with exit code 0.
		if code, ok := toolsTaskExitCode(result); ok && code != 0 {
			return fmt.Errorf("task %q completed with exit code %d", taskID, code)
		}
		return nil
	},
}

// pollToolsTask polls a startCommandExecution task via getTask until it
// reaches a terminal state. The observed lifecycle is submitted → completed;
// failed-style statuses are treated as terminal failures and anything else
// keeps polling until the context expires (the full status enum is not
// documented — unknown states are treated as in-flight). Progress lines go
// to stderr.
func pollToolsTask(ctx context.Context, client *toolspkg.Client, ciID, sessionID, taskID string, interval time.Duration, endUserID string) (*toolspkg.InvokeResponse, error) {
	for {
		result, err := client.InvokeCodeInterpreter(ctx, ciID, sessionID, &toolspkg.InvokeRequest{
			Name:      "getTask",
			Arguments: &toolspkg.InvokeArguments{TaskID: taskID},
		}, endUserID)
		if err != nil {
			return nil, err
		}
		if result.IsError != nil && *result.IsError {
			return result, fmt.Errorf("getTask for task %q failed (isError=true)", taskID)
		}
		switch status := toolsTaskStatus(result); status {
		case "completed":
			fmt.Fprintf(os.Stderr, "Task %q completed.\n", taskID)
			return result, nil
		case "failed", "error", "cancelled", "canceled":
			return result, fmt.Errorf("task %q reached status %q", taskID, status)
		default:
			if status == "" {
				status = "(no taskStatus)"
			}
			fmt.Fprintf(os.Stderr, "Task %q status: %s ...\n", taskID, status)
		}
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("timed out waiting for task %q (last status %q)", taskID, toolsTaskStatus(result))
		case <-time.After(interval):
		}
	}
}

// toolsTaskStatus reads structuredContent.taskStatus from an invoke result
// ("" when absent).
func toolsTaskStatus(result *toolspkg.InvokeResponse) string {
	if result.StructuredContent == nil {
		return ""
	}
	if v, ok := (*result.StructuredContent)["taskStatus"].(string); ok {
		return v
	}
	return ""
}

// toolsTaskExitCode reads structuredContent.exitCode from a completed task
// result. ok is false when the field is absent (treated as exit code 0).
func toolsTaskExitCode(result *toolspkg.InvokeResponse) (int, bool) {
	if result.StructuredContent == nil {
		return 0, false
	}
	v, ok := (*result.StructuredContent)["exitCode"].(float64)
	if !ok {
		return 0, false
	}
	return int(v), true
}

// toolsInvokeTaskID reads structuredContent.taskId from a
// startCommandExecution result ("" when absent) — used for the invoke hint.
func toolsInvokeTaskID(result *toolspkg.InvokeResponse) string {
	if result.StructuredContent == nil {
		return ""
	}
	if v, ok := (*result.StructuredContent)["taskId"].(string); ok {
		return v
	}
	return ""
}

// renderToolsCodeInterpreterDetail prints a code interpreter as a
// field/value table.
func renderToolsCodeInterpreterDetail(ci *toolspkg.CodeInterpreter) error {
	rows := [][]string{
		{"ID", ci.ID},
		{"Name", ci.Name},
		{"Description", output.StrOrDash(ci.Description)},
		{"Status", ci.Status},
		{"Created", output.FormatTime(ci.CreatedAt)},
		{"Updated", output.FormatTime(ci.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// renderToolsCISession prints a code-interpreter session as a field/value
// table.
func renderToolsCISession(s *toolspkg.Session) error {
	rows := [][]string{
		{"ID", s.ID},
		{"Name", s.Name},
		{"Status", s.Status},
		{"End User", output.StrOrDash(s.EndUserID)},
		{"Created", output.FormatTime(s.CreatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// code-interpreter group
	toolsCodeInterpreterListCmd.Flags().Int("page", 1, "Page number (1-based)")
	toolsCodeInterpreterListCmd.Flags().Int("size", 10, "Page size")
	toolsCodeInterpreterCmd.AddCommand(toolsCodeInterpreterListCmd)

	toolsCodeInterpreterCreateCmd.Flags().String("name", "", "Code interpreter name (required unless --interactive)")
	toolsCodeInterpreterCreateCmd.Flags().String("description", "", "Code interpreter description (optional)")
	toolsCodeInterpreterCmd.AddCommand(toolsCodeInterpreterCreateCmd)

	toolsCodeInterpreterCmd.AddCommand(toolsCodeInterpreterGetCmd)

	toolsCodeInterpreterDeleteCmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	toolsCodeInterpreterDeleteCmd.Flags().Bool("dry-run", false, "Preview what will be deleted without executing")
	toolsCodeInterpreterCmd.AddCommand(toolsCodeInterpreterDeleteCmd)

	toolsCodeInterpreterWaitCmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait")
	toolsCodeInterpreterWaitCmd.Flags().Duration("interval", 5*time.Second, "Poll interval")
	toolsCodeInterpreterCmd.AddCommand(toolsCodeInterpreterWaitCmd)

	toolsCmd.AddCommand(toolsCodeInterpreterCmd)

	// code-interpreter session group (nested under code-interpreter)
	toolsCISessionCmd.PersistentFlags().String("end-user-id", "", "Optional end user id (X-End-User-ID header, 1-128 chars of A-Za-z0-9._:@-)")

	toolsCISessionListCmd.Flags().Int("page", 1, "Page number (1-based)")
	toolsCISessionListCmd.Flags().Int("size", 10, "Page size")
	toolsCISessionCmd.AddCommand(toolsCISessionListCmd)

	toolsCISessionCreateCmd.Flags().String("name", "", "Session name (optional; server defaults it)")
	toolsCISessionCmd.AddCommand(toolsCISessionCreateCmd)

	toolsCISessionCmd.AddCommand(toolsCISessionDeleteCmd)

	toolsCISessionInvokeCmd.Flags().String("tool", "", "Tool to run (required): "+strings.Join(toolspkg.InvokeTools, ", "))
	_ = toolsCISessionInvokeCmd.MarkFlagRequired("tool")
	toolsCISessionInvokeCmd.Flags().String("language", "", "executeCode: language (required for executeCode)")
	toolsCISessionInvokeCmd.Flags().String("code", "", "executeCode: code to run (required for executeCode)")
	toolsCISessionInvokeCmd.Flags().String("command", "", "executeCommand/startCommandExecution: command to run (required)")
	toolsCISessionInvokeCmd.Flags().String("directory-path", "", "listFiles: directory to list (optional)")
	toolsCISessionInvokeCmd.Flags().StringSlice("content", nil, "writeFiles: file to write as path=content (repeatable, split on the first '='; the CLI base64-encodes the content)")
	toolsCISessionInvokeCmd.Flags().StringSlice("content-b64", nil, "writeFiles: file to write as path=<base64 blob> (repeatable, for pre-encoded content)")
	toolsCISessionInvokeCmd.Flags().StringSlice("path", nil, "readFiles: file path to read (repeatable)")
	toolsCISessionInvokeCmd.Flags().String("task-id", "", "getTask: task id (required)")
	toolsCISessionCmd.AddCommand(toolsCISessionInvokeCmd)

	toolsCISessionTaskWaitCmd.Flags().String("task-id", "", "Task id returned by startCommandExecution (required)")
	_ = toolsCISessionTaskWaitCmd.MarkFlagRequired("task-id")
	toolsCISessionTaskWaitCmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait")
	toolsCISessionTaskWaitCmd.Flags().Duration("interval", 5*time.Second, "Poll interval")
	toolsCISessionCmd.AddCommand(toolsCISessionTaskWaitCmd)

	toolsCodeInterpreterCmd.AddCommand(toolsCISessionCmd)
}
