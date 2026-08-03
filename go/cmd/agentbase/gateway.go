package agentbase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/greennodehub/greennode-cli/internal/agentbase/cliinput"
	gatewaypkg "github.com/greennodehub/greennode-cli/internal/agentbase/gateway"
	"github.com/greennodehub/greennode-cli/internal/agentbase/output"
)

// gatewayCmd groups the MCP gateway lifecycle commands. The agentbase /gateway
// endpoint fronts the agent-core-gateway REST API (POST/GET/PATCH/DELETE
// /api/v1/gateways); gateways converge asynchronously (WAITING_CREATING →
// CREATING → ACTIVE), so `create`/`update`/`delete` return immediately and
// `wait` polls to a terminal state.
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage MCP gateways",
	Long: `Create and manage MCP gateways (the agent-core-gateway service).

A gateway is created asynchronously: create/update/delete return the resource
in a WAITING_* state and converge to ACTIVE (or DELETED). Use 'wait' to block
until a terminal state.

Gateway creation has many nested, mutually-exclusive fields (targets,
outboundAuth, inboundAuth/JWT, privateNetwork). For anything beyond the simple
path, generate a template, fill it in, and apply it with --file:

    grn agentbase gateway generate > gw.yaml
    # ...edit gw.yaml...
    grn agentbase gateway create --file gw.yaml
    grn agentbase gateway wait my-gateway

gateways share the ~/.greennode profile like the rest of agentbase.`,
}

