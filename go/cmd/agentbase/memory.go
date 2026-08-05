package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	memorypkg "github.com/greennodehub/greennode-cli/internal/agentbase/memory"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
)

// memoryCmd groups the agent-memory commands. The agentbase /memory endpoint
// fronts the agent-core-memory REST API (POST/GET/DELETE /memories +
// /memories/{id}/memory-records:search). A memory is a container for an agent's
// long-term facts (stored in an external Mem0 vector store) plus short-term
// events. Resources are synchronous (no async FSM), so there is no `wait`.
var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage agent memories",
	Long: `Create and manage agent memories (the agent-core-memory service).

A memory is a container holding an agent's long-term facts (memory records,
backed by an external Mem0 vector store) and short-term conversation events.
Memories are created synchronously (no WAITING_* state) and soft-deleted
(ACTIVE → DELETED).

The signature command is 'search' — semantic search over a memory's long-term
facts:

    grn agentbase memory create --file mem.yaml
    grn agentbase memory search <id> --namespace /strategies/SEMANTIC/actors/alice --query "dark mode"

Creation requires at least one long-term-memory strategy (name/type/namespace
template); for anything beyond the simple single-strategy path, generate a
template, fill it in, and apply with --file:

    grn agentbase memory generate > mem.yaml

Memories share the ~/.greennode profile like the rest of agentbase.`,
}

// newMemoryClient mirrors newRuntimeClient: resolve the shared profile + env,
// select the shared token provider, force-mint once so auth failures surface
// before the first call, and point the typed client at the memory endpoint.
func newMemoryClient(ctx context.Context, cmd *cobra.Command) (*memorypkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return memorypkg.NewClient(ab.endpoints.Memory, provider), nil
}

// ---------------------------------------------------------------------------
// create
// ---------------------------------------------------------------------------

// memoryFile is shared by create (--file); only one command runs at a time.
var memoryFile string

var memoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent memory",
	Long: `Create a new agent memory.

Creation requires at least one long-term-memory strategy (each with name, type,
and a namespace template). The simple flag path builds a single strategy; for
multiple strategies or a custom fact-extraction prompt, use --file with a
template produced by 'grn agentbase memory generate'.

The memory is created synchronously and is immediately ACTIVE (no wait needed).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if memoryFile != "" {
			data, err := os.ReadFile(memoryFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err := loadMemorySpec(data)
			if err != nil {
				return err
			}
			return createMemoryAndPrint(ctx, cmd, req)
		}
		f := cmd.Flags()

		name, _ := f.GetString("name")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Memory name (letters, numbers, . _ -)")
		if err != nil {
			return err
		}

		description, _ := f.GetString("description")
		description, err = cliinput.RequireOrPromptString(description, "--description", "Description")
		if err != nil {
			return err
		}

		eventExpiry, _ := f.GetInt("event-expiry-duration")

		strategyName, _ := f.GetString("strategy-name")
		strategyName, err = cliinput.RequireOrPromptString(strategyName, "--strategy-name", "Strategy name")
		if err != nil {
			return err
		}
		strategyType, _ := f.GetString("strategy-type")
		strategyType, err = cliinput.RequireOrPromptString(strategyType, "--strategy-type", "Strategy type (USER_PREFERENCE|SEMANTIC|CUSTOM|...)")
		if err != nil {
			return err
		}
		strategyNS, _ := f.GetString("strategy-namespace")
		strategyNS, err = cliinput.RequireOrPromptString(strategyNS, "--strategy-namespace", "Namespace template (e.g. /strategies/SEMANTIC/actors/{actorId})")
		if err != nil {
			return err
		}

		strategy := memorypkg.LongTermMemoryStrategy{
			Name:              strategyName,
			Type:              strings.ToUpper(strategyType),
			NamespaceTemplate: strategyNS,
		}
		if v, _ := f.GetString("strategy-prompt"); v != "" {
			strategy.CustomFactExtractionPrompt = v
		}
		if f.Changed("strategy-auto-generate") {
			strategy.EnableAutomaticMemoryRecordGeneration, _ = f.GetBool("strategy-auto-generate")
		}

		return createMemoryAndPrint(ctx, cmd, &memorypkg.CreateMemoryRequest{
			Name:                     name,
			Description:              description,
			EventExpiryDuration:      eventExpiry,
			LongTermMemoryStrategies: []memorypkg.LongTermMemoryStrategy{strategy},
		})
	},
}

func createMemoryAndPrint(ctx context.Context, cmd *cobra.Command, req *memorypkg.CreateMemoryRequest) error {
	client, err := newMemoryClient(ctx, cmd)
	if err != nil {
		return err
	}
	mem, err := client.Create(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Memory %q created (id %s, state %s).\n", mem.Name, mem.ID, mem.Status)
	return output.PrintResource(mem, func() string { return mem.ID }, func() error { return renderMemoryDetail(mem) })
}

// ---------------------------------------------------------------------------
// generate — print a commented create-template (kubectl-style)
// ---------------------------------------------------------------------------

var memoryGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print a memory create template (YAML or JSON)",
	Long: `Print a commented memory create template to stdout. Save it, fill it in, and
