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
// /v1/agent-runtimes). An agent runtime is a deployable container
// (image/command/args/env/network/inbound-auth/autoscaling); it converges
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

The runtime API addresses resources by id (not name; name is immutable). Create
is fully flag-driven (see 'create --help'); update carries fields whose flags
are unset over from the current version (see 'update --help'). For a full spec
from a file — or to keep the image-auth password out of the shell — generate a
template, fill it in, and apply with --file:

    grn agentbase runtime generate > rt.yaml
    # ...edit rt.yaml...
    grn agentbase runtime create --file rt.yaml
    grn agentbase runtime wait <id>

Runtimes share the ~/.greennode profile like the rest of agentbase.`,
	RunE: requireSubcommand,
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
// networkConfig/inboundAuth/protocol/allowedCidrs are only populated when the
// corresponding flags are set (they are optional server-side; omitting them
// keeps the platform defaults: public network, no inbound auth, HTTP).
type runtimeSpec struct {
	Description  string
	ImageURL     string
	ImageAuth    *runtimepkg.ImageAuth
	Command      []string
	Args         []string
	Env          map[string]string
	Network      *runtimepkg.NetworkConfig
	AllowedCidrs []string
	Inbound      *runtimepkg.InboundAuth
	Protocol     string
	FlavorID     string
	Autoscaling  runtimepkg.Autoscaling
}

// readRuntimeSpec reads + validates the mutable spec fields. name and poc are
// create-only and handled by the caller. On create (forUpdate=false) the
// @NotEmpty fields imageUrl/flavorId are prompted when unset; autoscaling has
// flag defaults so it is always populated. On update (forUpdate=true) unset
// flags are left empty — no prompting — and carried over from the current
// version by buildUpdateRequest; the autoscaling bounds are only checked when
// a scaling flag is actually set (the rest carries over).
func readRuntimeSpec(cmd *cobra.Command, forUpdate bool) (*runtimeSpec, error) {
	f := cmd.Flags()

	description, _ := f.GetString("description")

	imageURL, _ := f.GetString("image-url")
	if forUpdate {
		// Explicitly-set-but-empty is a user error; unset carries over.
		if f.Changed("image-url") && imageURL == "" {
			return nil, fmt.Errorf("--image-url cannot be empty")
		}
	} else {
		var err error
		if imageURL, err = cliinput.RequireOrPromptString(imageURL, "--image-url", "Container image URL"); err != nil {
			return nil, err
		}
	}

	flavorID, _ := f.GetString("flavor-id")
	if forUpdate {
		if f.Changed("flavor-id") && flavorID == "" {
			return nil, fmt.Errorf("--flavor-id cannot be empty")
		}
	} else {
		var err error
		if flavorID, err = cliinput.RequireOrPromptString(flavorID, "--flavor-id", "Flavor id"); err != nil {
			return nil, err
		}
	}

	command, _ := f.GetStringArray("command")
	args, _ := f.GetStringArray("args")

	envRaw, _ := f.GetStringArray("env")
	env, err := parseEnvVars(envRaw)
	if err != nil {
		return nil, err
	}

	// minReplicas was removed from the API — the platform owns the floor (1).
	// The flag stays registered (hidden, deprecated) so old scripts only warn.
	if f.Changed("min-replicas") {
		fmt.Fprintln(os.Stderr, "warning: --min-replicas is deprecated and ignored; the platform owns the replica floor (always 1)")
	}
	maxReplicas, _ := f.GetInt("max-replicas")
	cpuUtil, _ := f.GetInt("cpu-utilization")
	memUtil, _ := f.GetInt("memory-utilization")
	autoscaling := runtimepkg.Autoscaling{
		MaxReplicas:       maxReplicas,
		CPUUtilization:    cpuUtil,
		MemoryUtilization: memUtil,
	}
	if autoscaling.MaxReplicas < 1 && (!forUpdate || f.Changed("max-replicas") || f.Changed("cpu-utilization") || f.Changed("memory-utilization")) {
		return nil, fmt.Errorf("--max-replicas (%d) must be >= 1", autoscaling.MaxReplicas)
	}

	network, err := readNetworkConfig(cmd)
	if err != nil {
		return nil, err
	}
	allowedCidrs, _ := f.GetStringArray("allowed-cidrs")
	inbound, err := readInboundAuth(cmd)
	if err != nil {
		return nil, err
	}
	protocol, _ := f.GetString("protocol")
	if protocol != "" && protocol != "HTTP" && protocol != "MCP" {
		return nil, fmt.Errorf("--protocol must be HTTP or MCP (got %q)", protocol)
	}

	imageAuth, err := readImageAuth(cmd)
	if err != nil {
		return nil, err
	}

	return &runtimeSpec{
		Description:  description,
		ImageURL:     imageURL,
		ImageAuth:    imageAuth,
		Command:      command,
		Args:         args,
		Env:          env,
		Network:      network,
		AllowedCidrs: allowedCidrs,
		Inbound:      inbound,
		Protocol:     protocol,
		FlavorID:     flavorID,
		Autoscaling:  autoscaling,
	}, nil
}

// readNetworkConfig builds *NetworkConfig only when --network-mode is set.
// PUBLIC is the platform default (endpoints on the public internet); VPC
// attaches endpoints to a VPC/subnet (vpcId and subnetId are then required;
// routeCidrs adds egress routes).
func readNetworkConfig(cmd *cobra.Command) (*runtimepkg.NetworkConfig, error) {
	f := cmd.Flags()
	mode, _ := f.GetString("network-mode")
	if mode == "" {
		// A dangling --vpc-id/--subnet-id/--route-cidrs without --network-mode
		// used to be silently ignored — refuse it instead (it would otherwise
		// read as "leave the network public" while the user meant VPC).
		for _, name := range []string{"vpc-id", "subnet-id", "route-cidrs"} {
			if f.Changed(name) {
				return nil, fmt.Errorf("--%s only applies with --network-mode (PUBLIC or VPC)", name)
			}
		}
		return nil, nil
	}
	if mode != "PUBLIC" && mode != "VPC" {
		return nil, fmt.Errorf("--network-mode must be PUBLIC or VPC (got %q)", mode)
	}
	vpcID, _ := f.GetString("vpc-id")
	subnetID, _ := f.GetString("subnet-id")
	routeCidrs, _ := f.GetStringArray("route-cidrs")
	if mode == "VPC" && (vpcID == "" || subnetID == "") {
		return nil, fmt.Errorf("--network-mode VPC requires both --vpc-id and --subnet-id")
	}
	return &runtimepkg.NetworkConfig{
		Mode:       mode,
		VpcID:      vpcID,
		SubnetID:   subnetID,
		RouteCidrs: routeCidrs,
	}, nil
}

// readInboundAuth builds *InboundAuth only when --inbound-mode is set. NONE is
// open, IAM authenticates callers via the platform IAM, JWT validates caller
// JWTs (--inbound-jwt-* then required: DISCOVERY needs --inbound-jwt-discovery-url,
// JWKS needs --inbound-jwt-jwks).
func readInboundAuth(cmd *cobra.Command) (*runtimepkg.InboundAuth, error) {
	f := cmd.Flags()
	mode, _ := f.GetString("inbound-mode")
	if mode == "" {
		return nil, nil
	}
	if mode != "NONE" && mode != "IAM" && mode != "JWT" {
		return nil, fmt.Errorf("--inbound-mode must be NONE, IAM or JWT (got %q)", mode)
	}
	if mode != "JWT" {
		for _, name := range []string{"inbound-jwt-source", "inbound-jwt-jwks", "inbound-jwt-discovery-url", "inbound-jwt-audience", "inbound-jwt-client", "inbound-jwt-scope", "inbound-jwt-principal-claim"} {
			if f.Changed(name) {
				return nil, fmt.Errorf("--%s only applies when --inbound-mode is JWT", name)
			}
		}
		return &runtimepkg.InboundAuth{Mode: mode}, nil
	}
	source, _ := f.GetString("inbound-jwt-source")
	if source == "" {
		return nil, fmt.Errorf("--inbound-mode JWT requires --inbound-jwt-source (DISCOVERY or JWKS)")
	}
	if source != "DISCOVERY" && source != "JWKS" {
		return nil, fmt.Errorf("--inbound-jwt-source must be DISCOVERY or JWKS (got %q)", source)
	}
	jwksRaw, _ := f.GetString("inbound-jwt-jwks")
	discoveryURL, _ := f.GetString("inbound-jwt-discovery-url")
	switch source {
	case "DISCOVERY":
		if discoveryURL == "" {
			return nil, fmt.Errorf("--inbound-jwt-source DISCOVERY requires --inbound-jwt-discovery-url")
		}
	case "JWKS":
		if jwksRaw == "" {
			return nil, fmt.Errorf("--inbound-jwt-source JWKS requires --inbound-jwt-jwks (the key set JSON, max 32KiB)")
		}
		if !json.Valid([]byte(jwksRaw)) {
			return nil, fmt.Errorf("--inbound-jwt-jwks must be a valid JSON key set")
		}
		if len(jwksRaw) > 32*1024 {
			return nil, fmt.Errorf("--inbound-jwt-jwks is %d bytes; the API caps it at 32KiB (prefer DISCOVERY)", len(jwksRaw))
		}
	}
	audiences, _ := f.GetStringArray("inbound-jwt-audience")
	clients, _ := f.GetStringArray("inbound-jwt-client")
	scopes, _ := f.GetStringArray("inbound-jwt-scope")
	principalClaim, _ := f.GetString("inbound-jwt-principal-claim")
	return &runtimepkg.InboundAuth{
		Mode: mode,
		JWT: &runtimepkg.InboundAuthJWT{
			Source:           source,
			JWKS:             json.RawMessage(jwksRaw),
			DiscoveryURL:     discoveryURL,
			AllowedAudiences: audiences,
			AllowedClients:   clients,
			AllowedScopes:    scopes,
			PrincipalClaim:   principalClaim,
		},
	}, nil
}

// readImageAuth builds the imageAuth spec from the --image-auth-* flags. The
// API requires imageAuth to be present on create/update (@NotNull): with no
// image-auth flags (a public image) it returns {enabled: false} — a nil here
// used to make the server 400 on "imageAuth must not be null". username/
// password are required when auth is enabled; --image-auth-use-registry-
// credentials instead lets the platform pull with its own credentials (no
// username/password needed). The password is send-only (never present on
// responses); prefer --file for it.
func readImageAuth(cmd *cobra.Command) (*runtimepkg.ImageAuth, error) {
	f := cmd.Flags()
	enabled, _ := f.GetBool("image-auth-enabled")
	useRegistry, _ := f.GetBool("image-auth-use-registry-credentials")
	username, _ := f.GetString("image-auth-username")
	password, _ := f.GetString("image-auth-password")
	if useRegistry {
		return &runtimepkg.ImageAuth{Enabled: true, UseAgentBaseRegistryCredentials: &useRegistry}, nil
	}
	if !enabled && username == "" {
		// Public image: the spec must still carry imageAuth {enabled: false}.
		return &runtimepkg.ImageAuth{Enabled: false}, nil
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("--image-auth-enabled requires both --image-auth-username and --image-auth-password (or --image-auth-use-registry-credentials, or use --file)")
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

The whole spec is flag-driven. --name (immutable, prompted when unset),
--image-url, --flavor-id, repeatable --command/--args/--env KEY=VALUE, and
autoscaling (--max-replicas, --cpu-utilization, --memory-utilization; unset
sends the defaults 2/70/70).

Private-registry auth: --image-auth-username/--image-auth-password, or
--image-auth-use-registry-credentials to let the platform pull with its own
credentials. imageAuth is required by the API: with no image-auth flags the
request sends {enabled: false} (public image).

Network placement: --network-mode VPC with --vpc-id/--subnet-id and optional
repeatable --route-cidrs (unset = PUBLIC). Inbound auth: --inbound-mode
NONE/IAM/JWT (JWT adds --inbound-jwt-source DISCOVERY|JWKS and its flags).
Inbound CIDR allowlist: repeatable --allowed-cidrs. Protocol: --protocol
HTTP|MCP.

--file applies a full spec instead ('grn agentbase runtime generate' template)
— prefer it to keep the image-auth password (and inline JWKS key sets) out of
the shell history.

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
			req, err := loadRuntimeSpec(data, true)
			if err != nil {
				return err
			}
			// imageAuth is @NotNull server-side: a spec without it is a
			// public image — send {enabled: false} instead of letting the
			// server 400 on a missing field.
			if req.ImageAuth == nil {
				req.ImageAuth = &runtimepkg.ImageAuth{Enabled: false}
			}
			return createRuntimeAndPrint(ctx, cmd, req)
		}
		name, _ := cmd.Flags().GetString("name")
		name, err := cliinput.RequireOrPromptString(name, "--name", "Runtime name")
		if err != nil {
			return err
		}
		spec, err := readRuntimeSpec(cmd, false)
		if err != nil {
			return err
		}
		req := createRequestFromSpec(name, spec)
		// poc is create-only: send it only when the flag was explicitly set.
		if cmd.Flags().Changed("poc") {
			poc, _ := cmd.Flags().GetBool("poc")
			req.POC = &poc
		}
		return createRuntimeAndPrint(ctx, cmd, req)
	},
}

// createRequestFromSpec assembles the create request from the prompt-resolved
// name and the flag-read spec. The name must reach the request — a regression
// once dropped it and the server 400'd with "name must not be blank" while the
// flag path looked fine. Unit-tested in runtime_test.go.
func createRequestFromSpec(name string, spec *runtimeSpec) *runtimepkg.CreateAgentRuntimeRequest {
	return &runtimepkg.CreateAgentRuntimeRequest{
		Name:                 name,
		Description:          spec.Description,
		ImageURL:             spec.ImageURL,
		ImageAuth:            spec.ImageAuth,
		Command:              spec.Command,
		Args:                 spec.Args,
		EnvironmentVariables: spec.Env,
		NetworkConfig:        spec.Network,
		AllowedCidrs:         spec.AllowedCidrs,
		InboundAuth:          spec.Inbound,
		Protocol:             spec.Protocol,
		FlavorID:             spec.FlavorID,
		Autoscaling:          spec.Autoscaling,
	}
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
				NetworkConfig:        &runtimepkg.NetworkConfig{Mode: "PUBLIC"},
				AllowedCidrs:         []string{},
				InboundAuth:          &runtimepkg.InboundAuth{Mode: "NONE"},
				Protocol:             "HTTP",
				FlavorID:             "fill-flavor-id",
				Autoscaling:          runtimepkg.Autoscaling{MaxReplicas: 2, CPUUtilization: 70, MemoryUtilization: 70},
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
# Required: name, imageUrl, flavorId, imageAuth, autoscaling (and command/args/
# env, which may be empty lists/map but must be present). name is sealed
# (immutable); 5-50 chars, [a-zA-Z0-9._-].
name: my-runtime             # immutable
description: ""
imageUrl: registry.example.com/my-agent:latest
# imageAuth is required by the API. enabled: false (public image) is the
# default body below — for a private registry, enable it and fill username +
# password (send-only; the API never returns the password), or set
# useAgentBaseRegistryCredentials to let the platform pull with its own
# registry credentials.
imageAuth:
  enabled: false
  # useAgentBaseRegistryCredentials: false
  # username: ""
  # password: ""             # prefer keeping this in the file, not on the CLI
command:                     # container entrypoint (list); may be empty
  - python
  - -m
  - my_agent
args: []                     # container args (list); may be empty
environmentVariables:        # map; may be empty
  LOG_LEVEL: info
# networkConfig:             # optional network placement (omit = PUBLIC)
#   mode: VPC                # PUBLIC | VPC (VPC requires vpcId + subnetId)
#   vpcId: ""
#   subnetId: ""
#   routeCidrs: []           # egress routes (max 20)
# allowedCidrs:              # optional inbound CIDR allowlist (max 20)
#   - 10.0.0.0/8
# inboundAuth:               # optional inbound authentication (omit = NONE)
#   mode: JWT                # NONE | IAM | JWT
#   jwt:                     # required when mode is JWT
#     source: DISCOVERY      # DISCOVERY | JWKS
#     discoveryUrl: https://idp.example.com/.well-known/openid-configuration
#     # jwks: '{"keys": []}' # inline key set when source is JWKS (max 32KiB)
#     allowedAudiences: []   # max 20
#     allowedClients: []     # max 20
#     allowedScopes: []      # max 50
#     principalClaim: sub
protocol: HTTP               # HTTP | MCP (omit = HTTP)
flavorId: fill-flavor-id
poc: false                   # proof-of-concept runtime (create-only)
autoscaling:                 # HPA config (bounds: maxReplicas 1-10, util 10-90;
                             # minReplicas is platform-owned and no longer accepted)
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
	Short: "Update an agent runtime (new immutable version)",
	Long: `Update an agent runtime. Updating creates a new immutable version and rolls