// newGatewayClient mirrors newIdentityClient: resolve the shared profile +
// env, select the shared token provider, force-mint once so auth failures
// surface before the first call, and point the typed client at the gateway
// endpoint for the active env.
func newGatewayClient(ctx context.Context, cmd *cobra.Command) (*gatewaypkg.Client, error) {
	ab := mustLoadAgentbaseCtx(cmd)
	provider, err := newAuthProvider(ab)
	if err != nil {
		return nil, err
	}
	if _, err := provider.GetToken(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return gatewaypkg.NewClient(ab.endpoints.Gateway, provider), nil
}

// ---------------------------------------------------------------------------
// create
// ---------------------------------------------------------------------------

var (
	gatewayCreateFile string
)

var gatewayCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new MCP gateway",
	Long: `Create a new MCP gateway.

By default the gateway is built from flags (the simple path: no inline targets).
For targets, outbound auth, JWT inbound auth, or anything nested, use --file
with a template produced by 'grn agentbase gateway generate'.

The gateway is created asynchronously; this command returns as soon as the
service accepts the spec (state WAITING_CREATING). Converge with
'grn agentbase gateway wait <name>'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if gatewayCreateFile != "" {
			data, err := os.ReadFile(gatewayCreateFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			req, err := loadCreateSpec(data)
			if err != nil {
				return err
			}
			return createAndPrint(ctx, cmd, req)
		}
		req, err := buildCreateFromFlags(cmd)
		if err != nil {
			return err
		}
		return createAndPrint(ctx, cmd, req)
	},
}

func createAndPrint(ctx context.Context, cmd *cobra.Command, req *gatewaypkg.CreateGatewayRequest) error {
	client, err := newGatewayClient(ctx, cmd)
	if err != nil {
		return err
	}
	gw, err := client.Create(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Gateway %q submitted (state %s). Run 'grn agentbase gateway wait %s' to converge.\n", gw.Name, gw.State, gw.Name)
	return output.PrintResource(gw, func() string { return gw.Name }, func() error { return renderGatewayDetail(gw) })
}

// buildCreateFromFlags assembles a CreateGatewayRequest from flags (the simple
// path). Targets are not supported from flags — use --file for those.
func buildCreateFromFlags(cmd *cobra.Command) (*gatewaypkg.CreateGatewayRequest, error) {
	f := cmd.Flags()

	name, _ := f.GetString("name")
	name, err := cliinput.RequireOrPromptString(name, "--name", "Gateway name (3-40 chars, [a-z0-9-])")
	if err != nil {
		return nil, err
	}

	networkMode, _ := f.GetString("network-mode")
	networkMode, err = cliinput.RequireOrPromptString(networkMode, "--network-mode", "Network mode (PUBLIC|PRIVATE)")
	if err != nil {
		return nil, err
	}
	networkMode = strings.ToUpper(networkMode)

	flavorID, _ := f.GetString("flavor-id")
	flavorID, err = cliinput.RequireOrPromptString(flavorID, "--flavor-id", "Flavor id")
	if err != nil {
		return nil, err
	}

	replicas, _ := f.GetInt("replicas")
	if replicas <= 0 {
		if cliinput.IsInteractive() {
			replicas = cliinput.PromptIntDefault("Replicas (1-10)", 1)
		} else {
			return nil, fmt.Errorf("required flag %q not set", "--replicas")
		}
	}

	inboundMode, _ := f.GetString("inbound-mode")
	inboundMode, err = cliinput.RequireOrPromptString(inboundMode, "--inbound-mode", "Inbound auth mode (NONE|IAM|JWT)")
	if err != nil {
		return nil, err
	}
	inboundMode = strings.ToUpper(inboundMode)

	req := &gatewaypkg.CreateGatewayRequest{
		Name:        name,
		NetworkMode: networkMode,
		FlavorID:    flavorID,
		Replicas:    replicas,
		InboundAuth: gatewaypkg.InboundAuthRequest{Mode: inboundMode},
	}
	if v, _ := f.GetString("display-name"); v != "" {
		req.DisplayName = v
	}
	if v, _ := f.GetString("description"); v != "" {
		req.Description = v
	}
	if v, _ := f.GetString("policy-group-id"); v != "" {
		req.PolicyGroupID = v
	}
	if v, _ := f.GetStringArray("client-redirect-uri"); len(v) > 0 {
		req.InboundAuth.ClientRedirectURIs = v
	}
	if f.Changed("iam-require-owner") {
		b, _ := f.GetBool("iam-require-owner")
		req.InboundAuth.IAMRequireOwner = &b
	}
	if inboundMode == "JWT" || f.Changed("jwt-source") || f.Changed("jwt-discovery-url") || f.Changed("jwt-jwks") {
		req.InboundAuth.JWT = buildJWTFromFlags(cmd)
	}

	// PRIVATE-mode private network (vpcId/subnetId sealed at create).
	if networkMode == "PRIVATE" {
		vpc, _ := f.GetString("private-vpc-id")
		vpc, err = cliinput.RequireOrPromptString(vpc, "--private-vpc-id", "VPC id")
		if err != nil {
			return nil, err
		}
		subnet, _ := f.GetString("private-subnet-id")
		subnet, err = cliinput.RequireOrPromptString(subnet, "--private-subnet-id", "Subnet id")
		if err != nil {
			return nil, err
		}
		pn := &gatewaypkg.PrivateNetworkInput{VPCID: vpc, SubnetID: subnet}
		if v, _ := f.GetStringArray("private-route"); len(v) > 0 {
			pn.Routes = v
		}
		if f.Changed("public-endpoint-enabled") {
			b, _ := f.GetBool("public-endpoint-enabled")
			pn.PublicEndpointEnabled = b
		}
		req.PrivateNetwork = pn
	} else if f.Changed("private-vpc-id") || f.Changed("private-subnet-id") {
		return nil, fmt.Errorf("private-network flags require --network-mode PRIVATE")
	}

	if v, _ := f.GetStringArray("allowed-cidr"); len(v) > 0 {
		arr := v
		req.AllowedCIDRs = &arr
	}
	if f.Changed("host-alias") {
		raw, _ := f.GetStringArray("host-alias")
		ha, err := parseHostAliases(raw)
		if err != nil {
			return nil, err
		}
		req.HostAliases = ha
	}
	return req, nil
}

func buildJWTFromFlags(cmd *cobra.Command) *gatewaypkg.JWTConfigReq {
	f := cmd.Flags()
	jwt := &gatewaypkg.JWTConfigReq{}
	if v, _ := f.GetString("jwt-source"); v != "" {
		jwt.Source = strings.ToUpper(v)
	}
	if v, _ := f.GetString("jwt-discovery-url"); v != "" {
		jwt.DiscoveryURL = v
	}
	if v, _ := f.GetString("jwt-jwks"); v != "" {
		jwt.JWKS = v
	}
	if v, _ := f.GetStringArray("allowed-audience"); len(v) > 0 {
		jwt.AllowedAudiences = v
	}
	if v, _ := f.GetStringArray("allowed-client"); len(v) > 0 {
		jwt.AllowedClients = v
	}
	if v, _ := f.GetStringArray("allowed-scope"); len(v) > 0 {
		jwt.AllowedScopes = v
	}
	if v, _ := f.GetString("principal-claim"); v != "" {
		jwt.PrincipalClaim = v
	}
	return jwt
}

// parseHostAliases parses repeated --host-alias "ip=host1,host2" values.
func parseHostAliases(raw []string) ([]gatewaypkg.HostAliasInput, error) {
	out := make([]gatewaypkg.HostAliasInput, 0, len(raw))
	for _, r := range raw {
		eq := strings.Index(r, "=")
		if eq < 0 {
			return nil, fmt.Errorf("invalid --host-alias %q: expected ip=host1,host2", r)
		}
		ip := strings.TrimSpace(r[:eq])
		var hosts []string
		for _, h := range strings.Split(r[eq+1:], ",") {
			if h = strings.TrimSpace(h); h != "" {
				hosts = append(hosts, h)
			}
		}
		if ip == "" || len(hosts) == 0 {
			return nil, fmt.Errorf("invalid --host-alias %q: needs an ip and at least one hostname", r)
		}
		out = append(out, gatewaypkg.HostAliasInput{IP: ip, Hostnames: hosts})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// generate — print a commented create-template (kubectl-style)
// ---------------------------------------------------------------------------

var gatewayGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print a gateway create template (YAML or JSON)",
	Long: `Print a commented gateway create template to stdout. Save it, fill it in, and
apply with 'grn agentbase gateway create --file <file>'.

Defaults to YAML (with comments); pass -o json for a JSON skeleton.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.GetFormat() == output.FormatJSON {
			example := &gatewaypkg.CreateGatewayRequest{
				Name:        "my-gateway",
				NetworkMode: "PUBLIC",
				FlavorID:    "fill-flavor-id",
				Replicas:    1,
				InboundAuth: gatewaypkg.InboundAuthRequest{Mode: "NONE"},
			}
			b, err := json.MarshalIndent(example, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		fmt.Print(gatewayCreateTemplateYAML)
		return nil
	},
}

// gatewayCreateTemplateYAML is a hand-written, commented skeleton of
// CreateGatewayRequest. Keys are the JSON (camelCase) field names so the file
// round-trips through 'create --file' exactly.
const gatewayCreateTemplateYAML = `# Gateway create spec.
# Fill in and apply with:  grn agentbase gateway create --file <this-file>
#
# Required: name, networkMode, flavorId, replicas, inboundAuth.mode
# Sealed at create (recreate to change): name, networkMode, flavorId, replicas,
#   and (PRIVATE) privateNetwork.vpcId/subnetId.
name: my-gateway            # 3-40 chars, [a-z0-9-]
displayName: ""
description: ""
networkMode: PUBLIC         # PUBLIC | PRIVATE  (sealed)
flavorId: fill-flavor-id    # a catalog flavor id (sealed)
replicas: 1                 # 1..10  (sealed)
policyGroupId: ""           # optional; bind a policy group (FK)

# Inbound (caller) authentication.
inboundAuth:
  mode: NONE                # NONE | IAM | JWT  (required)
  # clientRedirectUris:       # DCR/authorize allowlist
  #   - https://app.example.com/callback
  # iamRequireOwner: true     # IAM mode only
  # jwt:                      # required when mode=JWT
  #   source: DISCOVERY       # DISCOVERY | JWKS
  #   discoveryUrl: https://idp.example.com/.well-known/openid-configuration
  #   jwks: ""                # inline JWKS (<=32KiB) when source=JWKS
  #   allowedAudiences: []
  #   allowedClients: []
  #   allowedScopes: []
  #   principalClaim: sub

# Upstream MCP servers (whole-list replace; no per-target add/remove API).
# targets:
#   - name: weather          # 3-50 chars
#     type: MCP              # only MCP today
#     endpoint: https://mcp.example.com
#     outboundAuth:
#       type: NONE           # NONE | APIKEY | OAUTH | INBOUND_FORWARD
#       # providerSource: CUSTOM   # OAUTH only: CUSTOM | MANAGED
#       # flow: 2LO                # APIKEY/OAUTH: 2LO | 3LO
#       # providerName: ""
#       # scopes: []
#       # returnUrl: ""            # 3LO only
#       # headerName: ""
#       # headerValuePrefix: ""
#       # customParameters: {}

# Inbound client-IP allowlist (IPv4 CIDRs). Omit = allow all (0.0.0.0/0);
# an explicit empty list blocks all client IPs.
# allowedCidrs:
#   - 0.0.0.0/0

# /etc/hosts overrides applied to gateway pods.
# hostAliases:
#   - ip: 10.0.0.1
#     hostnames:
#       - foo.local

# PRIVATE mode only (required when networkMode=PRIVATE; forbidden otherwise).
# privateNetwork:
#   vpcId: fill-vpc-id       # sealed at create
#   subnetId: fill-subnet-id # sealed at create
#   routes:                  # CIDRs the worker programs as node routes
#     - 172.16.0.0/12
#   publicEndpointEnabled: false
`

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

var gatewayListCmd = &cobra.Command{
	Use:   "list",
	Short: "List gateways",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")
		client, err := newGatewayClient(ctx, cmd)
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
			if len(resp.Items) > 0 {
				output.PrintID(resp.Items[0].Name)
			}
			return nil
		}
		if len(resp.Items) == 0 {
			fmt.Fprintln(os.Stderr, "No gateways found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Items))
		for i := range resp.Items {
			g := resp.Items[i]
			rows = append(rows, []string{
				g.Name, g.NetworkMode, g.State, flavorStr(g.Flavor),
				strconv.Itoa(g.Replicas), endpointStr(&g), formatTimeVal(g.CreatedAt),
			})
		}
		output.Table([]string{"Name", "Mode", "State", "Flavor", "Replicas", "Endpoint", "Created"}, rows)
		p := resp.Pagination
		fmt.Fprintf(os.Stderr, "Page %d of %d (%d total items)\n", p.Page, totalPages(p), p.TotalItems)
		return nil
	},
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

var gatewayGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show a gateway",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		gw, err := client.Get(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(gw, func() string { return gw.Name }, func() error { return renderGatewayDetail(gw) })
	},
}

