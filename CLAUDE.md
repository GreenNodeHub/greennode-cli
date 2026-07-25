# CLAUDE.md — GreenNode CLI

## Project overview

GreenNode CLI (`grn`) is a unified command-line tool for managing GreenNode services. Written in Go, distributed as a single binary. VKS (GreenNode Kubernetes Service) is the first service; other product teams add their own services.

- **Repo**: `vngcloud/greennode-cli`
- **Docs**: https://vngcloud.github.io/greennode-cli/
- **Language**: Go (using cobra CLI framework)
- **Binary**: Single file, zero runtime dependencies

## Project structure

```
go/
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root command + global flags + --version
│   ├── register.go                  # Blank-imports all product packages (triggers init())
│   ├── configure/
│   │   ├── configure.go             # Interactive setup
│   │   ├── list.go                  # grn configure list
│   │   ├── get.go                   # grn configure get
│   │   └── set.go                   # grn configure set
│   └── vks/
│       ├── vks.go                   # VKS parent command
│       ├── helpers.go               # Shared utilities (client, output, parsing)
│       ├── list_clusters.go         # Auto-pagination
│       ├── get_cluster.go
│       ├── create_cluster.go        # --dry-run validation
│       ├── update_cluster.go
│       ├── delete_cluster.go        # Confirm + --force + --dry-run
│       ├── list_nodegroups.go
│       ├── get_nodegroup.go
│       ├── create_nodegroup.go
│       ├── update_nodegroup.go
│       ├── update_nodegroup_metadata.go  # Update labels/tags/taints
│       ├── upgrade_nodegroup_version.go  # Upgrade node group k8s version
│       ├── list_nodes.go                 # List nodes in a node group
│       ├── delete_nodegroup.go
│       ├── list_cluster_versions.go      # Available k8s versions
│       ├── config_auto_healing.go        # Configure auto-healing
│       ├── get_cluster_events.go         # Cluster events (paginated)
│       ├── get_nodegroup_events.go       # Node group events (paginated)
│       ├── generate_kubeconfig.go        # Request kubeconfig (async)
│       ├── update_kubeconfig.go          # Fetch + merge kubeconfig
│       ├── completion.go            # registerCompletions() — wires all VKS flag value completers
│       ├── wait.go                  # Polling waiters (cluster-active, cluster-deleted, nodegroup-active, nodegroup-deleted)
│       ├── wait_test.go             # Wait command tests
│       └── auto_upgrade.go          # Set/delete auto-upgrade
├── internal/
│   ├── cli/
│   │   ├── client.go                # NewClient(cmd, serviceName) — shared HTTP client factory
│   │   ├── completion.go            # FlagValues/FlagValuesFrom/FlagFromAPI/ExtractIDs + RegisterResourceCompleter/ResourceCompletion
│   │   ├── output.go                # Output(cmd, data) — unified JSON/table/text printing
│   │   ├── parse.go                 # ParseCommaSeparated, BuildEventsQuery helpers
│   │   └── registry.go             # RegisterService / Services — product self-registration
│   ├── config/
│   │   ├── config.go               # Config + credentials loading (INI)
│   │   └── writer.go               # ConfigFileWriter (0600 perms)
│   ├── auth/token.go                # OAuth2 Client Credentials (IAM)
│   ├── client/client.go             # HTTP client with retry + auto-refresh
│   ├── formatter/formatter.go       # JSON/Table/Text + JMESPath
│   ├── kubeconfig/
│   │   └── kubeconfig.go            # Merge kubeconfig into ~/.kube/config
│   ├── resources/
│   │   └── vserver/
│   │       └── vserver.go           # vserver resource completers (vpc/subnet/ssh-key/security-group/disk-type)
│   └── validator/validator.go       # ID format validation
├── go.mod, go.sum
```

## Code conventions

- All source code text in **English**
- Use cobra for all commands
- Internal packages in `internal/` (not importable externally)
- Commands in `cmd/` following cobra patterns
- Use `cobra.Command` with `RunE` for error handling

## GreenNode API quirks

- **IAM API uses camelCase**: `grantType`, `accessToken`, `expiresIn`
- **VKS API pagination is 0-based**: page 0 = first page
- **`--version` conflict**: Use `--k8s-version` for Kubernetes version

## Adding a new command

1. Create file in `cmd/vks/<command_name>.go`
2. Define `cobra.Command` with Use, Short, RunE
3. Add flags via `cmd.Flags()`
4. Register in `cmd/vks/vks.go` init(): `VksCmd.AddCommand(newCmd)`
5. Add `validator.ValidateID()` for any ID args in URLs
6. Add `--dry-run` for create/update/delete commands
7. Add `--force` + confirmation for delete commands

