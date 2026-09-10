# Architecture & Adding a Service

GreenNode CLI (`grn`) is a single Go binary built with [cobra](https://github.com/spf13/cobra).
It is designed so multiple product teams can add their own service CLI in the
same repo without conflicting with each other.

## Layout

```
go/
├── cmd/
│   ├── root.go          # Root command, global flags, mounts services from the registry
│   ├── register.go      # Blank-imports each product package (the one file to touch per service)
│   ├── configure/       # `grn configure` (built-in, credentials)
│   └── vks/             # VKS product CLI (one file per command + vks.go parent)
└── internal/
    ├── cli/             # SHARED infrastructure used by every product
    │   ├── client.go        # NewClient(cmd, serviceName) — per-service HTTP client
    │   ├── output.go        # Output(cmd, data) — JSON/table/text rendering
    │   ├── parse.go         # ParseCommaSeparated, BuildEventsQuery, ...
    │   ├── registry.go      # RegisterService / Services — product self-registration
    │   └── completion.go    # Value-completion framework + resource registry
    ├── resources/vserver/   # Cross-service completion providers (platform-owned)
    ├── config/          # Config + credentials (INI), region/endpoint resolution
    ├── auth/            # Profile-selected machine and user token providers
    ├── client/          # Low-level HTTP client (retry, 401 refresh, typed APIError)
    ├── formatter/       # JSON/table/text + JMESPath
    ├── operation/       # Descriptor-driven command parsing and execution
    ├── redact/          # Shared JSON, URL, header and path-value masking
    └── validator/       # ID validation
```

**Rule of thumb:** product-specific code lives in a per-service command package; anything shared
goes in `internal/cli` (or another `internal/*` package). A product package must
not import another product package.

## Adding a new service

A new product (e.g. `vserver`) is mounted without touching `root.go`:

1. **Create the service's command package** with a parent `cobra.Command`:

   ```go
   package vserver

   import (
       "…/internal/cli"  // shared CLI infra (NewClient, Output, RegisterService)
       "github.com/spf13/cobra"
   )

   var VserverCmd = &cobra.Command{
       Use:   "vserver",
       Short: "VNG Cloud vServer commands",
       Run:   func(cmd *cobra.Command, args []string) { cmd.Help() },
   }

   func init() {
       // ... VserverCmd.AddCommand(...) for each subcommand
       cli.RegisterService(VserverCmd) // self-register; root mounts it
   }
   ```

2. **Blank-import the package in `cmd/register.go`** — the only shared file you edit:

   ```go
   import (
       _ "…/cmd/vks"
       _ "…/resources/vserver"  // cross-service completion providers
       _ "…/cmd/vserver"        // add this line
   )
   ```

3. **Declare the service endpoint** per region in `internal/config/config.go` `REGIONS`:

   ```go
   "HCM-3": {
       "vks_endpoint":     "https://vks.api.vngcloud.vn",
       "vserver_endpoint": "https://hcm-3.api.vngcloud.vn/vserver/vserver-gateway",
   },
   ```

`root.go` iterates `cli.Services()` and never needs editing per service.

## Self-contained subcommand: `agentbase`

AgentBase is compiled into the default binary and release builds. It shares INI profiles and `cli.NewTokenProvider` with VKS/vServer, including `auth_mode`, `iam_env`, refresh-token rotation, and `agent_identity`. Its product HTTP client and table/JSON/ID output helpers remain separate; only safe reads may refresh and retry after 401, while writes and credential/provisioning reads do not replay. It self-registers with `cli.RegisterService`. Its command reference lives under `docs/commands/agentbase/`.

## Shared foundation

`cli.NewClient` resolves regional endpoints. `cli.NewClientWithEndpoint` handles a documented global endpoint; `cli.BuildClient` supplies a resolver, optional region override, and a post-build validation/header hook. All use the existing profile-selected machine/user provider. Construction performs no token or API request. Command context and read timeout are applied to both IAM and API requests; vServer keeps its public builder and project-ID contract.

`internal/client` preserves `GetAllPages` and the existing decoded-response helpers. Status-aware methods expose status and empty-body metadata; raw, byte, and stream methods retain the service's response representation. GET/HEAD/OPTIONS may retry transient failures up to three times and refresh once after 401. Writes, explicit no-retry calls (including state-changing GETs), and streams are never replayed automatically, including after 401. An uncertain write requires independent state verification, not a blind retry.

Shared API requests reject redirects and non-2xx responses. This prevents redirected write replay and credential forwarding beyond the validated endpoint.

IAM token requests also reject redirects and expose neither provider response bodies nor token metadata in diagnostics. Machine authentication sends its existing Basic client credentials once per grant; user refresh and PKCE exchange retain their existing Basic contract without retries.

`internal/redact` masks credential-shaped JSON fields, sensitive query/header names, and explicitly declared secret path values without modifying transmitted data. Sensitive response methods also suppress displayed error bodies and debug responses. `APIError.Body` deliberately retains raw data for product-specific handling: never log it directly. Key-based masking cannot discover arbitrary secrets inside free text; unknown binary bodies should be logged as metadata only.

`internal/operation` owns descriptor parsing and the shared execution order: validate path/query/body → offline validation → dry-run early return → live validation → confirmation → client/request → response validation → output. Services own endpoint facts, body/schema contracts, status acceptance, and specialized hooks. Dry-run must not construct clients, read credentials, call the API, prompt, or write output files. The root translates legacy bool-based confirmation refusals into failure exits; new engine commands return the refusal directly.

Use the [public contract-fixture convention](contract-fixtures.md) for independently sourced service tests. Shared engine tests alone do not establish provider compatibility.

vBackup, vCR, vDB, vLB, vMonitor, vMonitor Log, vStorage, and vStorage Gateway use the descriptor engine. SaaS AI uses shared transport with custom multipart and binary handling. All nine register in default and release builds; service tests retain independent public contract evidence.

IAM management uses `internal/iamclient` for its global Accounts and Policies endpoints, with shared profile authentication. Its management commands do not replace `login`/`logout` or add token-issuance and impersonation flows. Protected credential body files, current-identity checks, confirmation, no-retry requests, and operation-specific response validation remain product-owned.

## Writing a command

Follow the existing `cmd/vks/*.go` files. Each command:

- builds its client with `cli.NewClient(cmd, "<service>")` (resolves the
  `<service>_endpoint` for the active region),
- prints results with `cli.Output(cmd, data)` (honours `--output` / `--query`),
- validates any ID used in a URL with `validator.ValidateID(...)`,
- adds `--dry-run` for create/update/delete and `--force` + confirmation for delete.

```go
func runGetThing(cmd *cobra.Command, args []string) error {
    id, _ := cmd.Flags().GetString("id")
    if err := validator.ValidateID(id, "id"); err != nil {
        return err
    }
    c, err := cli.NewClient(cmd, "vserver")
    if err != nil {
        return err
    }
    res, err := c.Get(fmt.Sprintf("/v2/%s/things/%s", projectID, id), nil)
    if err != nil {
        return err
    }
    return cli.Output(cmd, res)
}
```

## Shell completion

Static command/flag completion is automatic (`grn completion <shell>`). For flag
**value** completion:

- Enum: `cmd.RegisterFlagCompletionFunc(name, cli.FlagValues("a", "b"))`
- Config-derived: `cli.FlagValuesFrom(fn func() []string)`
- API-backed: `cli.FlagFromAPI(func(ctx, cmd) ([]string, error))` — bounded timeout,
  fails silently, prefix-filtered. Use `cli.ExtractIDs(resp, "id", "uuid")` to pull
  IDs out of a list response.
- Cross-service resource: a consumer uses `cli.ResourceCompletion("<svc>:<resource>")`;
  the owning service registers the provider with
  `cli.RegisterResourceCompleter("<svc>:<resource>", ...)`. See
  the vserver resource-completer package for the pattern.

## Ownership

Review ownership is routed by path: platform-owned code (shared infrastructure,
the root command, and `configure`) is distinguished from product-team-owned code
(a service's command package). Add an ownership entry for a new service directory.

## Running tests

```bash
cd go
go test ./...
```

> On macOS 26 (Darwin 25) with Go 1.22, `go test` may abort with
> `dyld: missing LC_UUID`. Use the external linker:
> `CGO_ENABLED=1 go test -ldflags='-linkmode=external' ./...`
