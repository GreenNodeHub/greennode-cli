package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	runtimepkg "github.com/greennodehub/greennode-cli/internal/agentbase/runtime"
)

// runtimeCmd groups the agent-runtime lifecycle commands. The agentbase /runtime
// endpoint fronts the agent-core-runtime REST API (POST/GET/PATCH/DELETE
// /agent-runtimes — note: bare paths, no /api/v1). An agent runtime is a
// deployable container (image/command/args/env/autoscaling); it converges
// asynchronously (CREATING → ACTIVE), so create/update/delete return immediately
// and `wait` polls to a terminal state.
//
// The runtime API keys on `id`, not name (name is immutable). Update is a
// full-spec replacement (every field required), not a merge-patch.
var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Manage agent runtimes",
	Long: `Create and manage agent runtimes (the runtime service).

An agent runtime is a deployable container defined by its image, command, args,
environment, autoscaling, and flavor. Runtimes converge asynchronously:
create/update return the resource in a CREATING/UPDATING state and reach ACTIVE
(or ERROR / SERVICE_ACCOUNT_ERROR). Use 'wait' to block until terminal.

The runtime API addresses resources by id (not name; name is immutable). Update
is a full-spec replacement — every field is required, so for anything beyond the
simple path, generate a template, fill it in, and apply with --file:

    grn agentbase runtime generate > rt.yaml
    # ...edit rt.yaml...
    grn agentbase runtime create --file rt.yaml
    grn agentbase runtime wait <id>

Runtimes share the ~/.greennode profile like the rest of agentbase.`,
}

// newRuntimeClient mirrors newGatewayClient: resolve the shared profile + env,
// select the shared token provider, force-mint once so auth failures surface
// before the first call, and point the typed client at the runtime endpoint.
func newRuntimeClient(ctx context.Context, cmd *cobra.Command) (*runtimepkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return runtimepkg.NewClient(ab.endpoints.Runtime, provider), nil
}

// ---------------------------------------------------------------------------
// shared spec flags + reader (create and update share every field except name)
// ---------------------------------------------------------------------------

// runtimeSpec holds the mutable spec fields common to create and update.
type runtimeSpec struct {
	Description string
	ImageURL    string
	ImageAuth   *runtimepkg.ImageAuth
	Command     []string
	Args        []string
	Env         map[string]string
	FlavorID    string
	Autoscaling runtimepkg.Autoscaling
}

// readRuntimeSpec reads + prompts the mutable spec fields. name is handled by
// the caller (create-only). Required (@NotEmpty) fields: imageUrl, flavorId;
// autoscaling has flag defaults so it is always populated.
func readRuntimeSpec(cmd *cobra.Command) (*runtimeSpec, error) {
	f := cmd.Flags()

	description, _ := f.GetString("description")

	imageURL, _ := f.GetString("image-url")
	imageURL, err := cliinput.RequireOrPromptString(imageURL, "--image-url", "Container image URL")
	if err != nil {
		return nil, err
	}

	flavorID, _ := f.GetString("flavor-id")
	flavorID, err = cliinput.RequireOrPromptString(flavorID, "--flavor-id", "Flavor id")
	if err != nil {
		return nil, err
	}

	command, _ := f.GetStringArray("command")
	args, _ := f.GetStringArray("args")

	envRaw, _ := f.GetStringArray("env")
	env, err := parseEnvVars(envRaw)
	if err != nil {
		return nil, err
	}

	minReplicas, _ := f.GetInt("min-replicas")
	maxReplicas, _ := f.GetInt("max-replicas")
	cpuUtil, _ := f.GetInt("cpu-utilization")
	memUtil, _ := f.GetInt("memory-utilization")
	autoscaling := runtimepkg.Autoscaling{
		MinReplicas:       minReplicas,
		MaxReplicas:       maxReplicas,
		CPUUtilization:    cpuUtil,
		MemoryUtilization: memUtil,
	}
	if autoscaling.MaxReplicas < autoscaling.MinReplicas {
		return nil, fmt.Errorf("--max-replicas (%d) must be >= --min-replicas (%d)",
			autoscaling.MaxReplicas, autoscaling.MinReplicas)
	}

	imageAuth, err := readImageAuth(cmd)
	if err != nil {
		return nil, err
	}

	return &runtimeSpec{
		Description: description,
		ImageURL:    imageURL,
		ImageAuth:   imageAuth,
		Command:     command,
		Args:        args,
		Env:         env,
		FlavorID:    flavorID,
		Autoscaling: autoscaling,
	}, nil
}

