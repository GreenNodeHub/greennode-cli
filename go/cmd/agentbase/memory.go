package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vngcloud/greennode-cli/internal/agentbase/cliinput"
	memorypkg "github.com/vngcloud/greennode-cli/internal/agentbase/memory"
	"github.com/vngcloud/greennode-cli/internal/agentbase/output"
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
}

// memoryName holds the --name value for create.
var memoryName string