apply with 'grn agentbase memory create --file <file>'.

Defaults to YAML (with comments); pass -o json for a JSON skeleton.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			example := &memorypkg.CreateMemoryRequest{
				Name:                "my-memory",
				Description:         "",
				EventExpiryDuration: 30,
				LongTermMemoryStrategies: []memorypkg.LongTermMemoryStrategy{{
					Name:              "semantic",
					Type:              "SEMANTIC",
					NamespaceTemplate: "/strategies/SEMANTIC/actors/{actorId}",
				}},
			}
			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(memoryCreateTemplateYAML)
		return nil
	},
}

// memoryCreateTemplateYAML is a hand-written, commented skeleton of
// CreateMemoryRequest. Keys are the JSON (camelCase) field names so the file
// round-trips through 'create --file' exactly.
const memoryCreateTemplateYAML = `# Agent memory create spec.
# Fill in and apply with:  grn agentbase memory create --file <this-file>
#
# Required: name, description, longTermMemoryStrategies (each with name, type,
# namespaceTemplate). name regex: ^[a-zA-Z0-9._-]*$ (max 50).
name: my-memory
description: ""
eventExpiryDuration: 30     # days, 1-365 (short-term event TTL)

# At least one long-term-memory strategy. type is a built-in strategy key
# (USER_PREFERENCE, SEMANTIC, CUSTOM, ...). namespaceTemplate resolves to the
# namespace memory-records are stored/searched under; it is REQUIRED for
# 'memory search --namespace'. {actorId} is substituted per actor at runtime.
longTermMemoryStrategies:
  - name: semantic
    type: SEMANTIC                       # USER_PREFERENCE | SEMANTIC | CUSTOM | ...
    namespaceTemplate: /strategies/SEMANTIC/actors/{actorId}
    # customFactExtractionPrompt: ""     # optional, max 1000 chars
    enableAutomaticMemoryRecordGeneration: false
  # - name: preferences
  #   type: USER_PREFERENCE
  #   namespaceTemplate: /strategies/USER_PREFERENCE/actors/{actorId}
`

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List memories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.List(ctx, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			output.JSON(resp)
			return nil
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(resp.ListData[0].ID)
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No memories found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			m := resp.ListData[i]
			rows = append(rows, []string{
				m.ID, m.Name, m.Status, fmt.Sprintf("%dd", m.EventExpiryDuration), formatTimeVal(m.CreatedAt),
			})
		}
		output.Table([]string{"ID", "Name", "State", "Event TTL", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

var memoryGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		mem, err := client.Get(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(mem, func() string { return mem.ID }, func() error { return renderMemoryDetail(mem) })
	},
}

// ---------------------------------------------------------------------------
// delete (soft: ACTIVE → DELETED)
// ---------------------------------------------------------------------------

var memoryDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.Delete(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Memory %q deleted.\n", args[0])
		output.PrintDeletedID(args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// search — semantic search over a memory's long-term records
// ---------------------------------------------------------------------------

var memorySearchCmd = &cobra.Command{
	Use:   "search <id>",
	Short: "Semantic-search a memory's long-term facts",
	Long: `Run a semantic search over a memory's long-term memory records (backed by the
external Mem0 vector store). Returns ranked facts with relevance scores.

namespace is the resolved namespace string the records live under (from the
memory's strategy namespaceTemplate, e.g. /strategies/SEMANTIC/actors/<actor>).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		namespace, _ := f.GetString("namespace")
		if namespace == "" {
			return fmt.Errorf("required flag %q not set", "namespace")
		}
		query, _ := f.GetString("query")
		if query == "" {
			return fmt.Errorf("required flag %q not set", "query")
		}
		limit, _ := f.GetInt("limit")
		threshold, _ := f.GetFloat64("threshold")

		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		records, err := client.Search(ctx, args[0], namespace, &memorypkg.SearchMemoryRecordsRequest{
			Query:          query,
			Limit:          limit,
			ScoreThreshold: threshold,
		})
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			output.JSON(records)
			return nil
		case output.FormatID:
			if len(records) > 0 {
				output.PrintID(records[0].ID)
			}
			return nil
		}
		return renderMemoryRecords(records, namespace, query)
	},
}

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderMemoryDetail(mem *memorypkg.Memory) error {
	rows := [][]string{
		{"ID", mem.ID},
		{"Name", mem.Name},
		{"Description", output.StrOrDash(mem.Description)},
		{"Event TTL", fmt.Sprintf("%d days", mem.EventExpiryDuration)},
		{"State", mem.Status},
		{"Created", formatTimeVal(mem.CreatedAt)},
		{"Updated", formatTimeVal(mem.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// renderMemoryRecords renders ranked search results. Sorted by the server
// (Mem0 ranks by relevance); we just display score + the fact text.
func renderMemoryRecords(records []memorypkg.MemoryRecord, namespace, query string) error {
	fmt.Fprintf(os.Stderr, "namespace=%s query=%q → %d result(s)\n", namespace, query, len(records))
	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "No matching memory records.")
		return nil
	}
	rows := make([][]string, 0, len(records))
	for i := range records {
		r := records[i]
		rows = append(rows, []string{scoreStr(r.Score), r.Memory, formatTimeVal(r.UpdatedAt)})
	}
	output.Table([]string{"Score", "Memory", "Updated"}, rows)
	return nil
}

// scoreStr formats a relevance score, or "-" when absent (plain list, not search).
func scoreStr(s *float64) string {
	if s == nil {
		return "-"
	}
	return fmt.Sprintf("%.3f", *s)
}

// ---------------------------------------------------------------------------
// file parsing (YAML or JSON -> struct)
// ---------------------------------------------------------------------------

// loadMemorySpec parses a YAML/JSON create spec into CreateMemoryRequest.
// yaml.Unmarshal into a map then json.Unmarshal into the struct so the file's
// camelCase keys bind to the struct's json tags (yaml.v3 does not honor json
// tags directly).
func loadMemorySpec(data []byte) (*memorypkg.CreateMemoryRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req memorypkg.CreateMemoryRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid memory spec: %w", err)
	}
	if req.Name == "" {
		return nil, fmt.Errorf("spec is missing required field: name")
	}
	if len(req.LongTermMemoryStrategies) == 0 {
		return nil, fmt.Errorf("spec is missing required field: longTermMemoryStrategies (at least one, each with name/type/namespaceTemplate)")
	}
	for i, s := range req.LongTermMemoryStrategies {
		if s.Name == "" || s.Type == "" || s.NamespaceTemplate == "" {
			return nil, fmt.Errorf("longTermMemoryStrategies[%d] is missing required field(s): name, type, namespaceTemplate", i)
		}
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// sub-resources (Slice 3): actor / session / event / strategy / record
// ---------------------------------------------------------------------------

var memoryActorCmd = &cobra.Command{Use: "actor", Short: "Browse a memory's actors"}
var memorySessionCmd = &cobra.Command{Use: "session", Short: "Browse a memory's sessions"}
var memoryEventCmd = &cobra.Command{Use: "event", Short: "Manage a session's events"}
var memoryStrategyCmd = &cobra.Command{Use: "strategy", Short: "Browse a memory's long-term-memory strategies"}
var memoryRecordCmd = &cobra.Command{Use: "record", Short: "Manage a memory's long-term records"}

// --- actor list ---

var memoryActorListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List actors in a memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListActors(ctx, args[0], page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(resp.ListData[0].ActorID)
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No actors found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			a := resp.ListData[i]
			rows = append(rows, []string{a.ActorID, a.Status})
		}
		output.Table([]string{"Actor ID", "Status"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// --- session list ---

var memorySessionListCmd = &cobra.Command{
	Use:   "list <id> <actor-id>",
	Short: "List an actor's sessions in a memory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListSessions(ctx, args[0], args[1], page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(resp.ListData[0].SessionID)
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No sessions found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			s := resp.ListData[i]
			rows = append(rows, []string{s.SessionID, s.Status})
		}
		output.Table([]string{"Session ID", "Status"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// --- event list / create / delete ---

var memoryEventListCmd = &cobra.Command{
	Use:   "list <id> <actor-id> <session-id>",
	Short: "List events in a session",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		events, err := client.ListSessionEvents(ctx, args[0], args[1], args[2], from, to, page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(events)
		case output.FormatID:
			return nil
		}
		return renderSessionEvents(events)
	},
}

var memoryEventCreateCmd = &cobra.Command{
	Use:   "create <id> <actor-id> <session-id>",
	Short: "Append an event to a session",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		eventType, _ := f.GetString("type")
		if eventType == "" {
			return fmt.Errorf("required flag %q not set", "type")
		}
		role, _ := f.GetString("role")
		message, _ := f.GetString("message")
		binaryData, _ := f.GetString("binary-data")
		req := &memorypkg.EventCreateRequest{
			Payload: memorypkg.EventPayload{
				Type:       eventType,
				Role:       role,
				Message:    message,
				BinaryData: binaryData,
			},
		}
		if v, _ := f.GetString("event-timestamp"); v != "" {
			req.EventTimestamp = v
		}
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.CreateSessionEvent(ctx, args[0], args[1], args[2], req); err != nil {
			return err
		}
		output.Successf("Event appended to session %s.", args[2])
		return nil
	},
}

var memoryEventDeleteCmd = &cobra.Command{
	Use:   "delete <id> <actor-id> <session-id> <event-id>",
	Short: "Delete a session event",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteSessionEvent(ctx, args[0], args[1], args[2], args[3]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Event %q deleted.\n", args[3])
		output.PrintDeletedID(args[3])
		return nil
	},
}

// --- strategy list ---

var memoryStrategyListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List a memory's long-term-memory strategies",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		strategies, err := client.ListStrategies(ctx, args[0])
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(strategies)
		case output.FormatID:
			if len(strategies) > 0 {
				output.PrintID(strategies[0].ID)
			}
			return nil
		}
		if len(strategies) == 0 {
			fmt.Fprintln(os.Stderr, "No strategies found.")
			return nil
		}
		rows := make([][]string, 0, len(strategies))
		for i := range strategies {
			s := strategies[i]
			rows = append(rows, []string{s.ID, s.Name, s.Type, s.NamespaceTemplate, s.Status})
		}
		output.Table([]string{"ID", "Name", "Type", "Namespace Template", "Status"}, rows)
		return nil
	},
}

// --- record list / delete / insert / generate-from-session / generate-from-content ---

var memoryRecordListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List a memory's long-term records (under a namespace)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			return fmt.Errorf("required flag %q not set", "namespace")
		}
		limit, _ := cmd.Flags().GetInt("limit")
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		records, err := client.ListRecords(ctx, args[0], namespace, limit)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(records)
		case output.FormatID:
			if len(records) > 0 {
				output.PrintID(records[0].ID)
			}
			return nil
		}
		fmt.Fprintf(os.Stderr, "namespace=%s → %d record(s)\n", namespace, len(records))
		if len(records) == 0 {
			fmt.Fprintln(os.Stderr, "No memory records found.")
			return nil
		}
		rows := make([][]string, 0, len(records))
		for i := range records {
			r := records[i]
			rows = append(rows, []string{r.ID, r.Memory, formatTimeVal(r.UpdatedAt)})
		}
		output.Table([]string{"ID", "Memory", "Updated"}, rows)
		return nil
	},
}

var memoryRecordDeleteCmd = &cobra.Command{
	Use:   "delete <id> <record-id>",
	Short: "Delete a memory record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.DeleteRecord(ctx, args[0], args[1]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Memory record %q deleted.\n", args[1])
		output.PrintDeletedID(args[1])
		return nil
	},
}

// memoryInsertRecords holds the repeatable --record values for 'record insert'.
var memoryInsertRecords []string

var memoryRecordInsertCmd = &cobra.Command{
	Use:   "insert <id>",
	Short: "Insert long-term records directly (skip extraction)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		namespace, _ := cmd.Flags().GetString("namespace")
		if namespace == "" {
			return fmt.Errorf("required flag %q not set", "namespace")
		}
		if len(memoryInsertRecords) == 0 {
			return fmt.Errorf("required flag %q not set", "record")
		}
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.InsertRecords(ctx, args[0], namespace, &memorypkg.MemoryRecordInsertDirectlyRequest{
			MemoryRecords: memoryInsertRecords,
		}); err != nil {
			return err
		}
		output.Successf("Inserted %d record(s) into namespace %s.", len(memoryInsertRecords), namespace)
		return nil
	},
}

var memoryRecordGenerateFromSessionCmd = &cobra.Command{
	Use:   "generate-from-session <id>",
	Short: "Generate long-term records from a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		actorID, _ := f.GetString("actor-id")
		sessionID, _ := f.GetString("session-id")
		strategyID, _ := f.GetString("strategy-id")
		if actorID == "" || sessionID == "" || strategyID == "" {
			return fmt.Errorf("required flags not set: --actor-id, --session-id, --strategy-id")
		}
		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.GenerateRecordsFromSession(ctx, args[0], actorID, sessionID, strategyID); err != nil {
			return err
		}
		output.Successf("Record generation from session %s queued.", sessionID)
		return nil
	},
}

// memoryGenerateFile / memoryGenerateMessages hold the --file / --message values
// for 'record generate-from-content'.
var memoryGenerateFile string
var memoryGenerateMessages []string

var memoryRecordGenerateFromContentCmd = &cobra.Command{
	Use:   "generate-from-content <id>",
	Short: "Generate long-term records from chat content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		strategyID, _ := f.GetString("strategy-id")
		if strategyID == "" {
			return fmt.Errorf("required flag %q not set", "strategy-id")
		}
		actorID, _ := f.GetString("actor-id")
		sessionID, _ := f.GetString("session-id")

		var req *memorypkg.MemoryRecordGenerateFromContentRequest
		if memoryGenerateFile != "" {
			data, err := os.ReadFile(memoryGenerateFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err = loadGenerateFromContentSpec(data)
			if err != nil {
				return err
			}
		} else if len(memoryGenerateMessages) > 0 {
			msgs := make([]memorypkg.ChatMessage, len(memoryGenerateMessages))
			for i, m := range memoryGenerateMessages {
				msgs[i] = memorypkg.ChatMessage{Role: "user", Content: m}
			}
			req = &memorypkg.MemoryRecordGenerateFromContentRequest{ChatMessages: msgs}
		} else {
			return fmt.Errorf("provide --file or at least one --message")
		}

		client, err := newMemoryClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.GenerateRecordsFromContent(ctx, args[0], strategyID, actorID, sessionID, req); err != nil {
			return err
		}
		output.Successf("Record generation from content queued (%d message(s)).", len(req.ChatMessages))
		return nil
	},
}

// renderSessionEvents renders the raw event messages as a best-effort table.
// The memory service publishes no response schema for the events list, so each
// event is parsed leniently; unparseable elements render as <raw>.
func renderSessionEvents(events []json.RawMessage) error {
	fmt.Fprintf(os.Stderr, "%d event(s).\n", len(events))
	if len(events) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(events))
	for i, raw := range events {
		var v struct {
			Type           string `json:"type"`
			Message        string `json:"message"`
			EventTimestamp string `json:"eventTimestamp"`
			LastTimestamp  string `json:"lastTimestamp"`
			Timestamp      string `json:"timestamp"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			rows = append(rows, []string{fmt.Sprintf("%d", i), "<raw>", truncate(string(raw), 60)})
			continue
		}
		ts := v.EventTimestamp
		if ts == "" {
			ts = v.LastTimestamp
		}
		if ts == "" {
			ts = v.Timestamp
		}
		rows = append(rows, []string{fmt.Sprintf("%d", i), v.Type, truncate(v.Message, 60), ts})
	}
	output.Table([]string{"#", "Type", "Message", "Timestamp"}, rows)
	return nil
}

// truncate clips s to n runes with an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// loadGenerateFromContentSpec parses a YAML/JSON spec (a chatMessages array)
// into MemoryRecordGenerateFromContentRequest.
func loadGenerateFromContentSpec(data []byte) (*memorypkg.MemoryRecordGenerateFromContentRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	// The file is a chatMessages array, not an object wrapping it. Wrap so it
	// decodes into the request struct.
	if _, ok := m["chatMessages"]; !ok {
		m = map[string]interface{}{"chatMessages": m}
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req memorypkg.MemoryRecordGenerateFromContentRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid generate-from-content spec: %w", err)
	}
	if len(req.ChatMessages) == 0 {
		return nil, fmt.Errorf("spec has no chatMessages")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(memoryCmd)

	// create
	create := memoryCreateCmd
	create.Flags().StringVarP(&memoryName, "name", "n", "", "Memory name (required without --interactive)")
	create.Flags().String("description", "", "Description (required)")
	create.Flags().Int("event-expiry-duration", 30, "Short-term event TTL in days (1-365)")
	create.Flags().String("strategy-name", "", "Single strategy: name (use --file for multiple)")
	create.Flags().String("strategy-type", "", "Single strategy: type (USER_PREFERENCE|SEMANTIC|CUSTOM|...)")
	create.Flags().String("strategy-namespace", "", "Single strategy: namespace template (e.g. /strategies/SEMANTIC/actors/{actorId})")
	create.Flags().String("strategy-prompt", "", "Single strategy: custom fact-extraction prompt (max 1000)")
	create.Flags().Bool("strategy-auto-generate", false, "Single strategy: auto-generate memory records from events")
	create.Flags().StringVar(&memoryFile, "file", "", "Apply a spec file (see 'generate'); authoritative when set")
	memoryCmd.AddCommand(create)

	// generate
	memoryCmd.AddCommand(memoryGenerateCmd)

	// list
	memoryListCmd.Flags().Int("page", 1, "Page number (1-based)")
	memoryListCmd.Flags().Int("size", 10, "Page size")
	memoryCmd.AddCommand(memoryListCmd)

	// get
	memoryCmd.AddCommand(memoryGetCmd)

	// delete
	memoryCmd.AddCommand(memoryDeleteCmd)

	// search
	memorySearchCmd.Flags().String("namespace", "", "Resolved namespace (required, e.g. /strategies/SEMANTIC/actors/<actor>)")
	memorySearchCmd.Flags().String("query", "", "Search query (required)")
	memorySearchCmd.Flags().Int("limit", 100, "Max results (5-200)")
	memorySearchCmd.Flags().Float64("threshold", 0, "Min relevance score (0-1)")
	memoryCmd.AddCommand(memorySearchCmd)

	// --- sub-resources (Slice 3) ---

	// actor
	memoryActorListCmd.Flags().Int("page", 1, "Page number (1-based)")
	memoryActorListCmd.Flags().Int("size", 10, "Page size")
	memoryActorCmd.AddCommand(memoryActorListCmd)
	memoryCmd.AddCommand(memoryActorCmd)

	// session
	memorySessionListCmd.Flags().Int("page", 1, "Page number (1-based)")
	memorySessionListCmd.Flags().Int("size", 10, "Page size")
	memorySessionCmd.AddCommand(memorySessionListCmd)
	memoryCmd.AddCommand(memorySessionCmd)

	// event
	memoryEventListCmd.Flags().String("from", "", "fromTimestamp (RFC3339)")
	memoryEventListCmd.Flags().String("to", "", "toTimestamp (RFC3339)")
	memoryEventListCmd.Flags().Int("page", 1, "Page number (1-based)")
	memoryEventListCmd.Flags().Int("size", 100, "Page size (capped to 100)")
	memoryEventCmd.AddCommand(memoryEventListCmd)

	memoryEventCreateCmd.Flags().String("type", "", "Event type (required)")
	memoryEventCreateCmd.Flags().String("role", "", "Event role")
	memoryEventCreateCmd.Flags().String("message", "", "Event message (max 100k chars)")
	memoryEventCreateCmd.Flags().String("binary-data", "", "Event binary data (max ~10 MiB)")
	memoryEventCreateCmd.Flags().String("event-timestamp", "", "Event timestamp (RFC3339)")
	memoryEventCmd.AddCommand(memoryEventCreateCmd)

	memoryEventCmd.AddCommand(memoryEventDeleteCmd)
	memoryCmd.AddCommand(memoryEventCmd)

	// strategy
	memoryStrategyCmd.AddCommand(memoryStrategyListCmd)
	memoryCmd.AddCommand(memoryStrategyCmd)

	// record
	memoryRecordListCmd.Flags().String("namespace", "", "Resolved namespace (required)")
	memoryRecordListCmd.Flags().Int("limit", 100, "Max results")
	memoryRecordCmd.AddCommand(memoryRecordListCmd)

	memoryRecordCmd.AddCommand(memoryRecordDeleteCmd)

	memoryRecordInsertCmd.Flags().String("namespace", "", "Resolved namespace (required)")
	memoryRecordInsertCmd.Flags().StringArrayVar(&memoryInsertRecords, "record", nil, "Record text (repeatable; at least one required)")
	memoryRecordCmd.AddCommand(memoryRecordInsertCmd)

	memoryRecordGenerateFromSessionCmd.Flags().String("actor-id", "", "Actor id (required)")
	memoryRecordGenerateFromSessionCmd.Flags().String("session-id", "", "Session id (required)")
	memoryRecordGenerateFromSessionCmd.Flags().String("strategy-id", "", "Long-term-memory strategy id (required)")
	memoryRecordCmd.AddCommand(memoryRecordGenerateFromSessionCmd)

	memoryRecordGenerateFromContentCmd.Flags().String("strategy-id", "", "Long-term-memory strategy id (required)")
	memoryRecordGenerateFromContentCmd.Flags().String("actor-id", "", "Actor id (optional)")
	memoryRecordGenerateFromContentCmd.Flags().String("session-id", "", "Session id (optional)")
	memoryRecordGenerateFromContentCmd.Flags().StringVar(&memoryGenerateFile, "file", "", "Spec file with a chatMessages array (authoritative when set)")
	memoryRecordGenerateFromContentCmd.Flags().StringArrayVar(&memoryGenerateMessages, "message", nil, "Chat message content, role=user (repeatable)")
	memoryRecordCmd.AddCommand(memoryRecordGenerateFromContentCmd)

	memoryCmd.AddCommand(memoryRecordCmd)
}

// memoryName holds the --name value for create.
var memoryName string