// readImageAuth builds *ImageAuth only when --image-auth-enabled is set (or a
// username is supplied). username/password are required when auth is enabled.
// The password is send-only (never present on responses); prefer --file for it.
func readImageAuth(cmd *cobra.Command) (*runtimepkg.ImageAuth, error) {
	f := cmd.Flags()
	enabled, _ := f.GetBool("image-auth-enabled")
	username, _ := f.GetString("image-auth-username")
	password, _ := f.GetString("image-auth-password")
	if !enabled && username == "" {
		return nil, nil
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("--image-auth-enabled requires both --image-auth-username and --image-auth-password (or use --file)")
	}
	return &runtimepkg.ImageAuth{Enabled: enabled, Username: username, Password: password}, nil
}

// parseEnvVars parses repeated --env KEY=VALUE values into a map. An empty input
// yields an empty map (the API requires the field present but allows empty).
func parseEnvVars(raw []string) (map[string]string, error) {
	env := map[string]string{}
	for _, r := range raw {
		eq := strings.Index(r, "=")
		if eq <= 0 {
			return nil, fmt.Errorf("invalid --env %q: expected KEY=VALUE", r)
		}
		env[strings.TrimSpace(r[:eq])] = r[eq+1:]
	}
	return env, nil
}

// ---------------------------------------------------------------------------
// create
// ---------------------------------------------------------------------------

// runtimeFile is shared by create and update (--file); only one runs at a time.
var runtimeFile string

var runtimeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent runtime",
	Long: `Create a new agent runtime.

By default the runtime is built from flags (the simple path). For environment
variables, private-registry auth, or multi-element command/args, use --file with
a template produced by 'grn agentbase runtime generate'.

The runtime is created asynchronously; this command returns as soon as the
service accepts the spec (state CREATING). Converge with
'grn agentbase runtime wait <id>'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if runtimeFile != "" {
			data, err := os.ReadFile(runtimeFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err := loadRuntimeSpec(data)
			if err != nil {
				return err
			}
			return createRuntimeAndPrint(ctx, cmd, req)
		}
		name, _ := cmd.Flags().GetString("name")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Runtime name")
		if err != nil {
			return err
		}
		spec, err := readRuntimeSpec(cmd)
		if err != nil {
			return err
		}
		return createRuntimeAndPrint(ctx, cmd, &runtimepkg.CreateAgentRuntimeRequest{
			Name:                 name,
			Description:          spec.Description,
			ImageURL:             spec.ImageURL,
			ImageAuth:            spec.ImageAuth,
			Command:              spec.Command,
			Args:                 spec.Args,
			EnvironmentVariables: spec.Env,
			FlavorID:             spec.FlavorID,
			Autoscaling:          spec.Autoscaling,
		})
	},
}

func createRuntimeAndPrint(ctx context.Context, cmd *cobra.Command, req *runtimepkg.CreateAgentRuntimeRequest) error {
	client, err := newRuntimeClient(ctx, cmd)
	if err != nil {
		return err
	}
	rt, err := client.Create(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Runtime %q submitted (id %s, state %s). Run 'grn agentbase runtime wait %s' to converge.\n",
		rt.Name, rt.ID, rt.Status, rt.ID)
	return output.PrintResource(rt, func() string { return rt.ID }, func() error { return renderRuntimeDetail(rt) })
}

// ---------------------------------------------------------------------------
// generate — print a commented create-template (kubectl-style)
// ---------------------------------------------------------------------------

var runtimeGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print a runtime create template (YAML or JSON)",
	Long: `Print a commented agent-runtime create template to stdout. Save it, fill it in,
and apply with 'grn agentbase runtime create --file <file>'.

Defaults to YAML (with comments); pass -o json for a JSON skeleton.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			example := &runtimepkg.CreateAgentRuntimeRequest{
				Name:                 "my-runtime",
				Description:          "",
				ImageURL:             "fill-image-url",
				Command:              []string{},
				Args:                 []string{},
				EnvironmentVariables: map[string]string{},
				FlavorID:             "fill-flavor-id",
				Autoscaling:          runtimepkg.Autoscaling{MinReplicas: 1, MaxReplicas: 2, CPUUtilization: 70, MemoryUtilization: 70},
			}
			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(runtimeCreateTemplateYAML)
		return nil
	},
}