// ---------------------------------------------------------------------------
// update (JSON Merge Patch, RFC 7396)
// ---------------------------------------------------------------------------

var gatewayUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a gateway's mutable fields",
	Long: `Update a gateway's mutable fields (JSON Merge Patch semantics: omit = leave
alone, null = clear, value = replace). Sealed fields (name, networkMode,
flavorId, replicas, privateNetwork.vpcId/subnetId) cannot be changed — recreate
the gateway instead.

Flags set only the simple mutable fields. For inboundAuth, targets,
hostAliases, or privateNetwork.routes use --file with a partial merge-patch
(template: 'grn agentbase gateway generate', then keep only the keys to change).

Clear the policy group with --clear-policy-group-id.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		if gatewayCreateFile != "" {
			data, err := os.ReadFile(gatewayCreateFile)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			patch, err := yamlToMap(data)
			if err != nil {
				return err
			}
			return updateAndPrint(ctx, cmd, name, patch)
		}

		f := cmd.Flags()
		patch := map[string]interface{}{}
		if f.Changed("display-name") {
			v, _ := f.GetString("display-name")
			patch["displayName"] = v
		}
		if f.Changed("description") {
			v, _ := f.GetString("description")
			patch["description"] = v
		}
		if f.Changed("policy-group-id") {
			v, _ := f.GetString("policy-group-id")
			patch["policyGroupId"] = v
		}
		if clear, _ := f.GetBool("clear-policy-group-id"); clear {
			patch["policyGroupId"] = nil
		}
		if f.Changed("allowed-cidr") {
			v, _ := f.GetStringArray("allowed-cidr")
			patch["allowedCidrs"] = v
		}
		if f.Changed("host-alias") {
			raw, _ := f.GetStringArray("host-alias")
			ha, err := parseHostAliases(raw)
			if err != nil {
				return err
			}
			patch["hostAliases"] = ha
		}
		if len(patch) == 0 {
			return fmt.Errorf("no changes specified: set flags or pass --file (template: 'grn agentbase gateway generate')")
		}
		return updateAndPrint(ctx, cmd, name, patch)
	},
}

func updateAndPrint(ctx context.Context, cmd *cobra.Command, name string, patch map[string]interface{}) error {
	client, err := newGatewayClient(ctx, cmd)
	if err != nil {
		return err
	}
	gw, err := client.Update(ctx, name, patch)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Gateway %q updating (state %s). Run 'grn agentbase gateway wait %s' to converge.\n", gw.Name, gw.State, gw.Name)
	return output.PrintResource(gw, func() string { return gw.Name }, func() error { return renderGatewayDetail(gw) })
}

// ---------------------------------------------------------------------------
// delete
// ---------------------------------------------------------------------------

var gatewayDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a gateway",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.Delete(ctx, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Gateway %q deleting. Run 'grn agentbase gateway wait %s' to confirm.\n", args[0], args[0])
		output.PrintDeletedID(args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// wait — poll get until a terminal state
// ---------------------------------------------------------------------------

var gatewayWaitCmd = &cobra.Command{
	Use:   "wait <name>",
	Short: "Wait for a gateway to reach a terminal state",
	Long: `Poll a gateway until it reaches a terminal state: ACTIVE / DELETED (success)