the default endpoint forward.

The API is a full-spec replacement, but the flag path is merge-like: a field
whose flags are left unset is carried over from the current version (a stderr
note lists the carried fields), and a set flag replaces its field wholesale
(no element-level merge). Set flags only for what you intend to change.

imageAuth is the exception — the API never returns the stored password, so
username/password image auth cannot be carried over: re-supply
--image-auth-username/--image-auth-password (or --file with a full spec).
Without image auth the update re-sends {enabled: false}.

--file applies a full spec instead ('grn agentbase runtime generate' template
minus name) — it must include imageAuth explicitly, since a missing imageAuth
would silently wipe stored credentials.

The update is asynchronous; converge with 'grn agentbase runtime wait <id>'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]
		client, err := newRuntimeClient(ctx, cmd)
		if err != nil {
			return err
		}
		if runtimeFile != "" {
			data, err := os.ReadFile(runtimeFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			createReq, err := loadRuntimeSpec(data, false)
			if err != nil {
				return err
			}
			// imageAuth is required server-side and cannot be defaulted on
			// update: a missing imageAuth would silently wipe the stored
			// credentials. Make the user state it — {enabled: false} drops
			// them deliberately, username/password or
			// useAgentBaseRegistryCredentials keeps them.
			if createReq.ImageAuth == nil {
				return fmt.Errorf("spec is missing required field: imageAuth (the API never returns the stored password, so it cannot be carried over; set imageAuth explicitly: {enabled: false} drops it, username/password or useAgentBaseRegistryCredentials keeps it)")
			}
			// poc is create-only: a spec file carrying poc is accepted for
			// convenience but the field is not forwarded on update.
			return updateRuntimeAndPrint(ctx, client, cmd, id, &runtimepkg.UpdateAgentRuntimeRequest{
				Description:          createReq.Description,
				ImageURL:             createReq.ImageURL,
				ImageAuth:            createReq.ImageAuth,
				Command:              createReq.Command,
				Args:                 createReq.Args,
				EnvironmentVariables: createReq.EnvironmentVariables,
				NetworkConfig:        createReq.NetworkConfig,
				AllowedCidrs:         createReq.AllowedCidrs,
				InboundAuth:          createReq.InboundAuth,
				Protocol:             createReq.Protocol,
				FlavorID:             createReq.FlavorID,
				Autoscaling:          createReq.Autoscaling,
			})
		}
		spec, err := readRuntimeSpec(cmd, true)
		if err != nil {
			return err
		}
		// Full-spec replacement: base the request on the current version and
		// let the changed flags override — unset flags used to reset fields to
		// the server defaults (inboundAuth NONE, networkConfig PUBLIC, ...) .
		cur, err := latestVersion(ctx, client, id)
		if err != nil {
			return err
		}
		req, carried, err := buildUpdateRequest(spec, cur, cmd.Flags().Changed)
		if err != nil {
			return err
		}
		if len(carried) > 0 {
			fmt.Fprintf(os.Stderr, "Carrying over from version %d: %s\n", cur.Version, strings.Join(carried, ", "))
		}
		return updateRuntimeAndPrint(ctx, client, cmd, id, req)
	},
}