// runtimeCreateTemplateYAML is a hand-written, commented skeleton of
// CreateAgentRuntimeRequest. Keys are the JSON (camelCase) field names so the
// file round-trips through 'create --file' exactly.
const runtimeCreateTemplateYAML = `# Agent runtime create spec.
# Fill in and apply with:  grn agentbase runtime create --file <this-file>
#
# Required: name, imageUrl, flavorId, autoscaling (and command/args/env, which
# may be empty lists/map but must be present). name is sealed (immutable).
name: my-runtime             # immutable
description: ""
imageUrl: registry.example.com/my-agent:latest
# imageAuth:                 # optional private-registry credentials (send-only)
#   enabled: true
#   username: ""
#   password: ""             # prefer keeping this in the file, not on the CLI
command:                     # container entrypoint (list); may be empty
  - python
  - -m
  - my_agent
args: []                     # container args (list); may be empty
environmentVariables:        # map; may be empty
  LOG_LEVEL: info
flavorId: fill-flavor-id
autoscaling:                 # HPA config (bounds: replicas 1-10, util 10-90)
  minReplicas: 1
  maxReplicas: 2
  cpuUtilization: 70
  memoryUtilization: 70
`

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

var runtimeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent runtimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newRuntimeClient(ctx, cmd)
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
			fmt.Fprintln(os.Stderr, "No agent runtimes found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			rt := resp.ListData[i]
			rows = append(rows, []string{
				rt.ID, rt.Name, rt.Status, formatTimeVal(rt.CreatedAt),
			})
		}
		// Columns are limited to the AgentRuntime DTO's fields (id/name/status/
		// createdAt/...). The DTO does not echo flavorId/replicas/image back, so
		// those are not shown here; use the gateway or version sub-resource for
		// the full spec.
		output.Table([]string{"ID", "Name", "State", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

var runtimeGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show an agent runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		rt, err := client.Get(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(rt, func() string { return rt.ID }, func() error { return renderRuntimeDetail(rt) })
	},
}

// ---------------------------------------------------------------------------
// update (full-spec replacement, NOT merge-patch)
// ---------------------------------------------------------------------------

var runtimeUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an agent runtime (full-spec replacement)",
	Long: `Update an agent runtime. Unlike gateway, this is a FULL-SPEC replacement, not a
merge-patch: every field is required server-side (the create spec minus name).
Updating creates a new version and rolls the default endpoint forward.

For anything beyond the simple path, use --file with the create template
('grn agentbase runtime generate') minus the name field.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]
		if runtimeFile != "" {
			data, err := os.ReadFile(runtimeFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			createReq, err := loadRuntimeSpec(data)
			if err != nil {
				return err
			}
			return updateRuntimeAndPrint(ctx, cmd, id, &runtimepkg.UpdateAgentRuntimeRequest{
				Description:          createReq.Description,
				ImageURL:             createReq.ImageURL,
				ImageAuth:            createReq.ImageAuth,
				Command:              createReq.Command,
				Args:                 createReq.Args,
				EnvironmentVariables: createReq.EnvironmentVariables,
				FlavorID:             createReq.FlavorID,
				Autoscaling:          createReq.Autoscaling,
			})
		}
		spec, err := readRuntimeSpec(cmd)
		if err != nil {
			return err
		}
		return updateRuntimeAndPrint(ctx, cmd, id, &runtimepkg.UpdateAgentRuntimeRequest{
			Description:          spec.Description,
			ImageURL:             spec.ImageURL,
			ImageAuth:            spec.ImageAuth,
			Command:              spec.Command,
			Args:                 spec.Args,
			EnvironmentVariables: spec.Env,
			FlavorID:             spec.FlavorID,
			Autoscaling:          spec.Autoscaling,
		})
	},
}

func updateRuntimeAndPrint(ctx context.Context, cmd *cobra.Command, id string, req *runtimepkg.UpdateAgentRuntimeRequest) error {
	client, err := newRuntimeClient(ctx, cmd)
	if err != nil {
		return err
	}
	rt, err := client.Update(ctx, id, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Runtime %q updating (state %s). Run 'grn agentbase runtime wait %s' to converge.\n",
		rt.Name, rt.Status, rt.ID)
	return output.PrintResource(rt, func() string { return rt.ID }, func() error { return renderRuntimeDetail(rt) })
}

// ---------------------------------------------------------------------------
// delete
// ---------------------------------------------------------------------------

var runtimeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an agent runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.Delete(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Runtime %q deleting. Run 'grn agentbase runtime wait %s' to confirm.\n", args[0], args[0])
		output.PrintDeletedID(args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// wait — poll get until a terminal state
// ---------------------------------------------------------------------------

var runtimeWaitCmd = &cobra.Command{
	Use:   "wait <id>",
	Short: "Wait for an agent runtime to reach a terminal state",
	Long: `Poll an agent runtime until it reaches a terminal state: ACTIVE / DELETED
(success) or ERROR / SERVICE_ACCOUNT_ERROR (failure). Use after create/update/delete.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		pctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		rt, err := pollRuntimeToTerminal(pctx, client, args[0], interval)
		if err != nil {
			return err
		}
		return output.PrintResource(rt, func() string { return rt.ID }, func() error { return renderRuntimeDetail(rt) })
	},
}