or CREATE_ERROR / UPDATE_ERROR / ERROR (failure). Use after create/update/delete.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		timeout, _ := cmd.Flags().GetDuration("timeout")
		interval, _ := cmd.Flags().GetDuration("interval")
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		name := args[0]
		for {
			gw, err := client.Get(ctx, name)
			if err != nil {
				return err
			}
			if gatewayTerminalFailure[gw.State] {
				return fmt.Errorf("gateway %q failed: state=%s %s", name, gw.State, lastErrStr(gw.LastError))
			}
			if gatewayTerminalSuccess[gw.State] {
				fmt.Fprintf(os.Stderr, "Gateway %q reached state %s.\n", name, gw.State)
				return output.PrintResource(gw, func() string { return gw.Name }, func() error { return renderGatewayDetail(gw) })
			}
			fmt.Fprintf(os.Stderr, "Gateway %q state: %s ...\n", name, gw.State)
			select {
			case <-ctx.Done():
				return fmt.Errorf("timed out waiting for gateway %q (last state %s)", name, gw.State)
			case <-time.After(interval):
			}
		}
	},
}

// gatewayTerminalSuccess / gatewayTerminalFailure partition the State enum.
// The transient (WAITING_*/CREATING/UPDATING/DELETING/*_CLEANING) states keep
// polling; everything else is terminal.
var (
	gatewayTerminalSuccess = map[string]bool{
		"ACTIVE":  true,
		"DELETED": true,
	}
	gatewayTerminalFailure = map[string]bool{
		"CREATE_ERROR":   true,
		"UPDATE_ERROR":   true,
		"ERROR":          true,
		"ERROR_DELETING": true,
	}
)

// ---------------------------------------------------------------------------
// rendering helpers
// ---------------------------------------------------------------------------

func renderGatewayDetail(gw *gatewaypkg.GatewayResponse) error {
	rows := [][]string{
		{"ID", gw.ID},
		{"Name", gw.Name},
		{"Display Name", output.StrOrDash(gw.DisplayName)},
		{"Description", output.StrOrDash(gw.Description)},
		{"Network Mode", gw.NetworkMode},
		{"State", gw.State},
		{"Flavor", flavorStr(gw.Flavor)},
		{"Replicas", strconv.Itoa(gw.Replicas)},
		{"Inbound Auth", gw.InboundAuth.Mode},
		{"Policy Group", output.StrOrDash(gw.PolicyGroupID)},
		{"Agent Identity", output.StrOrDash(gw.AgentIdentityName)},
		{"Endpoint", endpointStr(gw)},
		{"Allowed CIDRs", strList(gw.AllowedCIDRs)},
		{"Targets", strconv.Itoa(len(gw.Targets))},
		{"Host Aliases", strconv.Itoa(len(gw.HostAliases))},
		{"IAM Service Account", output.StrOrDash(gw.IAM.ServiceAccountID)},
		{"Last Error", lastErrStr(gw.LastError)},
		{"Created", formatTimeVal(gw.CreatedAt)},
		{"Updated", formatTimeVal(gw.UpdatedAt)},
	}
	output.Table([]string{"Field", "Value"}, rows)
	return nil
}

func flavorStr(f *gatewaypkg.FlavorSnapshotResponse) string {
	if f == nil {
		return "-"
	}
	return fmt.Sprintf("%s (cpu=%d, mem=%dGi)", output.StrOrDash(f.DisplayName), f.CPU, f.MemoryGi)
}

// endpointStr prefers the mode-specific endpoint, falling back to the generic one.
func endpointStr(gw *gatewaypkg.GatewayResponse) string {
	switch gw.NetworkMode {
	case "PUBLIC":
		if gw.PublicEndpoint != "" {
			return gw.PublicEndpoint
		}
	case "PRIVATE":
		if gw.PrivateEndpoint != "" {
			return gw.PrivateEndpoint
		}
	}
	return output.StrOrDash(gw.Endpoint)
}

func lastErrStr(le *gatewaypkg.LastErrorResponse) string {
	if le == nil {
		return "-"
	}
	return fmt.Sprintf("%s: %s (stage %s)", le.Code, le.Message, le.Stage)
}

func strList(s []string) string {
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, ", ")
}

