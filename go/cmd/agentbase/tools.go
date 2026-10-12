package agentbase

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	toolspkg "github.com/greennodehub/greennode-cli/internal/agentbase/tools"
)

// toolsCmd groups the built-in-tools commands: platform-managed headless
// browsers and code interpreters that agents invoke as tools. Both resources
// share one service (agent-core-built-in-tools, its own endpoint) and one
// shape: create is async (CREATING → ACTIVE|FAILED), sessions attach to the
// resource, and only the code interpreter has an invoke.
var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage built-in tools (browsers, code interpreters)",
	Long: `Manage the AgentBase built-in tools: platform-managed headless
browsers and code interpreters that agents invoke as tools.

Two groups:

  browser             — headless browsers: list/get/create/delete + wait,
                        plus sessions (CDP automation stream, live-view
                        stream, recording download).
  code-interpreter    — code-execution sandboxes: list/get/create/delete +
                        wait, plus sessions and ` + "`invoke`" + ` (run a tool:
                        executeCode, executeCommand, listFiles, ...).

Both resources create asynchronously (CREATING → ACTIVE|FAILED): run
` + "`wait`" + ` after create before attaching sessions (creating a session on a
non-ACTIVE resource fails with 409). The service is dev-only today; prod
endpoints are placeholders until it ships there.

tools commands share the ~/.greennode profile and the agent-core-built-in-tools
endpoint (see 'grn agentbase context current').`,
}

// newToolsClient builds the tools client for the agent-core-built-in-tools
// endpoint (same shared auth seam as runtime/marketplace).
func newToolsClient(ctx context.Context, cmd *cobra.Command) (*toolspkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return toolspkg.NewClient(ab.endpoints.BuiltInTools, provider), nil
}

// flagEndUserID reads the session groups' shared --end-user-id flag: the
// optional X-End-User-ID header value for session-scoped endpoints (1-128
// chars of A-Za-z0-9._:@-, validated server-side).
func flagEndUserID(cmd *cobra.Command) string {
	return flagString(cmd, "end-user-id")
}

// ---------------------------------------------------------------------------
// browser
// ---------------------------------------------------------------------------

// toolsBrowserCmd groups the headless-browser commands.
var toolsBrowserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Manage headless browsers",
	Long: `Manage platform-managed headless browsers.

Create is asynchronous: 201 CREATING → poll with 'wait' until ACTIVE (or
FAILED). Sessions attach to a browser (see 'browser session') and expose the
CDP automation stream, the live-view stream, and (after the session is
deleted) its recording.`,
}

var toolsBrowserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List browsers",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.ListBrowsers(ctx, page, size)
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
			fmt.Fprintln(os.Stderr, "No browsers found.")
			return nil
		}
		rows := make([][]string, 0, len(result.ListData))
		for i := range result.ListData {
			b := result.ListData[i]
			rows = append(rows, []string{
				b.ID, b.Name, output.StrOrDash(b.Description), b.Status,
				output.FormatTime(b.CreatedAt),
			})
		}
		output.Table([]string{"ID", "Name", "Description", "Status", "Created"}, rows)
		printToolsPageFooter(result.Page, result.PageSize, result.TotalItem, result.TotalPage)
		return nil
	},
}

var toolsBrowserCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a browser",
	Long: `Create a headless browser.

Creation is asynchronous: the API returns the browser in CREATING; run
'grn agentbase tools browser wait <id>' until it is ACTIVE before attaching
sessions (creating a session on a non-ACTIVE browser fails with 409).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Browser name")
		if err != nil {
			return err
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		b, err := client.CreateBrowser(ctx, &toolspkg.CreateBrowserRequest{Name: name, Description: description})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Browser %q created (id %s, status %s).\n", b.Name, b.ID, b.Status)
		fmt.Fprintln(os.Stderr, "Run 'grn agentbase tools browser wait "+b.ID+"' to wait for ACTIVE.")
		return output.PrintResource(b, func() string { return b.ID }, func() error { return renderToolsBrowserDetail(b) })
	},
}

var toolsBrowserGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		b, err := client.GetBrowser(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(b, func() string { return b.ID }, func() error { return renderToolsBrowserDetail(b) })
	},
}

var toolsBrowserDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a browser",
	Long: `Delete a headless browser.

Deleting a browser deletes its sessions too. There is no --force; the delete
is immediate.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteBrowser(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Browser %q deleted.\n", args[0])
		return output.PrintDeletedID(args[0])
	},
}