// pollRuntimeToTerminal polls a runtime by id until it reaches a terminal state
// (ACTIVE/DELETED = success, ERROR/SERVICE_ACCOUNT_ERROR = failure) or ctx
// expires. Shared by `runtime wait`, `deploy up`, and `deploy destroy` so there
// is one polling loop. Progress lines go to stderr. Returns the last-fetched
// runtime and a non-nil error on failure/timeout (the caller decides whether to
// render it).
func pollRuntimeToTerminal(ctx context.Context, client *runtimepkg.Client, id string, interval time.Duration) (*runtimepkg.AgentRuntime, error) {
	for {
		rt, err := client.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if runtimeTerminalFailure[rt.Status] {
			return rt, fmt.Errorf("runtime %q failed: state=%s %s", id, rt.Status, rt.StatusReason)
		}
		if runtimeTerminalSuccess[rt.Status] {
			fmt.Fprintf(os.Stderr, "Runtime %q reached state %s.\n", id, rt.Status)
			return rt, nil
		}
		fmt.Fprintf(os.Stderr, "Runtime %q state: %s ...\n", id, rt.Status)
		select {
		case <-ctx.Done():
			return rt, fmt.Errorf("timed out waiting for runtime %q (last state %s)", id, rt.Status)
		case <-time.After(interval):
		}
	}
}

// runtimeTerminalSuccess / runtimeTerminalFailure partition the AgentRuntime
// status enum. CREATING/DELETING/UPDATING are transient (the runtime has no
// WAITING_* states — those live on the underlying endpoint).
var (
	runtimeTerminalSuccess = map[string]bool{
		"ACTIVE":  true,
		"DELETED": true,
	}
	runtimeTerminalFailure = map[string]bool{
		"ERROR":                 true,
		"SERVICE_ACCOUNT_ERROR": true,
	}
)

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderRuntimeDetail(rt *runtimepkg.AgentRuntime) error {
	rows := [][]string{
		{"ID", rt.ID},
		{"Name", rt.Name},
		{"Description", output.StrOrDash(rt.Description)},
		{"State", rt.Status},
		{"Status Reason", output.StrOrDash(rt.StatusReason)},
		{"Created", formatTimeVal(rt.CreatedAt)},
		{"Updated", formatTimeVal(rt.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

// ---------------------------------------------------------------------------
// file parsing (YAML or JSON -> struct)
// ---------------------------------------------------------------------------

// loadRuntimeSpec parses a YAML/JSON create spec into CreateAgentRuntimeRequest.
// yaml.Unmarshal into a map then json.Unmarshal into the struct so the file's
// camelCase keys bind to the struct's json tags (yaml.v3 does not honor json
// tags directly). Used by both create (--file) and update (--file, minus name).
func loadRuntimeSpec(data []byte) (*runtimepkg.CreateAgentRuntimeRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req runtimepkg.CreateAgentRuntimeRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid runtime spec: %w", err)
	}
	if req.Name == "" || req.ImageURL == "" || req.FlavorID == "" {
		return nil, fmt.Errorf("spec is missing required field(s): name, imageUrl, flavorId")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Sub-resources (Slice 4): endpoints / runtime-level ops / tracing
// ---------------------------------------------------------------------------

var runtimeEndpointCmd = &cobra.Command{Use: "endpoint", Short: "Manage a runtime's endpoints"}
var runtimeTraceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Query the tracing backend (raw passthrough)",
	Long: `Query the runtime tracing backend.

These are Google-AIP custom verbs that forward arbitrary query params to the
tracing backend and return that backend's raw JSON. Use --param key=value
(repeatable) to pass query params through verbatim; the response is printed
raw (use -o json for clean stdout).`,
}

// --- endpoint list/create/update/delete/start/stop ---

var runtimeEndpointListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List a runtime's endpoints",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListEndpoints(ctx, args[0], page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(resp.ListData[0].ID)
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No endpoints found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			e := resp.ListData[i]
			rows = append(rows, []string{
				e.ID, e.Name, fmt.Sprintf("%d", e.Version), fmt.Sprintf("%d", e.LiveVersion),
				fmt.Sprintf("%d", e.CurrentReplicaCount), e.URL, e.DisplayStatus,
			})
		}
		output.Table([]string{"ID", "Name", "Version", "Live", "Replicas", "URL", "Status"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

var runtimeEndpointCreateCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "Create a new endpoint on a runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("required flag %q not set", "name")
		}
		version, _ := cmd.Flags().GetInt("version")
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		ep, err := client.CreateEndpoint(ctx, args[0], &runtimepkg.AgentRuntimeEndpointCreateRequest{
			Name: name, Version: version,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Endpoint %q created (id %s, status %s).\n", ep.Name, ep.ID, ep.DisplayStatus)
		return output.PrintResource(ep, func() string { return ep.ID }, func() error { return renderEndpointDetail(ep) })
	},
}

var runtimeEndpointUpdateCmd = &cobra.Command{
	Use:   "update <id> <endpoint-id>",
	Short: "Roll an endpoint to a target version",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		version, _ := cmd.Flags().GetInt("version")
		if version <= 0 {
			return fmt.Errorf("required flag %q must be a positive version", "version")
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		ep, err := client.UpdateEndpoint(ctx, args[0], args[1], version)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Endpoint %s rolled to version %d.\n", args[1], ep.Version)
		return output.PrintResource(ep, func() string { return ep.ID }, func() error { return renderEndpointDetail(ep) })
	},
}

var runtimeEndpointDeleteCmd = &cobra.Command{
	Use:   "delete <id> <endpoint-id>",
	Short: "Delete an endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		ep, err := client.DeleteEndpoint(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Endpoint %q deleted.\n", args[1])
		if ep.ID != "" {
			return output.PrintResource(ep, func() string { return ep.ID }, func() error { return renderEndpointDetail(ep) })
		}
		output.PrintDeletedID(args[1])
		return nil
	},
}

var runtimeEndpointStartCmd = &cobra.Command{
	Use:   "start <id> <endpoint-id>",
	Short: "Start an endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.StartEndpoint(ctx, args[0], args[1]); err != nil {
			return err
		}
		output.Successf("Endpoint %s started.", args[1])
		return nil
	},
}