func formatTimeVal(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func totalPages(p gatewaypkg.Pagination) int64 {
	if p.PageSize <= 0 {
		return 0
	}
	return (p.TotalItems + p.PageSize - 1) / p.PageSize
}

// ---------------------------------------------------------------------------
// file parsing (YAML or JSON -> struct / merge-patch map)
// ---------------------------------------------------------------------------

// yamlToMap parses a YAML or JSON document into a generic map. Used for update
// --file (raw merge-patch). json round-trips through the map so the caller can
// re-decode into a typed struct using its JSON tags (see loadCreateSpec).
func yamlToMap(data []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse YAML/JSON spec: %w", err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}

// loadCreateSpec parses a YAML/JSON create spec into CreateGatewayRequest.
// yaml.Unmarshal into a map then json.Unmarshal into the struct so the file's
// camelCase keys bind to the struct's json tags (yaml.v3 does not honor json
// tags directly).
func loadCreateSpec(data []byte) (*gatewaypkg.CreateGatewayRequest, error) {
	m, err := yamlToMap(data)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var req gatewaypkg.CreateGatewayRequest
	if err := json.Unmarshal(jb, &req); err != nil {
		return nil, fmt.Errorf("invalid gateway spec: %w", err)
	}
	if req.Name == "" || req.NetworkMode == "" || req.FlavorID == "" {
		return nil, fmt.Errorf("spec is missing required field(s): name, networkMode, flavorId (and replicas/inboundAuth.mode)")
	}
	return &req, nil
}

// ---------------------------------------------------------------------------

func init() {
	AgentbaseCmd.AddCommand(gatewayCmd)

	// create
	create := gatewayCreateCmd
	create.Flags().StringVarP(&gatewayCreateName, "name", "n", "", "Gateway name (required without --interactive)")
	create.Flags().String("display-name", "", "Display name")
	create.Flags().String("description", "", "Description")
	create.Flags().String("network-mode", "", "Network mode: PUBLIC | PRIVATE (required, sealed)")
	create.Flags().String("flavor-id", "", "Flavor id (required, sealed)")
	create.Flags().Int("replicas", 0, "Replica count 1-10 (required, sealed)")
	create.Flags().String("inbound-mode", "", "Inbound auth mode: NONE | IAM | JWT (required)")
	create.Flags().StringArray("client-redirect-uri", nil, "Allowed client redirect URI (repeatable)")
	create.Flags().Bool("iam-require-owner", false, "IAM mode: require the caller to own the resource")
	create.Flags().String("jwt-source", "", "JWT source: DISCOVERY | JWKS")
	create.Flags().String("jwt-discovery-url", "", "JWT OIDC discovery URL (DISCOVERY)")
	create.Flags().String("jwt-jwks", "", "JWT inline JWKS (JWKS)")
	create.Flags().StringArray("allowed-audience", nil, "Allowed JWT audience (repeatable)")
	create.Flags().StringArray("allowed-client", nil, "Allowed JWT client id (repeatable)")
	create.Flags().StringArray("allowed-scope", nil, "Allowed JWT scope (repeatable)")
	create.Flags().String("principal-claim", "", "JWT principal claim (default sub)")
	create.Flags().String("policy-group-id", "", "Policy group id to bind (FK)")
	create.Flags().String("private-vpc-id", "", "PRIVATE mode: VPC id (sealed)")
	create.Flags().String("private-subnet-id", "", "PRIVATE mode: subnet id (sealed)")
	create.Flags().StringArray("private-route", nil, "PRIVATE mode: node route CIDR (repeatable)")
	create.Flags().Bool("public-endpoint-enabled", false, "PRIVATE mode: expose a public endpoint")
	create.Flags().StringArray("allowed-cidr", nil, "Inbound client-IP allowlist CIDR (repeatable)")
	create.Flags().StringArray("host-alias", nil, "/etc/hosts override: ip=host1,host2 (repeatable)")
	create.Flags().StringVar(&gatewayCreateFile, "file", "", "Apply a spec file (see 'generate'); authoritative when set")
	gatewayCmd.AddCommand(create)

	// generate
	gatewayCmd.AddCommand(gatewayGenerateCmd)

	// list
	gatewayListCmd.Flags().Int("page", 1, "Page number (1-based)")
	gatewayListCmd.Flags().Int("size", 50, "Page size")
	gatewayCmd.AddCommand(gatewayListCmd)

	// get
	gatewayCmd.AddCommand(gatewayGetCmd)

	// update (reuses the --file flag var; simple mutable fields via flags)
	update := gatewayUpdateCmd
	update.Flags().StringVar(&gatewayCreateFile, "file", "", "Apply a partial merge-patch spec file")
	update.Flags().String("display-name", "", "Set display name")
	update.Flags().String("description", "", "Set description")
	update.Flags().String("policy-group-id", "", "Set policy group id")
	update.Flags().Bool("clear-policy-group-id", false, "Clear the policy group binding")
	update.Flags().StringArray("allowed-cidr", nil, "Replace inbound allowlist (repeatable)")
	update.Flags().StringArray("host-alias", nil, "Replace /etc/hosts overrides: ip=host1,host2 (repeatable)")
	gatewayCmd.AddCommand(update)

	// delete
	gatewayCmd.AddCommand(gatewayDeleteCmd)

	// wait
	gatewayWaitCmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait")
	gatewayWaitCmd.Flags().Duration("interval", 5*time.Second, "Poll interval")
	gatewayCmd.AddCommand(gatewayWaitCmd)

	// --- Slice 5 sub-resources ---

	// flavors
	gatewayFlavorsListCmd.Flags().String("resource-type", "", "Filter by resource type")
	gatewayFlavorsListCmd.Flags().String("network-mode", "", "Filter by network mode (PUBLIC|PRIVATE)")
	gatewayFlavorsListCmd.Flags().String("zone-id", "", "Filter by zone id")
	gatewayFlavorsCmd.AddCommand(gatewayFlavorsListCmd)
	gatewayCmd.AddCommand(gatewayFlavorsCmd)

	// access-logs
	addAccessLogFilterFlags(gatewayAccessLogsListCmd)
	gatewayAccessLogsListCmd.Flags().Int("page", 1, "Page number (1-based)")
	gatewayAccessLogsListCmd.Flags().Int("page-size", 50, "Page size")
	gatewayAccessLogsCmd.AddCommand(gatewayAccessLogsListCmd)
	addAccessLogFilterFlags(gatewayAccessLogsStatsCmd)
	gatewayAccessLogsStatsCmd.Flags().String("interval", "", "Time-series bucket interval (e.g. 1h)")
	gatewayAccessLogsStatsCmd.Flags().Int("top-n", 5, "Number of top tools/targets/callers to return")
	gatewayAccessLogsCmd.AddCommand(gatewayAccessLogsStatsCmd)
	gatewayCmd.AddCommand(gatewayAccessLogsCmd)

	// inbound-auth / jwt / idp-app
	gatewayInboundAuthJwtIdpAppSetCmd.Flags().String("client-id", "", "IdP app client id (required)")
	gatewayInboundAuthJwtIdpAppSetCmd.Flags().String("client-secret", "", "IdP app client secret (omit to preserve existing)")
	gatewayInboundAuthJwtIdpAppSetCmd.Flags().StringArray("scope", nil, "IdP app scope (repeatable)")
	gatewayInboundAuthJwtIdpAppCmd.AddCommand(gatewayInboundAuthJwtIdpAppSetCmd)
	gatewayInboundAuthJwtIdpAppCmd.AddCommand(gatewayInboundAuthJwtIdpAppClearCmd)
	gatewayInboundAuthJwtCmd.AddCommand(gatewayInboundAuthJwtIdpAppCmd)
	gatewayInboundAuthCmd.AddCommand(gatewayInboundAuthJwtCmd)
	gatewayCmd.AddCommand(gatewayInboundAuthCmd)

	// private-network / routes
	gatewayPrivateNetworkRoutesSetCmd.Flags().StringArray("route", nil, "CIDR route (repeatable; ignored with --file)")
	gatewayPrivateNetworkRoutesSetCmd.Flags().String("if-match", "", "If-Match ETag for optimistic concurrency (optional)")
	gatewayPrivateNetworkRoutesSetCmd.Flags().String("file", "", "JSON/YAML {routes: [...]} spec (authoritative when set)")
	gatewayPrivateNetworkRoutesCmd.AddCommand(gatewayPrivateNetworkRoutesGetCmd)
	gatewayPrivateNetworkRoutesCmd.AddCommand(gatewayPrivateNetworkRoutesSetCmd)
	gatewayPrivateNetworkCmd.AddCommand(gatewayPrivateNetworkRoutesCmd)
	gatewayCmd.AddCommand(gatewayPrivateNetworkCmd)

	// service-account / repair
	gatewayServiceAccountCmd.AddCommand(gatewayServiceAccountRepairCmd)
	gatewayCmd.AddCommand(gatewayServiceAccountCmd)
}

// gatewayCreateName holds the --name value for create (the other create flags
// are read directly via cmd.Flags()).
var gatewayCreateName string

// ---------------------------------------------------------------------------
// Slice 5 sub-resources: flavors / access-logs / inbound-auth / private-network
// / service-account
// ---------------------------------------------------------------------------

// flavors list --------------------------------------------------------------

var gatewayFlavorsCmd = &cobra.Command{
	Use:   "flavors",
	Short: "Gateway placement flavors",
}

var gatewayFlavorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List gateway placement flavors",
	Long: `List gateway placement flavors (GET /api/v1/flavors). These are the flavors
selectable as a gateway's flavorId — distinct from the runtime compute-flavor
catalog. Filters are optional.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		resourceType, _ := f.GetString("resource-type")
		networkMode, _ := f.GetString("network-mode")
		zoneID, _ := f.GetString("zone-id")
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListFlavors(ctx, resourceType, networkMode, zoneID)
		if err != nil {
			return err
		}
		switch output.GetFormat() {
		case output.FormatJSON:
			return output.JSON(resp)
		case output.FormatID:
			if len(resp.Items) > 0 {
				output.PrintID(resp.Items[0].ID)
			}
			return nil
		}
		if len(resp.Items) == 0 {
			fmt.Fprintln(os.Stderr, "No flavors found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Items))
		for i := range resp.Items {
			it := resp.Items[i]
			rows = append(rows, []string{
				it.ID, it.DisplayName, strconv.Itoa(it.CPU), strconv.Itoa(it.MemoryGi),
				strList(it.NetworkModes), it.Availability, strconv.Itoa(it.SortOrder),
			})
		}
		output.Table([]string{"ID", "Display", "CPU", "Mem(Gi)", "Modes", "Availability", "Sort"}, rows)
		return nil
	},
}

// access-logs ---------------------------------------------------------------

var gatewayAccessLogsCmd = &cobra.Command{
	Use:   "access-logs",
	Short: "Gateway access logs",
}

// addAccessLogFilterFlags registers the shared access-log filter flags on a
// command (from/to/mcp-method/tool-name/target-name/http-status/client-ip).
func addAccessLogFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("from", "", "Filter: ISO8601 from (inclusive)")
	cmd.Flags().String("to", "", "Filter: ISO8601 to (exclusive)")
	cmd.Flags().String("mcp-method", "", "Filter: MCP method (e.g. tools/call)")
	cmd.Flags().String("tool-name", "", "Filter: MCP tool name")
	cmd.Flags().String("target-name", "", "Filter: upstream target name")
	cmd.Flags().String("http-status", "", "Filter: upstream HTTP status code")
	cmd.Flags().String("client-ip", "", "Filter: caller client IP")
}

// readAccessLogQuery reads the shared filter flags into AccessLogQuery. Only
// page/pageSize (list) or interval/topN (stats) are read by the caller.
func readAccessLogQuery(f *pflag.FlagSet) gatewaypkg.AccessLogQuery {
	return gatewaypkg.AccessLogQuery{
		From:       flagStr(f, "from"),
		To:         flagStr(f, "to"),
		MCPMethod:  flagStr(f, "mcp-method"),
		ToolName:   flagStr(f, "tool-name"),
		TargetName: flagStr(f, "target-name"),
		HTTPStatus: flagStr(f, "http-status"),
		ClientIP:   flagStr(f, "client-ip"),
	}
}

// flagStr reads a string flag, ignoring the lookup error (defaults to "").
func flagStr(f *pflag.FlagSet, name string) string {
	v, _ := f.GetString(name)
	return v
}

var gatewayAccessLogsListCmd = &cobra.Command{
	Use:   "list <name>",
	Short: "List a gateway's access-log entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		q := readAccessLogQuery(f)
		q.Page, _ = f.GetInt("page")
		q.PageSize, _ = f.GetInt("page-size")
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ListAccessLogs(ctx, args[0], q)
		if err != nil {
			return err
		}
		if output.GetFormat() == output.FormatJSON {
			return output.JSON(resp)
		}
		if len(resp.Items) == 0 {
			fmt.Fprintln(os.Stderr, "No access-log entries found.")
			return nil
		}
		rows := make([][]string, 0, len(resp.Items))
		for i := range resp.Items {
			it := resp.Items[i]
			rows = append(rows, []string{
				it.Timestamp, it.TargetName, it.MCP.Method, it.MCP.ToolName,
				strconv.Itoa(it.Response.Status), strconv.Itoa(it.DurationMs),
				accessLogErrStr(it.ErrorCode, it.ErrorMessage),
			})
		}
		output.Table([]string{"Time", "Target", "Method", "Tool", "Status", "DurMs", "Error"}, rows)
		p := resp.Pagination
		fmt.Fprintf(os.Stderr, "Page %d (size %d, %d total)\n", p.Page, p.PageSize, p.Total)
		return nil
	},
}

var gatewayAccessLogsStatsCmd = &cobra.Command{
	Use:   "stats <name>",
	Short: "Aggregate access-log stats for a gateway",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		q := readAccessLogQuery(f)
		q.Interval, _ = f.GetString("interval")
		q.TopN, _ = f.GetInt("top-n")
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.AccessLogStats(ctx, args[0], q)
		if err != nil {
			return err
		}
		if output.GetFormat() == output.FormatJSON {
			return output.JSON(resp)
		}
		rows := [][]string{
			{"Total Requests", strconv.Itoa(resp.TotalRequests)},
			{"Success Rate", fmt.Sprintf("%.4f", resp.SuccessRate)},
			{"Error Rate", fmt.Sprintf("%.4f", resp.ErrorRate)},
			{"Duration Avg (ms)", fmt.Sprintf("%.2f", resp.Duration.AvgMs)},
			{"Duration Max (ms)", fmt.Sprintf("%.2f", resp.Duration.MaxMs)},
			{"Duration Min (ms)", fmt.Sprintf("%.2f", resp.Duration.MinMs)},
			{"Range From", output.StrOrDash(resp.Range.From)},
			{"Range To", output.StrOrDash(resp.Range.To)},
			{"Interval", output.StrOrDash(resp.Range.Interval)},
		}
		output.Table([]string{"Metric", "Value"}, rows)
		if len(resp.StatusHistogram) > 0 {
			fmt.Fprintln(os.Stderr, "Status histogram:")
			hrows := make([][]string, 0, len(resp.StatusHistogram))
			for i := range resp.StatusHistogram {
				b := resp.StatusHistogram[i]
				hrows = append(hrows, []string{strconv.Itoa(b.Status), strconv.Itoa(b.Count)})
			}
			output.Table([]string{"Status", "Count"}, hrows)
		}
		printTermBuckets("Top tools", resp.TopTools)
		printTermBuckets("Top targets", resp.TopTargets)
		printTermBuckets("Top user agents", resp.TopUserAgents)
		printCallerBuckets(resp.TopCallers)
		return nil
	},
}

func accessLogErrStr(code, msg string) string {
	if code == "" && msg == "" {
		return "-"
	}
	if msg == "" {
		return code
	}
	return fmt.Sprintf("%s: %s", code, truncate(msg, 60))
}

func printTermBuckets(title string, buckets []gatewaypkg.AccessLogTermBucket) {
	if len(buckets) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "%s:\n", title)
	rows := make([][]string, 0, len(buckets))
	for i := range buckets {
		rows = append(rows, []string{output.StrOrDash(buckets[i].Name), strconv.Itoa(buckets[i].Count)})
	}
	output.Table([]string{"Name", "Count"}, rows)
}

func printCallerBuckets(buckets []gatewaypkg.AccessLogCallerBucket) {
	if len(buckets) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "Top callers:")
	rows := make([][]string, 0, len(buckets))
	for i := range buckets {
		rows = append(rows, []string{buckets[i].AuthMode, buckets[i].ID, strconv.Itoa(buckets[i].Count)})
	}
	output.Table([]string{"AuthMode", "ID", "Count"}, rows)
}

// inbound-auth / jwt / idp-app ----------------------------------------------

var gatewayInboundAuthCmd = &cobra.Command{
	Use:   "inbound-auth",
	Short: "Gateway inbound authentication",
}

var gatewayInboundAuthJwtCmd = &cobra.Command{
	Use:   "jwt",
	Short: "JWT inbound-auth configuration",
}

var gatewayInboundAuthJwtIdpAppCmd = &cobra.Command{
	Use:   "idp-app",
	Short: "Inbound-auth JWT IdP app credentials",
	Long: `Manage the gateway's inbound-auth JWT IdP application credentials (the
OAuth2 client/secret the gateway uses to talk to the IdP). The secret stays
server-side after it is set.`,
}

var gatewayInboundAuthJwtIdpAppSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set the inbound-auth JWT IdP app credentials",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		clientID, _ := f.GetString("client-id")
		clientID, err := cliinput.RequireOrPromptString(clientID, "--client-id", "IdP app client id")
		if err != nil {
			return err
		}
		req := &gatewaypkg.PutIdpAppRequest{ClientID: clientID}
		if f.Changed("client-secret") {
			secret, _ := f.GetString("client-secret")
			req.ClientSecret = &secret
		}
		if v, _ := f.GetStringArray("scope"); len(v) > 0 {
			req.Scopes = v
		}
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.PutIdpApp(ctx, args[0], req); err != nil {
			return err
		}
		output.Successf("IdP app credentials set for gateway %q.", args[0])
		return nil
	},
}

var gatewayInboundAuthJwtIdpAppClearCmd = &cobra.Command{
	Use:   "clear <name>",
	Short: "Clear the inbound-auth JWT IdP app credentials",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		if err := client.ClearIdpApp(ctx, args[0]); err != nil {
			return err
		}
		return output.PrintDeletedID(args[0])
	},
}

// private-network / routes --------------------------------------------------

var gatewayPrivateNetworkCmd = &cobra.Command{
	Use:   "private-network",
	Short: "PRIVATE-mode gateway private network",
}

var gatewayPrivateNetworkRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Private-network node routes (CIDRs)",
}

var gatewayPrivateNetworkRoutesGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Show a PRIVATE-mode gateway's private-network routes",
	Long: `Show the CIDR routes programmed on a PRIVATE-mode gateway's worker nodes
(GET .../private-network/routes). A PUBLIC-mode gateway 404s with
private_network_not_applicable, which is surfaced as-is.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.GetPrivateRoutes(ctx, args[0])
		if err != nil {
			return err
		}
		if output.GetFormat() == output.FormatJSON {
			return output.JSON(resp)
		}
		rows := make([][]string, 0, len(resp.Routes))
		for i := range resp.Routes {
			rows = append(rows, []string{resp.Routes[i]})
		}
		if len(rows) == 0 {
			fmt.Fprintln(os.Stderr, "No routes configured.")
			return nil
		}
		output.Table([]string{"CIDR"}, rows)
		return nil
	},
}

var gatewayPrivateNetworkRoutesSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Replace a PRIVATE-mode gateway's private-network routes",
	Long: `Replace (PUT, full replacement) the CIDR routes on a PRIVATE-mode gateway's
worker nodes. Pass --route repeatedly, or --file with a {routes: [...]} JSON/YAML
document. --if-match is the optional ETag (from a prior get) for optimistic
concurrency; omit to force the replace.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		f := cmd.Flags()
		req := &gatewaypkg.ReplacePrivateRoutesRequest{}
		if file, _ := f.GetString("file"); file != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read --file: %w", err)
			}
			m, err := yamlToMap(data)
			if err != nil {
				return err
			}
			jb, err := json.Marshal(m)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(jb, req); err != nil {
				return fmt.Errorf("invalid routes spec: %w", err)
			}
		} else {
			req.Routes, _ = f.GetStringArray("route")
		}
		ifMatch, _ := f.GetString("if-match")
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		resp, err := client.ReplacePrivateRoutes(ctx, args[0], ifMatch, req)
		if err != nil {
			return err
		}
		if output.GetFormat() == output.FormatJSON {
			return output.JSON(resp)
		}
		output.Successf("Replaced %d route(s) for gateway %q.", len(resp.Routes), args[0])
		return nil
	},
}

// service-account / repair --------------------------------------------------

var gatewayServiceAccountCmd = &cobra.Command{
	Use:   "service-account",
	Short: "Gateway IAM service account",
}

var gatewayServiceAccountRepairCmd = &cobra.Command{
	Use:   "repair <name>",
	Short: "Repair a gateway's IAM service account",
	Long: `Trigger an IAM service-account repair for a gateway (POST
.../service-account/repair). Use when iam.lastAuthFailureAt is set — the gateway
could not exchange for a token and needs its service account re-issued. Returns
the refreshed gateway.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := newGatewayClient(ctx, cmd)
		if err != nil {
			return err
		}
		gw, err := client.RepairServiceAccount(ctx, args[0])
		if err != nil {
			return err
		}
		return output.PrintResource(gw, func() string { return gw.Name }, func() error { return renderGatewayDetail(gw) })
	},
}