func updateRuntimeAndPrint(ctx context.Context, client *runtimepkg.Client, cmd *cobra.Command, id string, req *runtimepkg.UpdateAgentRuntimeRequest) error {
	rt, err := client.Update(ctx, id, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Runtime %q updating (state %s). Run 'grn agentbase runtime wait %s' to converge.\n",
		rt.Name, rt.Status, rt.ID)
	return output.PrintResource(rt, func() string { return rt.ID }, func() error { return renderRuntimeDetail(rt) })
}

// imageAuthFlags are the flags that explicitly set the imageAuth spec; any of
// them marks it as user-provided on update (otherwise it is derived from the
// current version).
var imageAuthFlags = []string{"image-auth-enabled", "image-auth-use-registry-credentials", "image-auth-username", "image-auth-password"}

// networkFlags are the flags that explicitly set networkConfig.
var networkFlags = []string{"network-mode", "vpc-id", "subnet-id", "route-cidrs"}

// autoscalingFlags are the flags that explicitly set autoscaling.
var autoscalingFlags = []string{"max-replicas", "cpu-utilization", "memory-utilization"}

// anyChanged reports whether any of the named flags was explicitly set.
func anyChanged(changed func(string) bool, names ...string) bool {
	for _, name := range names {
		if changed(name) {
			return true
		}
	}
	return false
}