var runtimeEndpointStopCmd = &cobra.Command{
	Use:   "stop <id> <endpoint-id>",
	Short: "Stop an endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.StopEndpoint(ctx, args[0], args[1]); err != nil {
			return err
		}
		output.Successf("Endpoint %s stopped.", args[1])
		return nil
	},
}

// --- endpoint logs / metrics / events ---

var runtimeEndpointLogsCmd = &cobra.Command{
	Use:   "logs <id> <endpoint-id>",
	Short: "Search an endpoint's logs",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		req, err := readLogSearchRequest(cmd)
		if err != nil {
			return err
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.EndpointLogs(ctx, args[0], args[1], req)
		if err != nil {
			return err
		}
		return renderLogs(result)
	},
}

var runtimeEndpointMetricsCmd = &cobra.Command{
	Use:   "metrics <id> <endpoint-id>",
	Short: "Fetch an endpoint's CPU/memory metrics",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		m, err := client.EndpointMetrics(ctx, args[0], args[1], from, to)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(m)
		case output.FormatID:
			return nil
		}
		return renderMetrics(m)
	},
}

var runtimeEndpointEventsCmd = &cobra.Command{
	Use:   "events <id> <endpoint-id>",
	Short: "List kubernetes events for an endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		events, err := client.EndpointEvents(ctx, args[0], args[1])
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(events)
		case output.FormatID:
			return nil
		}
		if len(events) == 0 {
			fmt.Fprintln(os.Stderr, "No events found.")
			return nil
		}
		rows := make([][]string, 0, len(events))
		for i := range events {
			e := events[i]
			rows = append(rows, []string{formatTimeVal(e.LastTimestamp), e.Message})
		}
		output.Table([]string{"Last Timestamp", "Message"}, rows)
		return nil
	},
}

// --- runtime-level logs / reset-service-account / versions ---

var runtimeLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Search a runtime's logs (runtime-level)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		req, err := readLogSearchRequest(cmd)
		if err != nil {
			return err
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		result, err := client.Logs(ctx, args[0], req)
		if err != nil {
			return err
		}
		return renderLogs(result)
	},
}

