package agentbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	abclient "github.com/greennodehub/greennode-cli/internal/agentbase/client"
	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	crpkg "github.com/greennodehub/greennode-cli/internal/agentbase/cr"
	deploypkg "github.com/greennodehub/greennode-cli/internal/agentbase/deploy"
	identitypkg "github.com/greennodehub/greennode-cli/internal/agentbase/identity"
	"github.com/greennodehub/greennode-cli/internal/agentbase/jsonslice"
	memorypkg "github.com/greennodehub/greennode-cli/internal/agentbase/memory"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
	runtimepkg "github.com/greennodehub/greennode-cli/internal/agentbase/runtime"
	coreclient "github.com/greennodehub/greennode-cli/internal/client"
	coreconfig "github.com/greennodehub/greennode-cli/internal/config"
)

// deployCmd groups the agent-lifecycle orchestrator. deploy has NO backend of
// its own — it composes the identity + memory + runtime (+ cr) clients. An agent
// is the set of resources sharing a name (the join key); there is no
// cross-service FK. The agent code is a container image, typically pushed to the
// user's vCR repo; `imageAuth: auto` resolves its pull credentials from cr.
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy and manage an agent across identity, memory, and runtime",
	Long: `Deploy and manage an agent as one unit (a composite over the identity,
memory, and runtime services — deploy has no backend of its own).

An "agent" is the set of resources that share a name: an identity (always), an
optional memory container (stateless agents omit it), and a runtime (the
container that runs the agent code). The agent code is a container image; push it
to your vCR repo and reference it in the manifest. ` + "`imageAuth: auto`" + `
resolves the pull credentials from your auto-provisioned robot account.

    grn agentbase deploy generate > agent.yaml
    # ...edit agent.yaml...
    grn agentbase deploy up --file agent.yaml
    grn agentbase deploy status my-agent
    grn agentbase deploy destroy my-agent

` + "`up`" + ` is idempotent (create-if-absent per service, then converge the
runtime to ACTIVE). ` + "`destroy`" + ` deletes runtime + memory; pass --purge to
also delete the identity. ` + "`up`" + `/` + "`destroy`" + ` look resources up by
name, so no state file is needed.`,
}

// ---------------------------------------------------------------------------
// shared clients (one token mint shared across all four typed clients)
// ---------------------------------------------------------------------------

// deployClients lazily builds the four underlying typed clients over a SINGLE
// shared token provider, so an `up` that touches identity+memory+runtime+cr
// mints one token, not four. Each client is constructed on first use.
type deployClients struct {
	ab       *agentbaseCtx
	provider coreclient.TokenProvider

	idc *identitypkg.Client
	mc  *memorypkg.Client
	rc  *runtimepkg.Client
	cc  *crpkg.Client
}

func newDeployClients(cmd *cobra.Command) (*deployClients, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return &deployClients{ab: ab, provider: provider}, nil
}

func (d *deployClients) identity() *identitypkg.Client {
	if d.idc == nil {
		d.idc = identitypkg.NewClient(d.ab.endpoints.Identity, d.provider)
	}
	return d.idc
}

func (d *deployClients) memory() *memorypkg.Client {
	if d.mc == nil {
		d.mc = memorypkg.NewClient(d.ab.endpoints.Memory, d.provider)
	}
	return d.mc
}

func (d *deployClients) runtime() *runtimepkg.Client {
	if d.rc == nil {
		d.rc = runtimepkg.NewClient(d.ab.endpoints.Runtime, d.provider)
	}
	return d.rc
}

func (d *deployClients) cr() *crpkg.Client {
	if d.cc == nil {
		d.cc = crpkg.NewClient(d.ab.endpoints.Cr, d.provider)
	}
	return d.cc
}

// deployStep is one row of a destroy outcome report (service + deleted/ not_found/ error + detail).
type deployStep struct {
	service string
	outcome string
	detail  string
}