// buildUpdateRequest assembles the update request for the flag path. The API
// is a full-spec replacement, so a field whose flags are unset is carried over
// from the current version (and recorded in the returned carry list for the
// caller's stderr note) instead of silently resetting to the server defaults
// (inboundAuth NONE, networkConfig PUBLIC, empty env/command/args). A changed
// flag replaces its field wholesale (no element-level merge).
//
// imageAuth is the exception: the API never returns the stored password and an
// update with enabled=false wipes the stored credentials, so it cannot be
// carried over blindly — see carriedImageAuth.
func buildUpdateRequest(spec *runtimeSpec, cur *runtimepkg.AgentRuntimeVersionDto, changed func(string) bool) (*runtimepkg.UpdateAgentRuntimeRequest, []string, error) {
	req := &runtimepkg.UpdateAgentRuntimeRequest{}
	var carried []string

	if changed("description") {
		req.Description = spec.Description
	} else {
		req.Description = cur.Description
		carried = append(carried, "description")
	}
	if changed("image-url") {
		req.ImageURL = spec.ImageURL
	} else {
		req.ImageURL = cur.ImageURL
		carried = append(carried, "imageUrl")
	}
	if changed("command") {
		req.Command = spec.Command
	} else {
		req.Command = cur.Command
		carried = append(carried, "command")
	}
	if changed("args") {
		req.Args = spec.Args
	} else {
		req.Args = cur.Args
		carried = append(carried, "args")
	}
	if changed("env") {
		req.EnvironmentVariables = spec.Env
	} else {
		req.EnvironmentVariables = cur.EnvironmentVariables
		carried = append(carried, "environmentVariables")
	}
	if anyChanged(changed, networkFlags...) {
		req.NetworkConfig = spec.Network
	} else {
		req.NetworkConfig = versionNetworkConfig(cur)
		carried = append(carried, "networkConfig")
	}
	if changed("allowed-cidrs") {
		req.AllowedCidrs = spec.AllowedCidrs
	} else {
		req.AllowedCidrs = cur.AllowedCidrs
		carried = append(carried, "allowedCidrs")
	}
	if changed("inbound-mode") {
		req.InboundAuth = spec.Inbound
	} else {
		req.InboundAuth = versionInboundAuth(cur)
		carried = append(carried, "inboundAuth")
	}
	if changed("protocol") {
		req.Protocol = spec.Protocol
	} else {
		req.Protocol = cur.Protocol
		carried = append(carried, "protocol")
	}
	if changed("flavor-id") {
		req.FlavorID = spec.FlavorID
	} else {
		req.FlavorID = cur.FlavorID
		carried = append(carried, "flavorId")
	}
	if anyChanged(changed, autoscalingFlags...) {
		req.Autoscaling = spec.Autoscaling
	} else {
		req.Autoscaling = cur.Autoscaling
		carried = append(carried, "autoscaling")
	}

	if anyChanged(changed, imageAuthFlags...) {
		req.ImageAuth = spec.ImageAuth
	} else {
		ia, err := carriedImageAuth(cur)
		if err != nil {
			return nil, nil, err
		}
		req.ImageAuth = ia
		carried = append(carried, "imageAuth")
	}
	return req, carried, nil
}