var runtimeResetServiceAccountCmd = &cobra.Command{
	Use:   "reset-service-account <id>",
	Short: "Reset a runtime's IAM service account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.ResetServiceAccount(ctx, args[0]); err != nil {
			return err
		}
		output.Successf("Service account reset for runtime %s.", args[0])
		return nil
	},
}

var runtimeVersionsCmd = &cobra.Command{
	Use:   "versions <id>",
	Short: "List a runtime's versions (full spec per version)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListVersions(ctx, args[0], page, size)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.ListData) > 0 {
				output.PrintID(fmt.Sprintf("%d", resp.ListData[0].Version))
			}
			return nil
		}
		if len(resp.ListData) == 0 {
			fmt.Fprintln(os.Stderr, "No versions found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.ListData))
		for i := range resp.ListData {
			v := resp.ListData[i]
			mode := ""
			if v.NetworkConfig != nil {
				mode = v.NetworkConfig.Mode
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", v.Version), v.ImageURL, v.FlavorID, v.Protocol, mode, formatTimeVal(v.CreatedAt),
			})
		}
		output.Table([]string{"Version", "Image", "Flavor", "Protocol", "Network Mode", "Created"}, rows)
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", resp.Page, resp.TotalPage, resp.TotalItem)
		return nil
	},
}

// --- trace get / search / tag-values ---

// runtimeTraceParams holds the repeatable --param key=value values.
var runtimeTraceParams []string

var runtimeTraceGetCmd = &cobra.Command{
	Use:   "get <trace-id>",
	Short: "Fetch a single trace by id (raw JSON)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		params, err := parseTraceParams(runtimeTraceParams)
		if err != nil {
			return err
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		raw, err := client.GetTrace(ctx, args[0], params)
		if err != nil {
			return err
		}
		return printRawJSON(raw)
	},
}

var runtimeTraceSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search traces (raw JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		params, err := parseTraceParams(runtimeTraceParams)
		if err != nil {
			return err
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		raw, err := client.SearchTraces(ctx, params)
		if err != nil {
			return err
		}
		return printRawJSON(raw)
	},
}

var runtimeTraceTagValuesCmd = &cobra.Command{
	Use:   "tag-values",
	Short: "List distinct values for a trace tag (raw JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		tagKey, _ := cmd.Flags().GetString("tag-key")
		if tagKey == "" {
			return fmt.Errorf("required flag %q not set", "tag-key")
		}
		params, err := parseTraceParams(runtimeTraceParams)
		if err != nil {
			return err
		}
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		raw, err := client.TraceTagValues(ctx, tagKey, params)
		if err != nil {
			return err
		}
		return printRawJSON(raw)
	},
}

// ---------------------------------------------------------------------------
// helpers (Slice 4)
// ---------------------------------------------------------------------------

