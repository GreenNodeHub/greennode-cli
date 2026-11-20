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
│   ├── vks/             # VKS product CLI (one file per command + vks.go parent)
│   ├── vserver/         # vServer product CLI (one package per noun + vserver.go parent)
│   └── vlb/             # VLB product CLI (one package per noun + vlb.go parent)
└── internal/
    ├── cli/             # SHARED infrastructure used by every product
    │   ├── client.go        # NewClient / NewClientWithConfig / ProjectID — per-service HTTP client
    │   ├── output.go        # Output(cmd, data) — JSON/table/text rendering
    │   ├── parse.go         # ParseCommaSeparated, BuildEventsQuery, ...
    │   ├── registry.go      # RegisterService / Services — product self-registration
    │   ├── waiter.go        # RunWaiter — shared polling loop for `wait` commands
    │   └── completion.go    # Value-completion framework + resource registry
    ├── resources/vserver/   # Cross-service completion providers (platform-owned)
    ├── resources/vlb/       # Cross-service completion providers for VLB resources
    ├── vserverclient/       # vServer endpoint + resource completers (wraps internal/cli)
    ├── vlbclient/           # VLB endpoint, URL paths + resource completers (wraps internal/cli)
    ├── config/          # Config + credentials (INI), region/endpoint resolution
    ├── auth/            # OAuth2 client-credentials token manager
    ├── client/          # Low-level HTTP client (retry, 401 refresh, typed APIError)
    ├── formatter/       # JSON/table/text + JMESPath
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
       "vlb_endpoint":     "https://hcm-3.api.vngcloud.vn/vserver/vlb-gateway",
   },
   ```

`root.go` iterates `cli.Services()` and never needs editing per service.

### Gating a service out of the public release

A service that is not ready to ship goes in its own `cmd/register_<service>.go`
behind `//go:build !vks_only` instead of in `cmd/register.go`. It is then
compiled (and tested) in dev and CI builds, but excluded from the release
binary, which `release.yml` builds with `-tags vks_only`. `vserver` and `vlb`
are both gated this way today; drop the tag when the service ships.

### Waiters

Services whose operations are asynchronous expose `wait` commands. The polling
loop, the stderr progress line and the exit-255-on-failure convention (matching
the AWS CLI, so a script can tell a failed wait from an ordinary error) live in
`internal/cli/waiter.go`:

```go
func RunWaiter(label, successMsg string, describe func() (interface{}, error),
    eval WaitEvaluator, delay, maxAttempts int) error
```

A product supplies only `describe()` and a `WaitEvaluator` deciding, per poll,
whether the wait is done, has failed, or hit a fatal error. Keep the evaluator a
pure function — that is where the service-specific status handling lives, and it
is what the tests exercise (`cmd/vks/wait_test.go`,
`cmd/vlb/loadbalancer/wait_test.go`). Services differ more than you would
expect: vks reads `status` off a bare object, vlb reads `displayStatus` out of a
`data` envelope.

### Project-scoped services

`vserver` and `vlb` put the project ID in every URL path, so their commands use
`cli.NewClientWithConfig(cmd, "<service>")`, which returns the resolved config
alongside the client, and `cli.ProjectID(cfg)` to read and validate it. The
per-service `internal/<service>client` packages wrap those with the service name
baked in (`vlbclient.Resolve` returns client, config and project ID in one call)
and hold the URL-path builders, so every ID interpolated into a path goes
through one auditable place. VKS is not project-scoped and uses plain
`cli.NewClient`.

## Self-contained subcommand: `agentbase`

The agentbase subcommand package is a self-contained service stack that is
compiled into the default `grn` binary and all release builds. Unlike
`vks`/`vserver`, it does not reuse the shared `internal/cli` infrastructure: it
ships its own v2 OAuth2 client-credentials auth, its own `./.greennode.json`
config loader, and its own HTTP client + output helpers. `AgentbaseCmd`
self-registers via
`cli.RegisterService(...)` in `init()`, so it appears under `grn` with no
further wiring. Its public command-reference pages live under
`docs/commands/agentbase/*` and are published in the `mkdocs.yml` nav.

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
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
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