// isNotFound reports whether err is an HTTP 404 from the shared client. The
// typed clients return *abclient.APIError on non-2xx, so a missing resource is
// distinguishable from a transport/auth failure.
func isNotFound(err error) bool {
	var apiErr *abclient.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// ---------------------------------------------------------------------------
// generate
// ---------------------------------------------------------------------------

var deployGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print an agent manifest template (YAML or JSON)",
	Long: `Print a commented agent manifest to stdout. Save it, fill it in, and apply
with 'grn agentbase deploy up --file <file>'.

Defaults to YAML (with comments); pass -o json for a JSON skeleton.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			example := map[string]any{
				"name":        "my-agent",
				"description": "A customer-support agent",
				"identity": map[string]any{
					"allowedReturnUrls": []string{"https://app.example.com/callback"},
				},
				"memory": map[string]any{
					"eventExpiryDuration": 3600,
					"strategies": []map[string]any{{
						"name":              "prefs",
						"type":              "USER_PREFERENCE",
						"namespaceTemplate": "/strategies/USER_PREFERENCE/actors/{actorId}",
					}},
				},
				"runtime": map[string]any{
					"image":     "registry.vngcloud.vn/<your-repo>/my-agent:v1",
					"imageAuth": "auto",
					"command":   []string{"./agent"},
					"args":      []string{"--port", "8080"},
					"env":       map[string]string{"LOG_LEVEL": "info"},
					"flavorId":  "agent.small",
					"autoscaling": map[string]any{
						"minReplicas": 1, "maxReplicas": 3,
						"cpuUtilization": 70, "memoryUtilization": 80,
					},
				},
			}
			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(deployManifestTemplateYAML)
		return nil
	},
}

// deployManifestTemplateYAML is a hand-written, commented manifest. Keys match
// the JSON struct tags so it round-trips through 'up --file' exactly. Note the
// short ergonomic keys (image/env/strategies) — the manifest is a deploy
// concept, not a passthrough of any one service's request body.
const deployManifestTemplateYAML = `# Agent manifest (deploy spec). Apply with:
#   grn agentbase deploy up --file <this-file>
#
# name is the shared join key across identity + memory + runtime (3-50 chars,
# ^[a-zA-Z0-9_-]+$). identity is always created. memory is OPTIONAL — delete the
# whole block for a stateless agent. runtime runs the agent code as a container.
name: my-agent
description: "A customer-support agent"

identity:
  allowedReturnUrls:
    - https://app.example.com/callback

# memory: OPTIONAL. Omit the whole block for a stateless agent. When present,
# at least one strategy (name/type/namespaceTemplate) is required.
memory:
  eventExpiryDuration: 3600
  strategies:
    - name: prefs
      type: USER_PREFERENCE                      # built-in key (USER_PREFERENCE|SEMANTIC|CUSTOM|...)
      namespaceTemplate: "/strategies/USER_PREFERENCE/actors/{actorId}"

runtime:
  image: registry.vngcloud.vn/<your-repo>/my-agent:v1
  imageAuth: auto                                # "auto" resolves pull creds from your vCR robot
                                                 # account; or {username: ..., password: ...}
  command: [./agent]
  args: [--port, "8080"]
  env: {LOG_LEVEL: info}
  flavorId: agent.small
  autoscaling: {minReplicas: 1, maxReplicas: 3, cpuUtilization: 70, memoryUtilization: 80}
`

// ---------------------------------------------------------------------------
// up — idempotent apply + converge
// ---------------------------------------------------------------------------

var deployFile string

var deployUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply an agent (create-if-absent across services) and converge",
	Long: `Apply an agent manifest: create the identity, optional memory, and runtime if
absent (each looked up by name), then converge the runtime to ACTIVE. Existing
resources are left as-is (memory has no update; runtime is not re-applied) —
re-run is safe and idempotent.

imageAuth: auto in the manifest resolves the runtime's private-registry pull
credentials from your vCR robot account (wiring cr into deploy).

Use --no-wait to return as soon as the runtime is submitted (state CREATING)
without converging.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		mfst, err := resolveManifest(cmd)
		if err != nil {
			return err
		}
		d, err := newDeployClients(cmd)
		if err != nil {
			return err
		}
		noWait, _ := cmd.Flags().GetBool("no-wait")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		setCurrent, _ := cmd.Flags().GetBool("set-current")

		// 1. identity (always).
		identID, _, err := deployApplyIdentity(ctx, cmd, d, mfst, setCurrent)
		if err != nil {
			return err
		}

		// 2. memory (optional). A failure here does not unwind identity — `up`
		//    is fire-and-report; re-run to retry, or `destroy` to tear down.
		memID, memSkipped, err := deployApplyMemory(ctx, d, mfst)
		if err != nil {
			return fmt.Errorf("identity applied (%s); memory failed: %w\nre-run 'up' (idempotent) or 'destroy %s' to tear down", identID, err, mfst.Name)
		}

		// 3. runtime (created or converged).
		rt, err := deployApplyRuntime(ctx, d, mfst, noWait, timeout, interval)
		if err != nil {
			return fmt.Errorf("identity (%s) + memory (%s) applied; runtime failed: %w\nre-run 'up' (idempotent) or 'destroy %s' to tear down", identID, memoryState(memID, memSkipped), err, mfst.Name)
		}
		return renderDeployRollup(mfst.Name, identID, memID, memSkipped, rt)
	},
}