// carriedImageAuth rebuilds the update's imageAuth from the current version.
// See buildUpdateRequest for why username/password auth cannot be carried over
// (the API never returns the stored password, and defaulting to enabled=false
// would wipe it).
func carriedImageAuth(cur *runtimepkg.AgentRuntimeVersionDto) (*runtimepkg.ImageAuth, error) {
	if cur.ImageAuth == nil || !cur.ImageAuth.Enabled {
		// imageAuth is required server-side: re-send {enabled: false}.
		return &runtimepkg.ImageAuth{Enabled: false}, nil
	}
	if cur.ImageAuth.UseAgentBaseRegistryCredentials {
		// Registry-credentials mode needs no password: carry it verbatim.
		useRegistry := true
		return &runtimepkg.ImageAuth{
			Enabled:                         true,
			UseAgentBaseRegistryCredentials: &useRegistry,
			Username:                        cur.ImageAuth.Username,
		}, nil
	}
	return nil, fmt.Errorf("current version uses username/password image auth: the API never returns the stored password, so it cannot be carried over; re-supply --image-auth-username/--image-auth-password (or apply a full spec with --file)")
}

// versionNetworkConfig converts a version's networkConfig entity into the
// update request's NetworkConfig (same shape, distinct types).
func versionNetworkConfig(cur *runtimepkg.AgentRuntimeVersionDto) *runtimepkg.NetworkConfig {
	if cur.NetworkConfig == nil {
		return nil
	}
	return &runtimepkg.NetworkConfig{
		Mode:       cur.NetworkConfig.Mode,
		VpcID:      cur.NetworkConfig.VpcID,
		SubnetID:   cur.NetworkConfig.SubnetID,
		RouteCidrs: cur.NetworkConfig.RouteCidrs,
	}
}