var toolsBrowserWaitCmd = &cobra.Command{
	Use:   "wait <id>",
	Short: "Wait for a browser to become ACTIVE",
	Long: `Poll a browser until it reaches a terminal state: ACTIVE (success) or
FAILED/DELETED/timeout (failure). Use after create — sessions can only attach
to an ACTIVE browser.`,
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
		if _, err := pollToolsResource(pctx, "browser", args[0], interval,
			func(ctx context.Context) (string, error) {
				b, err := client.GetBrowser(ctx, args[0])
				if err != nil {
					return "", err
				}
				return b.Status, nil
			}); err != nil {
			return err
		}
		return output.PrintResource(struct{ ID string }{ID: args[0]}, func() string { return args[0] }, func() error {
			fmt.Fprintf(os.Stdout, "Browser %q is ACTIVE.\n", args[0])
			return nil
		})
	},
}

// pollToolsResource polls a resource's status (via get) until ACTIVE (success)
// or FAILED/DELETED/ctx-expiry (failure). Shared by browser wait and
// code-interpreter wait — both resources share the CREATING→ACTIVE|FAILED
// lifecycle. Progress lines go to stderr.
func pollToolsResource(ctx context.Context, what, id string, interval time.Duration, get func(ctx context.Context) (string, error)) (string, error) {
	var last string
	for {
		status, err := get(ctx)
		if err != nil {
			return last, err
		}
		last = status
		switch status {
		case "ACTIVE":
			fmt.Fprintf(os.Stderr, "%s %q reached state %s.\n", strings.ToUpper(what[:1])+what[1:], id, status)
			return status, nil
		case "FAILED", "DELETED":
			return status, fmt.Errorf("%s %q reached state %s", what, id, status)
		}
		fmt.Fprintf(os.Stderr, "%s %q state: %s ...\n", strings.ToUpper(what[:1])+what[1:], id, status)
		select {
		case <-ctx.Done():
			return last, fmt.Errorf("timed out waiting for %s %q (last state %s)", what, id, last)
		case <-time.After(interval):
		}
	}
}