// deployApplyIdentity creates the identity if absent. Returns (id, created).
func deployApplyIdentity(ctx context.Context, cmd *cobra.Command, d *deployClients, mfst *deploypkg.Manifest, setCurrent bool) (string, bool, error) {
	idc := d.identity()
	if existing, err := idc.GetAgentIdentity(ctx, mfst.Name); err == nil {
		fmt.Fprintf(os.Stderr, "Identity %q exists (id %s) — skipping.\n", mfst.Name, str(existing.ID))
		return str(existing.ID), false, nil
	} else if !isNotFound(err) {
		return "", false, fmt.Errorf("identity lookup: %w", err)
	}
	req := &identitypkg.CreateAgentIdentityRequest{Name: mfst.Name}
	if len(mfst.Identity.AllowedReturnURLs) > 0 {
		req.AllowedReturnURLs = jsonslice.Array[string](mfst.Identity.AllowedReturnURLs)
	}
	created, err := idc.CreateAgentIdentity(ctx, req)
	if err != nil {
		return "", false, fmt.Errorf("identity create: %w", err)
	}
	if setCurrent {
		if err := coreconfig.NewConfigFileWriter().WriteAgentIdentity(resolveProfile(cmd), mfst.Name); err != nil {
			output.Warn("Identity created but failed to save as current: " + err.Error())
		}
	}
	fmt.Fprintf(os.Stderr, "Identity %q created (id %s).\n", mfst.Name, str(created.ID))
	return str(created.ID), true, nil
}