// versionInboundAuth converts a version's inboundAuth view into the update
// request's InboundAuth (same shape, distinct types; JWKS is copied raw).
func versionInboundAuth(cur *runtimepkg.AgentRuntimeVersionDto) *runtimepkg.InboundAuth {
	if cur.InboundAuth == nil {
		return nil
	}
	ia := &runtimepkg.InboundAuth{Mode: cur.InboundAuth.Mode}
	if cur.InboundAuth.JWT != nil {
		jwks := cur.InboundAuth.JWT.JWKS
		if len(jwks) == 0 {
			jwks = nil
		}
		ia.JWT = &runtimepkg.InboundAuthJWT{
			Source:           cur.InboundAuth.JWT.Source,
			JWKS:             jwks,
			DiscoveryURL:     cur.InboundAuth.JWT.DiscoveryURL,
			AllowedAudiences: cur.InboundAuth.JWT.AllowedAudiences,
			AllowedClients:   cur.InboundAuth.JWT.AllowedClients,
			AllowedScopes:    cur.InboundAuth.JWT.AllowedScopes,
			PrincipalClaim:   cur.InboundAuth.JWT.PrincipalClaim,
		}
	}
	return ia
}

// latestVersion fetches the runtime's current version spec: the entry with the
// highest version number, scanning every page of the versions list (each
// update appends a new immutable version, so the last one is the live spec).
func latestVersion(ctx context.Context, client *runtimepkg.Client, id string) (*runtimepkg.AgentRuntimeVersionDto, error) {
	const size = 100
	var latest *runtimepkg.AgentRuntimeVersionDto
	for page := 1; page <= 100; page++ {
		resp, err := client.ListVersions(ctx, id, page, size)
		if err != nil {
			return nil, fmt.Errorf("read current version: %w", err)
		}
		for i := range resp.ListData {
			if latest == nil || resp.ListData[i].Version > latest.Version {
				latest = &resp.ListData[i]
			}
		}
		if page >= resp.TotalPage || len(resp.ListData) == 0 {
			break
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("runtime %s has no versions to carry fields over from", id)
	}
	return latest, nil
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
	poc := ""
	if rt.POC != nil {
		poc = fmt.Sprintf("%t", *rt.POC)
	}
	rows := [][]string{
		{"ID", rt.ID},
		{"Name", rt.Name},
		{"Description", output.StrOrDash(rt.Description)},
		{"State", rt.Status},
		{"Display Status", output.StrOrDash(rt.DisplayStatus)},
		{"Status Reason", output.StrOrDash(rt.StatusReason)},
		{"PoC", output.StrOrDash(poc)},
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
// tags directly). Used by both create (--file, name required) and update
// (--file, name omitted/ignored: the create template minus name works as-is).
func loadRuntimeSpec(data []byte, requireName bool) (*runtimepkg.CreateAgentRuntimeRequest, error) {
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
	if req.ImageURL == "" || req.FlavorID == "" {
		return nil, fmt.Errorf("spec is missing required field(s): imageUrl, flavorId")
	}
	if requireName && req.Name == "" {
		return nil, fmt.Errorf("spec is missing required field: name")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Sub-resources (Slice 4): endpoints / runtime-level ops / tracing
// ---------------------------------------------------------------------------

var runtimeEndpointCmd = &cobra.Command{Use: "endpoint", Short: "Manage a runtime's endpoints", RunE: requireSubcommand}
var runtimeTraceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Query the tracing backend (raw passthrough)",
	Long: `Query the runtime tracing backend.

These are Google-AIP custom verbs that forward arbitrary query params to the
tracing backend and return that backend's raw JSON. Use --param key=value
(repeatable) to pass query params through verbatim; the response is printed
raw (use -o json for clean stdout).`,
	RunE: requireSubcommand,
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
		{"Outbound IPs", output.StrOrDash(strings.Join(ep.OutboundIps, ", "))},
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
	create.Flags().StringVarP(&runtimeName, "name", "n", "", "Runtime name (required without --interactive, immutable; 5-50 chars)")
	create.Flags().Bool("poc", false, "Mark as a proof-of-concept runtime (create-only)")
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
// update (everything except --name and --poc, which are create-only).
// autoscaling has flag defaults so it is always populated without prompting.
// The network/inbound flags stay unset by default (platform defaults: public
// network, no inbound auth, HTTP).
func addRuntimeSpecFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.String("description", "", "Description")
	f.String("image-url", "", "Container image URL (required)")
	f.Bool("image-auth-enabled", false, "Enable private-registry auth (requires --image-auth-username/password)")
	f.String("image-auth-username", "", "Private-registry username (with --image-auth-enabled)")
	f.String("image-auth-password", "", "Private-registry password (with --image-auth-enabled; prefer --file)")
	f.Bool("image-auth-use-registry-credentials", false, "imageAuth: let the platform pull using its own registry credentials")
	f.StringArray("command", nil, "Container entrypoint element (repeatable)")
	f.StringArray("args", nil, "Container arg (repeatable)")
	f.StringArray("env", nil, "Environment variable KEY=VALUE (repeatable)")
	f.String("network-mode", "", "networkConfig mode: PUBLIC or VPC (unset = PUBLIC)")
	f.String("vpc-id", "", "networkConfig vpcId (required with --network-mode VPC)")
	f.String("subnet-id", "", "networkConfig subnetId (required with --network-mode VPC)")
	f.StringArray("route-cidrs", nil, "networkConfig egress route CIDR (repeatable, max 20)")
	f.StringArray("allowed-cidrs", nil, "Inbound CIDR allowlist (repeatable, max 20)")
	f.String("inbound-mode", "", "inboundAuth mode: NONE, IAM or JWT (unset = NONE)")
	f.String("inbound-jwt-source", "", "inboundAuth JWT key source: DISCOVERY or JWKS (with --inbound-mode JWT)")
	f.String("inbound-jwt-jwks", "", "inboundAuth JWT inline key set JSON (with source JWKS, max 32KiB; prefer --file)")
	f.String("inbound-jwt-discovery-url", "", "inboundAuth JWT OIDC discovery URL (with source DISCOVERY)")
	f.StringArray("inbound-jwt-audience", nil, "inboundAuth JWT allowed audience (repeatable, max 20)")
	f.StringArray("inbound-jwt-client", nil, "inboundAuth JWT allowed client (repeatable, max 20)")
	f.StringArray("inbound-jwt-scope", nil, "inboundAuth JWT allowed scope (repeatable, max 50)")
	f.String("inbound-jwt-principal-claim", "", "inboundAuth JWT principal claim (default sub)")
	f.String("protocol", "", "Runtime protocol: HTTP or MCP (unset = HTTP)")
	f.String("flavor-id", "", "Flavor id (required)")
	f.Int("min-replicas", 1, "Deprecated: ignored; the platform owns the replica floor (always 1)")
	f.Int("max-replicas", 2, "Autoscaling: maximum replicas (1-10)")
	f.Int("cpu-utilization", 70, "Autoscaling: CPU scale-up threshold % (10-90)")
	f.Int("memory-utilization", 70, "Autoscaling: memory scale-up threshold % (10-90)")
	// Hidden + deprecated, not removed, so existing scripts only warn.
	_ = f.MarkDeprecated("min-replicas", "minReplicas is platform-owned (always 1); the flag is ignored")
}

// runtimeName holds the --name value for create.
var runtimeName string