## Shell completion

Static command/flag completion is automatic (cobra: `grn completion <shell>`). For flag VALUE completion:

- Enum: `cmd.RegisterFlagCompletionFunc(name, cli.FlagValues("a","b"))`
- Config-derived: `cli.FlagValuesFrom(fn func() []string)`
- API-backed: `cli.FlagFromAPI(func(ctx, cmd) ([]string,error))` — bounded timeout, fails silently
- Cross-service resource: consumer uses `cli.ResourceCompletion("<svc>:<resource>")`; the owning service registers a provider via `cli.RegisterResourceCompleter("<svc>:<resource>", ...)` (e.g. `internal/resources/vserver/`, blank-imported in `cmd/register.go`)

VKS wires its flags centrally in `cmd/vks/completion.go` `registerCompletions()`.

## Adding a new service

1. Create `cmd/<service>/` with a parent `cobra.Command` (e.g. `VserverCmd`)
2. In the package `init()`, register it: `cli.RegisterService(VserverCmd)`
3. Blank-import the package in `cmd/register.go`: `_ "github.com/vngcloud/greennode-cli/cmd/<service>"`
4. Build clients with `cli.NewClient(cmd, "<service>")`; print with `cli.Output(cmd, data)`
5. Add `<service>_endpoint` for each region in `internal/config/config.go` REGIONS
6. root.go needs no change — it mounts everything in the registry

## Security rules

- **Credential masking**: `configure list` and `configure get` mask client_id/client_secret (last 4 chars only)
- **Credential env vars supported**: `GRN_ACCESS_KEY_ID`/`GRN_SECRET_ACCESS_KEY` override credentials file (highest priority)
- **Input validation**: All cluster-id/nodegroup-id validated via `validator.ValidateID()` before URLs
- **SSL default on**: `--no-verify-ssl` prints warning to stderr
- **Tokens in memory only**: Never written to disk or logged
- **File permissions**: Credentials file created with 0600, directory 0700

## Building

```bash
cd go
CGO_ENABLED=0 go build -o grn .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o grn-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o grn-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o grn-windows-amd64.exe .
```

## Git workflow

- **Do not auto commit/push** — only change source code, user will ask for commit/push
- **Main branch is protected** — must use PR
- **Versioning/CHANGELOG are automated by release-please** — never edit the version
  (`go/cmd/root.go`) or `CHANGELOG.md` by hand
- **PR titles are Conventional Commits** (`feat:`/`fix:`/`feat!:`) — PRs are squash-merged,
  so the title becomes the release commit release-please reads
- **Release**: merge the `chore: release main` PR → tags `vX.Y.Z` + GitHub Release + binaries

## Documentation update rule

**After ANY change to business logic, security, configuration, or commands:**

1. Review ALL docs below and update what's affected
2. If unsure whether a doc needs updating, read it and check

**Docs to check:**

- `docs/commands/vks/` (GitHub Pages) — add/update command reference page, check `index.md` table
- `mkdocs.yml` — add nav entry for any new command page
- `README.md`
- `CLAUDE.md`
- `CONTRIBUTING.md`
- `docs/development/contributing.md` and `docs/development/architecture.md` (adding services/commands)

> Note: `docs/superpowers/` (design specs/plans) is local-only and git-ignored —
> never commit it to this public repo.

**Examples:**
- Added a command → create `docs/commands/vks/<command>.md` + add to `docs/commands/vks/index.md` table + add to `mkdocs.yml` nav
- Removed a command → delete doc page + remove from `index.md` + remove from `mkdocs.yml`
- Changed flags or output → update the command's doc page
- Changed auth/credentials → update README config section + CLAUDE.md security rules
- Changed project structure → update README structure + CLAUDE.md repository structure

Code without docs is not done.

## Key files

| File | Purpose |
|------|---------|
| `cmd/root.go` | Root command, global flags, --version |
| `cmd/vks/helpers.go` | Client creation, output formatting, label/taint parsing |
| `internal/config/config.go` | Config loading from ~/.greennode/, REGIONS map |
| `internal/config/writer.go` | INI file writer with 0600 perms |
| `internal/auth/token.go` | TokenManager — OAuth2 with IAM (camelCase) |
| `internal/client/client.go` | HTTP client with retry (3x backoff) + 401 refresh |
| `internal/formatter/formatter.go` | JSON/Table/Text + JMESPath |
| `internal/validator/validator.go` | ID format validation |