// deployApplyMemory creates the memory container if the block is present and no
// memory of this name exists. Returns (id, skipped). skipped=true means the
// manifest had no memory block (stateless agent). An existing memory is skipped
// (memory has no update).
func deployApplyMemory(ctx context.Context, d *deployClients, mfst *deploypkg.Manifest) (string, bool, error) {
	if mfst.Memory == nil {
		fmt.Fprintln(os.Stderr, "No memory block — stateless agent.")
		return "", true, nil
	}
	mc := d.memory()
	if existing, err := findMemoryByName(ctx, mc, mfst.Name); err != nil {
		return "", false, fmt.Errorf("memory lookup: %w", err)
	} else if existing != nil {
		fmt.Fprintf(os.Stderr, "Memory %q exists (id %s) — skipping (memory has no update).\n", mfst.Name, existing.ID)
		return existing.ID, false, nil
	}
	mem, err := mc.Create(ctx, buildMemoryCreateReq(mfst))
	if err != nil {
		return "", false, fmt.Errorf("memory create: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Memory %q created (id %s).\n", mfst.Name, mem.ID)
	return mem.ID, false, nil
}

// deployApplyRuntime creates the runtime if absent (resolving imageAuth) and
// converges it to ACTIVE unless noWait. An existing runtime is converged (not
// re-applied — update is a full-spec replacement the manifest may not carry).
func deployApplyRuntime(ctx context.Context, d *deployClients, mfst *deploypkg.Manifest, noWait bool, timeout, interval time.Duration) (*runtimepkg.AgentRuntime, error) {
	rc := d.runtime()
	rt, err := findRuntimeByName(ctx, rc, mfst.Name)
	if err != nil {
		return nil, fmt.Errorf("runtime lookup: %w", err)
	}
	if rt == nil {
		req, err := buildRuntimeCreateReq(ctx, d, mfst)
		if err != nil {
			return nil, err
		}
		rt, err = rc.Create(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("runtime create: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Runtime %q submitted (id %s, state %s).\n", mfst.Name, rt.ID, rt.Status)
	} else {
		fmt.Fprintf(os.Stderr, "Runtime %q exists (id %s) — converging, not re-applying.\n", mfst.Name, rt.ID)
	}
	if runtimeTerminalSuccess[rt.Status] {
		return rt, nil
	}
	if noWait {
		fmt.Fprintf(os.Stderr, "Runtime not converged (state %s); skipping wait due to --no-wait.\n", rt.Status)
		return rt, nil
	}
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rt, err = pollRuntimeToTerminal(pctx, rc, rt.ID, interval)
	if err != nil {
		return rt, err
	}
	return rt, nil
}

// buildMemoryCreateReq maps the manifest memory block onto the memory service's
// create request.
func buildMemoryCreateReq(mfst *deploypkg.Manifest) *memorypkg.CreateMemoryRequest {
	req := &memorypkg.CreateMemoryRequest{
		Name:                mfst.Name,
		Description:         mfst.Description,
		EventExpiryDuration: mfst.Memory.EventExpiryDuration,
	}
	for _, s := range mfst.Memory.LongTermMemoryStrategies {
		req.LongTermMemoryStrategies = append(req.LongTermMemoryStrategies, memorypkg.LongTermMemoryStrategy{
			Name:                                  s.Name,
			Type:                                  s.Type,
			NamespaceTemplate:                     s.NamespaceTemplate,
			CustomFactExtractionPrompt:            s.CustomFactExtractionPrompt,
			EnableAutomaticMemoryRecordGeneration: s.EnableAutomaticMemoryRecordGeneration,
		})
	}
	return req
}

// buildRuntimeCreateReq maps the manifest runtime block onto the runtime
// service's create request, resolving imageAuth. imageAuth:auto calls cr for
// the robot account; explicit {username,password} is used verbatim; absent
// imageAuth yields no private-registry auth (public image). Nil slices/maps are
// normalized to empty because the runtime's @NotNull fields reject null.
func buildRuntimeCreateReq(ctx context.Context, d *deployClients, mfst *deploypkg.Manifest) (*runtimepkg.CreateAgentRuntimeRequest, error) {
	rs := mfst.Runtime
	var imageAuth *runtimepkg.ImageAuth
	if rs.ImageAuth.Auto {
		cred, err := d.cr().GetRegistryCredential(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve imageAuth:auto from cr: %w", err)
		}
		imageAuth = &runtimepkg.ImageAuth{Enabled: true, Username: cred.Username, Password: cred.Secret}
		fmt.Fprintf(os.Stderr, "imageAuth:auto resolved from cr robot account %q.\n", cred.Username)
	} else if rs.ImageAuth.Username != "" || rs.ImageAuth.Password != "" {
		imageAuth = &runtimepkg.ImageAuth{Enabled: true, Username: rs.ImageAuth.Username, Password: rs.ImageAuth.Password}
	}
	return &runtimepkg.CreateAgentRuntimeRequest{
		Name:                 mfst.Name,
		Description:          mfst.Description,
		ImageURL:             rs.Image,
		ImageAuth:            imageAuth,
		Command:              nonNilStrings(rs.Command),
		Args:                 nonNilStrings(rs.Args),
		EnvironmentVariables: nonNilMap(rs.Env),
		FlavorID:             rs.FlavorID,
		Autoscaling: runtimepkg.Autoscaling{
			MinReplicas:       rs.Autoscaling.MinReplicas,
			MaxReplicas:       rs.Autoscaling.MaxReplicas,
			CPUUtilization:    rs.Autoscaling.CPUUtilization,
			MemoryUtilization: rs.Autoscaling.MemoryUtilization,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// status — cross-service rollup by name
// ---------------------------------------------------------------------------

var deployStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show the cross-service state of an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]
		d, err := newDeployClients(cmd)
		if err != nil {
			return err
		}

		type svcRow struct {
			service string
			state   string // present | absent | error
			id      string
			status  string
		}
		var rows []svcRow

		// identity (by-name get).
		if ident, err := d.identity().GetAgentIdentity(ctx, name); err == nil {
			rows = append(rows, svcRow{"identity", "present", str(ident.ID), ""})
		} else if isNotFound(err) {
			rows = append(rows, svcRow{"identity", "absent", "", ""})
		} else {
			rows = append(rows, svcRow{"identity", "error", "", err.Error()})
		}

		// memory (client-side filter).
		if mem, err := findMemoryByName(ctx, d.memory(), name); err == nil && mem != nil {
			rows = append(rows, svcRow{"memory", "present", mem.ID, mem.Status})
		} else if err == nil {
			rows = append(rows, svcRow{"memory", "absent", "", ""})
		} else {
			rows = append(rows, svcRow{"memory", "error", "", err.Error()})
		}

		// runtime (client-side filter; carries the async Status).
		if rt, err := findRuntimeByName(ctx, d.runtime(), name); err == nil && rt != nil {
			rows = append(rows, svcRow{"runtime", "present", rt.ID, rt.Status})
		} else if err == nil {
			rows = append(rows, svcRow{"runtime", "absent", "", ""})
		} else {
			rows = append(rows, svcRow{"runtime", "error", "", err.Error()})
		}

		switch output.GetFormat() {
		case output.FormatJSON:
			out := map[string]any{"name": name, "services": rows}
			return output.JSON(out)
		case output.FormatID:
			for _, r := range rows {
				if r.id != "" {
					output.PrintID(r.id)
				}
			}
			return nil
		}
		table := make([][]string, 0, len(rows))
		for _, r := range rows {
			table = append(table, []string{r.service, r.state, output.StrOrDash(r.id), output.StrOrDash(r.status)})
		}
		fmt.Fprintf(os.Stderr, "Agent %q\n", name)
		output.Table([]string{"Service", "State", "ID", "Status"}, table)
		return nil
	},
}

// ---------------------------------------------------------------------------
// destroy — best-effort reverse teardown
// ---------------------------------------------------------------------------

var deployDestroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Delete an agent's runtime and memory (and identity with --purge)",
	Long: `Tear down an agent by name (best-effort, reverse of apply):

  1. runtime — delete by id, wait for DELETED.
  2. memory  — soft-delete by id (ACTIVE -> DELETED).
  3. identity — only with --purge (it may be referenced by other agents).

Missing sub-resources are reported and skipped; each step's outcome is shown.
The runtime deletion is asynchronous, so destroy waits for it to reach DELETED
(use --timeout to bound the wait).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]
		d, err := newDeployClients(cmd)
		if err != nil {
			return err
		}
		purge, _ := cmd.Flags().GetBool("purge")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")

		var steps []deployStep

		// 1. runtime.
		if rt, lerr := findRuntimeByName(ctx, d.runtime(), name); lerr != nil {
			steps = append(steps, deployStep{"runtime", "error", lerr.Error()})
		} else if rt == nil {
			steps = append(steps, deployStep{"runtime", "not_found", ""})
		} else if derr := d.runtime().Delete(ctx, rt.ID); derr != nil {
			steps = append(steps, deployStep{"runtime", "error", derr.Error()})
		} else {
			pctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			if _, werr := pollRuntimeToTerminal(pctx, d.runtime(), rt.ID, interval); werr != nil {
				steps = append(steps, deployStep{"runtime", "error", "deleted but " + werr.Error()})
			} else {
				steps = append(steps, deployStep{"runtime", "deleted", rt.ID})
			}
		}

		// 2. memory (soft-delete; synchronous).
		if mem, lerr := findMemoryByName(ctx, d.memory(), name); lerr != nil {
			steps = append(steps, deployStep{"memory", "error", lerr.Error()})
		} else if mem == nil {
			steps = append(steps, deployStep{"memory", "not_found", ""})
		} else if derr := d.memory().Delete(ctx, mem.ID); derr != nil {
			steps = append(steps, deployStep{"memory", "error", derr.Error()})
		} else {
			steps = append(steps, deployStep{"memory", "deleted", mem.ID})
		}

		// 3. identity (only with --purge).
		if purge {
			if derr := d.identity().DeleteAgentIdentity(ctx, name); derr != nil {
				if isNotFound(derr) {
					steps = append(steps, deployStep{"identity", "not_found", ""})
				} else {
					steps = append(steps, deployStep{"identity", "error", derr.Error()})
				}
			} else {
				steps = append(steps, deployStep{"identity", "deleted", name})
			}
		}

		return renderDestroySteps(name, steps)
	},
}

// ---------------------------------------------------------------------------
// name-based lookups (memory + runtime have no by-name get, so list + filter)
// ---------------------------------------------------------------------------

const deployLookupPageSize = 100

// findMemoryByName pages through the memory list (client-side name filter).
// Returns nil (no error) when no memory of that name exists.
func findMemoryByName(ctx context.Context, mc *memorypkg.Client, name string) (*memorypkg.Memory, error) {
	page := 1
	for {
		resp, err := mc.List(ctx, page, deployLookupPageSize)
		if err != nil {
			return nil, err
		}
		for i := range resp.ListData {
			if resp.ListData[i].Name == name {
				return &resp.ListData[i], nil
			}
		}
		if page >= int(resp.TotalPage) || len(resp.ListData) == 0 {
			return nil, nil
		}
		page++
	}
}

// findRuntimeByName pages through the runtime list (client-side name filter).
// Returns nil (no error) when no runtime of that name exists.
func findRuntimeByName(ctx context.Context, rc *runtimepkg.Client, name string) (*runtimepkg.AgentRuntime, error) {
	page := 1
	for {
		resp, err := rc.List(ctx, page, deployLookupPageSize)
		if err != nil {
			return nil, err
		}
		for i := range resp.ListData {
			if resp.ListData[i].Name == name {
				return &resp.ListData[i], nil
			}
		}
		if page >= int(resp.TotalPage) || len(resp.ListData) == 0 {
			return nil, nil
		}
		page++
	}
}

// ---------------------------------------------------------------------------
// manifest resolution (--file or flags) + spec loader
// ---------------------------------------------------------------------------

// resolveManifest builds the manifest from --file (authoritative) or flags.
func resolveManifest(cmd *cobra.Command) (*deploypkg.Manifest, error) {
	if deployFile != "" {
		data, err := os.ReadFile(deployFile)
		if err != nil {
			return nil, fmt.Errorf("read --file: %w", err)
		}
		return loadDeployManifest(data)
	}
	return buildManifestFromFlags(cmd)
}

// buildManifestFromFlags builds a minimal manifest for the simple path. Memory
// is opt-in via repeatable --memory-strategy <TYPE> (one strategy per flag, with
// a default namespaceTemplate). imageAuth from flags is "auto" only; explicit
// username/password requires --file.
func buildManifestFromFlags(cmd *cobra.Command) (*deploypkg.Manifest, error) {
	f := cmd.Flags()
	name, _ := f.GetString("name")
	name, err := cliinput.RequireOrPromptString(name, "--name", "Agent name (the shared join key)")
	if err != nil {
		return nil, err
	}
	desc, _ := f.GetString("description")
	image, _ := f.GetString("image")
	image, err = cliinput.RequireOrPromptString(image, "--image", "Container image URL")
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
	mfst := &deploypkg.Manifest{
		Name:        name,
		Description: desc,
		Runtime: deploypkg.RuntimeSpec{
			Image:    image,
			FlavorID: flavorID,
			Command:  command,
			Args:     args,
		},
	}
	if envRaw, _ := f.GetStringArray("env"); len(envRaw) > 0 {
		env, err := parseEnvVars(envRaw)
		if err != nil {
			return nil, err
		}
		mfst.Runtime.Env = env
	}
	if ia, _ := f.GetString("image-auth"); ia != "" {
		if ia != "auto" {
			return nil, fmt.Errorf("--image-auth must be %q (explicit username/password via --file only)", "auto")
		}
		mfst.Runtime.ImageAuth = deploypkg.ImageAuthSpec{Auto: true}
	}
	minReplicas, _ := f.GetInt("min-replicas")
	maxReplicas, _ := f.GetInt("max-replicas")
	cpuUtil, _ := f.GetInt("cpu-utilization")
	memUtil, _ := f.GetInt("memory-utilization")
	mfst.Runtime.Autoscaling = deploypkg.ManifestAutoscale{
		MinReplicas:       minReplicas,
		MaxReplicas:       maxReplicas,
		CPUUtilization:    cpuUtil,
		MemoryUtilization: memUtil,
	}
	if mfst.Runtime.Autoscaling.MaxReplicas < mfst.Runtime.Autoscaling.MinReplicas {
		return nil, fmt.Errorf("--max-replicas (%d) must be >= --min-replicas (%d)",
			mfst.Runtime.Autoscaling.MaxReplicas, mfst.Runtime.Autoscaling.MinReplicas)
	}
	if strategies, _ := f.GetStringArray("memory-strategy"); len(strategies) > 0 {
		mem := &deploypkg.MemorySpec{}
		for _, t := range strategies {
			mem.LongTermMemoryStrategies = append(mem.LongTermMemoryStrategies, deploypkg.ManifestMemoryStrategy{
				Name:              strings.ToLower(t),
				Type:              t,
				NamespaceTemplate: fmt.Sprintf("/strategies/%s/actors/{actorId}", t),
			})
		}
		mfst.Memory = mem
	}
	return mfst, nil
}

// intFromFlag helper removed — autoscaling flags are read directly via GetInt.

// loadDeployManifest decodes a YAML/JSON manifest via the map bridge and
// validates the required fields.
func loadDeployManifest(data []byte) (*deploypkg.Manifest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var mfst deploypkg.Manifest
	if err := json.Unmarshal(jb, &mfst); err != nil {
		return nil, fmt.Errorf("invalid deploy manifest: %w", err)
	}
	if mfst.Name == "" {
		return nil, fmt.Errorf("manifest is missing required field: name")
	}
	if mfst.Runtime.Image == "" {
		return nil, fmt.Errorf("manifest.runtime is missing required field: image")
	}
	if mfst.Runtime.FlavorID == "" {
		return nil, fmt.Errorf("manifest.runtime is missing required field: flavorId")
	}
	if mfst.Memory != nil && len(mfst.Memory.LongTermMemoryStrategies) == 0 {
		return nil, fmt.Errorf("manifest.memory present but has no strategies (at least one required)")
	}
	return &mfst, nil
}

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderDeployRollup(name, identID, memID string, memSkipped bool, rt *runtimepkg.AgentRuntime) error {
	rollup := map[string]any{
		"name":     name,
		"identity": map[string]string{"id": identID},
		"runtime":  map[string]string{"id": rt.ID, "status": rt.Status},
		"memory":   memoryRollup(memID, memSkipped),
	}
	switch output.GetFormat() {
	case output.FormatJSON:
		return output.JSON(rollup)
	case output.FormatID:
		output.PrintID(rt.ID)
		return nil
	}
	rows := [][]string{
		{"identity", "present", identID, ""},
		{"memory", memoryState(memID, memSkipped), output.StrOrDash(memID), ""},
		{"runtime", "present", rt.ID, rt.Status},
	}
	fmt.Fprintf(os.Stderr, "Agent %q applied.\n", name)
	output.Table([]string{"Service", "State", "ID", "Status"}, rows)
	return nil
}

func renderDestroySteps(name string, steps []deployStep) error {
	switch output.GetFormat() {
	case output.FormatJSON:
		return output.JSON(map[string]any{"name": name, "steps": steps})
	case output.FormatID:
		for _, s := range steps {
			if s.outcome == "deleted" && s.detail != "" {
				output.PrintID(s.detail)
			}
		}
		return nil
	}
	rows := make([][]string, 0, len(steps))
	for _, s := range steps {
		rows = append(rows, []string{s.service, s.outcome, output.StrOrDash(s.detail)})
	}
	fmt.Fprintf(os.Stderr, "Agent %q destroyed.\n", name)
	output.Table([]string{"Service", "Outcome", "ID"}, rows)
	return nil
}

// memoryState renders the memory row state for the rollup table.
func memoryState(memID string, skipped bool) string {
	if skipped {
		return "skipped (stateless)"
	}
	if memID != "" {
		return "present"
	}
	return "absent"
}

func memoryRollup(memID string, skipped bool) map[string]string {
	if skipped {
		return map[string]string{"state": "skipped"}
	}
	return map[string]string{"id": memID}
}

// nonNilStrings returns s if non-nil, else an empty slice (the runtime's
// @NotNull fields reject JSON null).
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// nonNilMap returns m if non-nil, else an empty map (same @NotNull reason).
func nonNilMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(deployCmd)

	deployCmd.AddCommand(deployGenerateCmd)

	up := deployUpCmd
	up.Flags().StringVar(&deployFile, "file", "", "Apply a manifest file (see 'deploy generate')")
	up.Flags().String("name", "", "Agent name (the shared join key; required without --file)")
	up.Flags().String("description", "", "Description")
	up.Flags().String("image", "", "Container image URL (required without --file)")
	up.Flags().String("flavor-id", "", "Flavor id (required without --file)")
	up.Flags().String("image-auth", "", "Private-registry auth: 'auto' (resolve from cr); explicit via --file")
	up.Flags().StringArray("command", nil, "Container entrypoint element (repeatable)")
	up.Flags().StringArray("args", nil, "Container arg (repeatable)")
	up.Flags().StringArray("env", nil, "Environment variable KEY=VALUE (repeatable)")
	up.Flags().Int("min-replicas", 1, "Autoscaling min replicas")
	up.Flags().Int("max-replicas", 2, "Autoscaling max replicas")
	up.Flags().Int("cpu-utilization", 70, "Autoscaling target CPU utilization (10-90)")
	up.Flags().Int("memory-utilization", 70, "Autoscaling target memory utilization (10-90)")
	up.Flags().StringArray("memory-strategy", nil, "Memory strategy TYPE (repeatable; e.g. USER_PREFERENCE). Adds a memory container")
	up.Flags().Bool("set-current", false, "Set the created identity as the current agent in the profile")
	up.Flags().Bool("no-wait", false, "Return as soon as the runtime is submitted (do not converge to ACTIVE)")
	up.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait for the runtime to converge")
	up.Flags().Duration("interval", 5*time.Second, "Poll interval while converging")
	deployCmd.AddCommand(up)

	status := deployStatusCmd
	status.Flags().Duration("timeout", 10*time.Second, "Per-service lookup timeout (reserved)")
	deployCmd.AddCommand(status)

	destroy := deployDestroyCmd
	destroy.Flags().Bool("purge", false, "Also delete the identity (default: leave it)")
	destroy.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait for the runtime to delete")
	destroy.Flags().Duration("interval", 5*time.Second, "Poll interval while deleting")
	deployCmd.AddCommand(destroy)
}
