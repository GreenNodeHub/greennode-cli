package agentbase

import (
	"context"
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

Deleting a code interpreter deletes its sessions too. There is no --force; the
delete is immediate.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
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
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.ListCodeInterpreterSessions(ctx, args[0], flagEndUserID(cmd), page, size)
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
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		s, err := client.CreateCodeInterpreterSession(ctx, args[0], &toolspkg.CreateSessionRequest{Name: name}, flagEndUserID(cmd))
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
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteCodeInterpreterSession(ctx, args[0], args[1], flagEndUserID(cmd)); err != nil {
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
		if len(content) == 0 {
			return nil, fmt.Errorf("--tool writeFiles requires at least one --content path=blob")
		}
		files := make([]toolspkg.FileContent, 0, len(content))
		for _, item := range content {
			path, blob, ok := strings.Cut(item, "=")
			if !ok || path == "" || blob == "" {
				return nil, fmt.Errorf("invalid --content %q: must be path=blob (split on the first '=')", item)
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
  writeFiles           --content path=blob (repeatable, at least one)
  readFiles            --path (repeatable, at least one)
  getTask              --task-id (required)

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
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.InvokeCodeInterpreter(ctx, args[0], args[1], &toolspkg.InvokeRequest{Name: tool, Arguments: arguments}, flagEndUserID(cmd))
		if err != nil {
			return err
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
	toolsCISessionInvokeCmd.Flags().StringSlice("content", nil, "writeFiles: file to write as path=blob (repeatable, split on the first '=')")
	toolsCISessionInvokeCmd.Flags().StringSlice("path", nil, "readFiles: file path to read (repeatable)")
	toolsCISessionInvokeCmd.Flags().String("task-id", "", "getTask: task id (required)")
	toolsCISessionCmd.AddCommand(toolsCISessionInvokeCmd)

	toolsCodeInterpreterCmd.AddCommand(toolsCISessionCmd)
}