// renderToolsBrowserDetail prints a browser as a field/value table.
func renderToolsBrowserDetail(b *toolspkg.Browser) error {
	rows := [][]string{
		{"ID", b.ID},
		{"Name", b.Name},
		{"Description", output.StrOrDash(b.Description)},
		{"Status", b.Status},
		{"Created", output.FormatTime(b.CreatedAt)},
		{"Updated", output.FormatTime(b.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// printToolsPageFooter prints the shared listData paging footer.
func printToolsPageFooter(page, pageSize, totalItem, totalPage int) {
	fmt.Fprintf(os.Stdout, "Page %d of %d | %d per page | %d total\n", page, totalPage, pageSize, totalItem)
}

// ---------------------------------------------------------------------------
// browser session
// ---------------------------------------------------------------------------

// toolsBrowserSessionCmd groups a browser's session commands. The shared
// --end-user-id flag is the optional X-End-User-ID header sent on the
// session-scoped endpoints (empty = omitted).
var toolsBrowserSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage browser sessions",
	Long: `Manage a browser's sessions.

Sessions run inside a browser and expose the CDP automation stream (the
endpoint a browser client such as browser-mcp connects to), the live-view
stream (screenshots; signed URL via 'live-view-stream-sign'), and the session
recording (downloadable only after the session is deleted and archived).

--end-user-id is optional (the X-End-User-ID header): the end user on whose
behalf the session acts, 1-128 chars of A-Za-z0-9._:@- (validated
server-side).`,
}

var toolsBrowserSessionListCmd = &cobra.Command{
	Use:   "list <browser-id>",
	Short: "List a browser's sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.ListBrowserSessions(ctx, args[0], flagEndUserID(cmd), page, size)
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
			fmt.Fprintln(os.Stderr, "No browser sessions found.")
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

var toolsBrowserSessionCreateCmd = &cobra.Command{
	Use:   "create <browser-id>",
	Short: "Create a browser session",
	Long: `Create a session on a browser.

The browser must be ACTIVE (create is async — run 'browser wait' first);
otherwise the API rejects the session with 409. Session creation itself is
synchronous: the response is the ready session. The session's streams appear
in 'browser session get'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name, _ := cmd.Flags().GetString("name")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		s, err := client.CreateBrowserSession(ctx, args[0], &toolspkg.CreateBrowserSessionRequest{Name: name}, flagEndUserID(cmd))
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Browser session %q created (id %s, status %s).\n", s.Name, s.ID, s.Status)
		fmt.Fprintln(os.Stderr, "Run 'grn agentbase tools browser session get "+args[0]+" "+s.ID+"' to see its streams.")
		return output.PrintResource(s, func() string { return s.ID }, func() error { return renderToolsBrowserSession(s) })
	},
}

var toolsBrowserSessionGetCmd = &cobra.Command{
	Use:   "get <browser-id> <session-id>",
	Short: "Show a browser session and its streams",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		d, err := client.GetBrowserSession(ctx, args[0], args[1], flagEndUserID(cmd))
		if err != nil {
			return err
		}
		return output.PrintResource(d, func() string { return d.ID }, func() error { return renderToolsBrowserSessionDetail(d) })
	},
}

var toolsBrowserSessionDeleteCmd = &cobra.Command{
	Use:   "delete <browser-id> <session-id>",
	Short: "Delete a browser session",
	Long: `Delete a browser session.

The session recording only becomes downloadable after the session is deleted
(and archived server-side): run 'browser session recording <browser-id>
<session-id>' afterwards.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteBrowserSession(ctx, args[0], args[1], flagEndUserID(cmd)); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Browser session %q deleted.\n", args[1])
		return output.PrintDeletedID(args[1])
	},
}

var toolsBrowserSessionAutomationStreamCmd = &cobra.Command{
	Use:   "automation-stream <browser-id> <session-id>",
	Short: "Print a session's CDP automation stream URL",
	Long: `Print the session's CDP automation stream URL.

This constructs the URL the same way a browser client (browser-mcp) does —
<v1-base>/v1/browsers/<browser-id>/sessions/<session-id>/automation-stream —
from the CLI's resolved endpoint. No HTTP call is made (relay streams are
consumed by a CDP client, not the CLI); --end-user-id is ignored here.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ab := mustLoadAgentbaseCtx(cmd)
		url := strings.TrimRight(ab.endpoints.BuiltInTools, "/") +
			fmt.Sprintf("/v1/browsers/%s/sessions/%s/automation-stream", args[0], args[1])
		return output.PrintResource(map[string]string{"url": url}, func() string { return url }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{{"Automation Stream URL", url}})
			return nil
		})
	},
}

var toolsBrowserSessionLiveViewStreamSignCmd = &cobra.Command{
	Use:   "live-view-stream-sign <browser-id> <session-id>",
	Short: "Sign a session's live-view stream URL",
	Long: `Request a short-lived signed URL for the session's live-view stream.

The live-view stream (screenshots) requires a signed URL; this returns one
plus its expiry in seconds. Pass --end-user-id to sign on behalf of an end
user.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		sig, err := client.SignLiveViewStream(ctx, args[0], args[1], flagEndUserID(cmd))
		if err != nil {
			return err
		}
		return output.PrintResource(sig, func() string { return sig.URL }, func() error {
			output.Table([]string{"Field", "Value"}, [][]string{
				{"Signed URL", sig.URL},
				{"Expires In", fmt.Sprintf("%ds", sig.ExpiresIn)},
			})
			return nil
		})
	},
}

var toolsBrowserSessionRecordingCmd = &cobra.Command{
	Use:   "recording <browser-id> <session-id>",
	Short: "Download a session's recording",
	Long: `Download the session's recording as NDJSON (one JSON event per line,
verbatim from the API).

The recording only exists after the session is deleted and archived
server-side — until then the API returns 404. Without --output-file the raw
NDJSON is written to stdout (pipe-friendly, regardless of -o); with
--output-file it is written to the file and, under -o id, the path is
printed.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		outputFile, _ := cmd.Flags().GetString("output-file")
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		raw, err := client.DownloadRecording(ctx, args[0], args[1], flagEndUserID(cmd))
		if err != nil {
			return err
		}
		if outputFile == "" {
			_, err = os.Stdout.Write(raw)
			return err
		}
		if err := os.WriteFile(outputFile, raw, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Recording written to %s (%d bytes).\n", outputFile, len(raw))
		if output.GetFormat() == output.FormatID {
			output.PrintID(outputFile)
		}
		return nil
	},
}

// toolsBrowserSessionStreamsCmd groups the session's stream toggles.
var toolsBrowserSessionStreamsCmd = &cobra.Command{
	Use:   "streams",
	Short: "Manage a browser session's streams",
	Long: `Manage a browser session's streams.

Only the automation stream's status is toggleable (ENABLED|DISABLED); the
live-view stream is always on. The update returns no body, so the CLI re-reads
the session and shows the resulting streams.`,
}

var toolsBrowserSessionStreamsUpdateCmd = &cobra.Command{
	Use:   "update <browser-id> <session-id>",
	Short: "Toggle a session's automation stream",
	Long: `Toggle the session's automation stream status (ENABLED or DISABLED).

The API returns no body, so the CLI re-reads the session and renders the
resulting streams. The live-view stream is always on and cannot be toggled.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		status, _ := cmd.Flags().GetString("status")
		if status != "ENABLED" && status != "DISABLED" {
			return fmt.Errorf("invalid --status %q: must be ENABLED or DISABLED", status)
		}
		client, err := newToolsClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.UpdateBrowserStreams(ctx, args[0], args[1], status, flagEndUserID(cmd)); err != nil {
			return err
		}
		d, err := client.GetBrowserSession(ctx, args[0], args[1], flagEndUserID(cmd))
		if err != nil {
			return err
		}
		return output.PrintResource(d, func() string { return d.ID }, func() error { return renderToolsBrowserSessionDetail(d) })
	},
}

// renderToolsBrowserSession prints a bare browser session (no streams) as a
// field/value table.
func renderToolsBrowserSession(s *toolspkg.BrowserSession) error {
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

// renderToolsBrowserSessionDetail prints a browser session detail: the
// session fields plus its stream endpoints (when the API returns them).
func renderToolsBrowserSessionDetail(d *toolspkg.BrowserSessionDetail) error {
	if err := renderToolsBrowserSession(&d.BrowserSession); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "Streams:")
	rows := [][]string{{"Automation Stream", "-"}, {"Automation Status", "-"}, {"Live View Stream", "-"}}
	if d.Streams != nil {
		if d.Streams.AutomationStream != nil {
			rows[0][1] = d.Streams.AutomationStream.StreamEndpoint
			rows[1][1] = d.Streams.AutomationStream.StreamStatus
		}
		if d.Streams.LiveViewStream != nil {
			rows[2][1] = d.Streams.LiveViewStream.StreamEndpoint
		}
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(toolsCmd)

	// browser group
	toolsBrowserListCmd.Flags().Int("page", 1, "Page number (1-based)")
	toolsBrowserListCmd.Flags().Int("size", 10, "Page size")
	toolsBrowserCmd.AddCommand(toolsBrowserListCmd)

	toolsBrowserCreateCmd.Flags().String("name", "", "Browser name (required unless --interactive)")
	toolsBrowserCreateCmd.Flags().String("description", "", "Browser description (optional)")
	toolsBrowserCmd.AddCommand(toolsBrowserCreateCmd)

	toolsBrowserCmd.AddCommand(toolsBrowserGetCmd)
	toolsBrowserCmd.AddCommand(toolsBrowserDeleteCmd)

	toolsBrowserWaitCmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait")
	toolsBrowserWaitCmd.Flags().Duration("interval", 5*time.Second, "Poll interval")
	toolsBrowserCmd.AddCommand(toolsBrowserWaitCmd)

	toolsCmd.AddCommand(toolsBrowserCmd)

	// browser session group (nested under browser)
	toolsBrowserSessionCmd.PersistentFlags().String("end-user-id", "", "Optional end user id (X-End-User-ID header, 1-128 chars of A-Za-z0-9._:@-)")

	toolsBrowserSessionListCmd.Flags().Int("page", 1, "Page number (1-based)")
	toolsBrowserSessionListCmd.Flags().Int("size", 10, "Page size")
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionListCmd)

	toolsBrowserSessionCreateCmd.Flags().String("name", "", "Session name (optional; server defaults it)")
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionCreateCmd)

	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionGetCmd)
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionDeleteCmd)
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionAutomationStreamCmd)
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionLiveViewStreamSignCmd)

	toolsBrowserSessionRecordingCmd.Flags().String("output-file", "", "Write the recording to this file instead of stdout (NDJSON)")
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionRecordingCmd)

	toolsBrowserSessionStreamsUpdateCmd.Flags().String("status", "", "Automation stream status: ENABLED or DISABLED (required)")
	_ = toolsBrowserSessionStreamsUpdateCmd.MarkFlagRequired("status")
	toolsBrowserSessionStreamsCmd.AddCommand(toolsBrowserSessionStreamsUpdateCmd)
	toolsBrowserSessionCmd.AddCommand(toolsBrowserSessionStreamsCmd)

	toolsBrowserCmd.AddCommand(toolsBrowserSessionCmd)
}