func renderEndpointDetail(ep *runtimepkg.AgentRuntimeEndpointDto) error {
	rows := [][]string{
		{"ID", ep.ID},
		{"Runtime ID", ep.AgentRuntimeID},
		{"Name", ep.Name},
		{"Version", fmt.Sprintf("%d", ep.Version)},
		{"Target Version", fmt.Sprintf("%d", ep.TargetVersion)},
		{"Live Version", fmt.Sprintf("%d", ep.LiveVersion)},
		{"Replicas", fmt.Sprintf("%d", ep.CurrentReplicaCount)},
		{"URL", output.StrOrDash(ep.URL)},
		{"Status", ep.Status},
		{"Display Status", ep.DisplayStatus},
		{"Created", formatTimeVal(ep.CreatedAt)},
		{"Updated", formatTimeVal(ep.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

func renderLogs(result *runtimepkg.LogSearchResult) error {
	switch output.GetFormat() {
	case output.FormatJSON:
		return output.JSON(result)
	case output.FormatID:
		return nil
	}
	fmt.Fprintf(os.Stderr, "%d log line(s) (total %d).\n", len(result.Logs), result.TotalCount)
	if len(result.Logs) == 0 {
		fmt.Fprintln(os.Stderr, "No logs found.")
		return nil
	}
	rows := make([][]string, 0, len(result.Logs))
	for i := range result.Logs {
		l := result.Logs[i]
		rows = append(rows, []string{l.Timestamp, l.Content})
	}
	output.Table([]string{"Timestamp", "Content"}, rows)
	return nil
}

func renderMetrics(m *runtimepkg.AgentRuntimeEndpointMetrics) error {
	if len(m.CpuCoresUsage) == 0 && len(m.MemoryBytesUsage) == 0 {
		fmt.Fprintln(os.Stderr, "No metrics found.")
		return nil
	}
	rows := make([][]string, 0, len(m.CpuCoresUsage)+len(m.MemoryBytesUsage))
	for i := range m.CpuCoresUsage {
		p := m.CpuCoresUsage[i]
		rows = append(rows, []string{"CPU cores", formatTimeVal(p.Timestamp), fmt.Sprintf("%.3f", p.Value)})
	}
	for i := range m.MemoryBytesUsage {
		p := m.MemoryBytesUsage[i]
		rows = append(rows, []string{"Memory bytes", formatTimeVal(p.Timestamp), fmt.Sprintf("%d", p.Value)})
	}
	output.Table([]string{"Series", "Timestamp", "Value"}, rows)
	return nil
}

// readLogSearchRequest builds a LogSearchRequest from the shared log flags.
func readLogSearchRequest(cmd *cobra.Command) (*runtimepkg.LogSearchRequest, error) {
	from, _ := cmd.Flags().GetInt("from")
	limit, _ := cmd.Flags().GetInt("limit")
	fromTs, _ := cmd.Flags().GetString("from-timestamp")
	toTs, _ := cmd.Flags().GetString("to-timestamp")
	query, _ := cmd.Flags().GetString("query")
	order, _ := cmd.Flags().GetString("order")
	return &runtimepkg.LogSearchRequest{
		From: from, Limit: limit, FromTimestamp: fromTs, ToTimestamp: toTs, Query: query, Order: order,
	}, nil
}

// parseTraceParams turns --param key=value entries into url.Values.
func parseTraceParams(raw []string) (url.Values, error) {
	q := url.Values{}
	for _, p := range raw {
		idx := strings.IndexByte(p, '=')
		if idx <= 0 {
			return nil, fmt.Errorf("--param %q must be key=value", p)
		}
		q.Set(p[:idx], p[idx+1:])
	}
	return q, nil
}

// printRawJSON writes the tracing backend's raw JSON to stdout verbatim.
func printRawJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	_, err := fmt.Println(string(raw))
	return err
}

func init() {
	AgentbaseCmd.AddCommand(runtimeCmd)

	// create
	create := runtimeCreateCmd
	addRuntimeSpecFlags(create)
	create.Flags().StringVarP(&runtimeName, "name", "n", "", "Runtime name (required without --interactive, immutable)")
	create.Flags().StringVar(&runtimeFile, "file", "", "Apply a spec file (see 'generate'); authoritative when set")
	runtimeCmd.AddCommand(create)

	// generate
	runtimeCmd.AddCommand(runtimeGenerateCmd)

	// list
	runtimeListCmd.Flags().Int("page", 1, "Page number (1-based)")
	runtimeListCmd.Flags().Int("size", 10, "Page size")
	runtimeCmd.AddCommand(runtimeListCmd)

	// get
	runtimeCmd.AddCommand(runtimeGetCmd)

	// update (same spec flags minus --name; reuses --file)
	update := runtimeUpdateCmd
	addRuntimeSpecFlags(update)
	update.Flags().StringVar(&runtimeFile, "file", "", "Apply a full-spec file (create template minus name)")
	runtimeCmd.AddCommand(update)

	// delete
	runtimeCmd.AddCommand(runtimeDeleteCmd)

	// wait
	runtimeWaitCmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait")
	runtimeWaitCmd.Flags().Duration("interval", 5*time.Second, "Poll interval")
	runtimeCmd.AddCommand(runtimeWaitCmd)

	// --- sub-resources (Slice 4): endpoints ---

	// endpoint list
	runtimeEndpointListCmd.Flags().Int("page", 1, "Page number (1-based)")
	runtimeEndpointListCmd.Flags().Int("size", 10, "Page size")
	runtimeEndpointCmd.AddCommand(runtimeEndpointListCmd)

	// endpoint create
	runtimeEndpointCreateCmd.Flags().String("name", "", "Endpoint name (required)")
	runtimeEndpointCreateCmd.Flags().Int("version", 0, "Target version (optional, defaults server-side)")
	runtimeEndpointCmd.AddCommand(runtimeEndpointCreateCmd)

	// endpoint update
	runtimeEndpointUpdateCmd.Flags().Int("version", 0, "Target version (required, positive)")
	runtimeEndpointCmd.AddCommand(runtimeEndpointUpdateCmd)

	// endpoint delete
	runtimeEndpointCmd.AddCommand(runtimeEndpointDeleteCmd)

	// endpoint start / stop
	runtimeEndpointCmd.AddCommand(runtimeEndpointStartCmd)
	runtimeEndpointCmd.AddCommand(runtimeEndpointStopCmd)

	// endpoint logs
	addLogFlags(runtimeEndpointLogsCmd)
	runtimeEndpointCmd.AddCommand(runtimeEndpointLogsCmd)

	// endpoint metrics
	runtimeEndpointMetricsCmd.Flags().String("from", "", "fromTimestamp (RFC3339)")
	runtimeEndpointMetricsCmd.Flags().String("to", "", "toTimestamp (RFC3339)")
	runtimeEndpointCmd.AddCommand(runtimeEndpointMetricsCmd)

	// endpoint events
	runtimeEndpointCmd.AddCommand(runtimeEndpointEventsCmd)

	runtimeCmd.AddCommand(runtimeEndpointCmd)

	// --- sub-resources: runtime-level ops ---

	// runtime logs
	addLogFlags(runtimeLogsCmd)
	runtimeCmd.AddCommand(runtimeLogsCmd)

	// reset-service-account
	runtimeCmd.AddCommand(runtimeResetServiceAccountCmd)

	// versions
	runtimeVersionsCmd.Flags().Int("page", 1, "Page number (1-based)")
	runtimeVersionsCmd.Flags().Int("size", 10, "Page size")
	runtimeCmd.AddCommand(runtimeVersionsCmd)

	// --- sub-resources: tracing ---

	runtimeTraceGetCmd.Flags().StringArrayVar(&runtimeTraceParams, "param", nil, "Passthrough query param key=value (repeatable)")
	runtimeTraceCmd.AddCommand(runtimeTraceGetCmd)

	runtimeTraceSearchCmd.Flags().StringArrayVar(&runtimeTraceParams, "param", nil, "Passthrough query param key=value (repeatable)")
	runtimeTraceCmd.AddCommand(runtimeTraceSearchCmd)

	runtimeTraceTagValuesCmd.Flags().String("tag-key", "", "Tag key (required)")
	runtimeTraceTagValuesCmd.Flags().StringArrayVar(&runtimeTraceParams, "param", nil, "Passthrough query param key=value (repeatable)")
	runtimeTraceCmd.AddCommand(runtimeTraceTagValuesCmd)

	runtimeCmd.AddCommand(runtimeTraceCmd)
}

// addLogFlags registers the shared log-search flags (from/limit/from-timestamp/
// to-timestamp/query/order) on a logs command.
func addLogFlags(cmd *cobra.Command) {
	cmd.Flags().Int("from", 0, "Offset (max 5000)")
	cmd.Flags().Int("limit", 100, "Max lines (max 500)")
	cmd.Flags().String("from-timestamp", "", "fromTimestamp (RFC3339)")
	cmd.Flags().String("to-timestamp", "", "toTimestamp (RFC3339)")
	cmd.Flags().String("query", "", "Log query filter")
	cmd.Flags().String("order", "", "Sort order")
}

// addRuntimeSpecFlags registers the mutable spec flags shared by create and
// update (everything except --name). autoscaling has flag defaults so it is
// always populated without prompting.
func addRuntimeSpecFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("description", "", "Description")
	f.String("image-url", "", "Container image URL (required)")
	f.Bool("image-auth-enabled", false, "Enable private-registry auth (requires --image-auth-username/password)")
	f.String("image-auth-username", "", "Private-registry username (with --image-auth-enabled)")
	f.String("image-auth-password", "", "Private-registry password (with --image-auth-enabled; prefer --file)")
	f.StringArray("command", nil, "Container entrypoint element (repeatable)")
	f.StringArray("args", nil, "Container arg (repeatable)")
	f.StringArray("env", nil, "Environment variable KEY=VALUE (repeatable)")
	f.String("flavor-id", "", "Flavor id (required)")
	f.Int("min-replicas", 1, "Autoscaling: minimum replicas (1-10)")
	f.Int("max-replicas", 2, "Autoscaling: maximum replicas (1-10)")
	f.Int("cpu-utilization", 70, "Autoscaling: CPU scale-up threshold % (10-90)")
	f.Int("memory-utilization", 70, "Autoscaling: memory scale-up threshold % (10-90)")
}

// runtimeName holds the --name value for create.
var runtimeName string
