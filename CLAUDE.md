# CLAUDE.md — GreenNode CLI

## Project overview

GreenNode CLI (`grn`) is a unified command-line tool for managing GreenNode services. Written in Go, distributed as a single binary. VKS (GreenNode Kubernetes Service) is the first service; other product teams add their own services.

- **Repo**: `GreenNodeHub/greennode-cli`
- **Docs**: https://greennodehub.github.io/greennode-cli/
- **Language**: Go (using cobra CLI framework)
- **Binary**: Single file, zero runtime dependencies

## Project structure

```
go/
├── main.go                          # Entry point
├── cmd/
│   ├── root.go                      # Root command + global flags + --version
│   ├── register.go                  # Blank-imports all product packages (triggers init())
│   ├── agentbase/                   # grn agentbase (default-on; -tags agentbase gate dropped at GA)
│   │   ├── agentbase.go             # AgentbaseCmd subcommand root (self-registers)
│   │   ├── identity.go              # identity group (login/workload/outbound-auth)
│   │   ├── context.go               # context group (switch/current/headers/decorators)
│   │   └── helpers.go               # mustLoadConfig / newAuthProvider
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
│   ├── agentbase/                   # self-contained agentbase stack (own auth/config/client)
│   │   ├── auth/                    # OAuth2 v2 clientcredentials
│   │   ├── client/                  # bearer-token HTTP client
│   │   ├── config/                  # ./.greennode.json loader
│   │   ├── identity/                # identity API client + models
│   │   ├── cliinput/                # interactive prompts
│   │   ├── jsonslice/               # typed JSON slice helper
│   │   └── output/                  # table/json/id + color + banner
│   └── validator/validator.go       # ID format validation
├── go.mod, go.sum

scripts/
├── install.sh           # one-liner installer (macOS / Linux)
├── install.ps1          # one-liner installer (Windows PowerShell)
├── install.cmd          # one-liner installer (Windows pure-CMD, PowerShell-less)
├── uninstall.sh         # one-liner uninstaller (macOS / Linux)
├── uninstall.ps1        # one-liner uninstaller (Windows PowerShell)
├── uninstall.cmd        # one-liner uninstaller (Windows pure-CMD)
└── new-product          # scaffold a new product command
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
3. Blank-import the package in `cmd/register.go`: `_ "github.com/greennodehub/greennode-cli/cmd/<service>"`
4. Build clients with `cli.NewClient(cmd, "<service>")`; print with `cli.Output(cmd, data)`
5. Add `<service>_endpoint` for each region in `internal/config/config.go` REGIONS
6. root.go needs no change — it mounts everything in the registry

Note: `cmd/agentbase` is gated behind the opt-in `agentbase` build tag
(`cmd/register_agentbase.go`), the inverse of the `!vks_only` pattern — it is
excluded from default and release builds while still in development.

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

# Build (agentbase is in the default binary now — no special build tag needed):
CGO_ENABLED=0 go build -o grn .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o grn-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o grn-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o grn-windows-amd64.exe .
```

## Installation one-liners

- **macOS / Linux:** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.sh | bash`
- **Windows (PowerShell):** `irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.ps1 | iex`
- **Windows (CMD, no PowerShell):** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.cmd -o install.cmd && install.cmd && del install.cmd`

Installers resolve the latest release tag via GitHub's `/releases/latest` 302, download the versioned binary (`grn-<plat>-v<ver>`) + a GNU-format `SHA256SUMS`, verify SHA-256, and install user-locally (`~/.local/bin/grn` / `%LOCALAPPDATA%\greennode\bin\grn.exe`, no sudo). Honors `GRN_INSTALL_BASE_URL` (self-host/mirror) and `GRN_INSTALL_TAG` (pin a version; used by tests).

### Uninstall one-liners

- **macOS / Linux:** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.sh | bash`
- **Windows (PowerShell):** `irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.ps1 | iex`
- **Windows (CMD, no PowerShell):** `curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.cmd -o uninstall.cmd && uninstall.cmd && del uninstall.cmd`

Removes the binary + PATH entry the installer added. Pass `--purge` (`-Purge` in PowerShell) to also remove config + credentials (`~/.greennode/` / `%USERPROFILE%\.greennode\`).

## Release artifacts

Each release (tag `vX.Y.Z`) attaches:
- 6 platform binaries: `grn-{linux,linux,darwin,darwin,windows,windows}-{amd64,arm64,amd64,arm64,amd64,arm64}-v<ver>[.exe]`
- `SHA256SUMS`: GNU `sha256sum` format (one line per binary), generated by `release.yml`'s `Generate SHA256SUMS` step.

## Git workflow

- **Do not auto commit/push** — only change source code, user will ask for commit/push
- **Main branch is protected** — must use PR
- **Versioning/CHANGELOG are automated by release-please** — never edit the version
  (`go/cmd/root.go`) or `CHANGELOG.md` by hand
- **PR titles are Conventional Commits** (`feat:`/`fix:`/`feat!:`) — PRs are squash-merged,
  so the title becomes the release commit release-please reads
- **Release**: merge the `chore: release main` PR → tags `vX.Y.Z` + GitHub Release + binaries
- **Release binaries are version-named** (`grn-<plat>-vX.Y.Z[.exe]`) and accompanied by a `SHA256SUMS` file (GNU format) — the `scripts/install.*` installers resolve the latest tag, download the versioned binary + checksums, and verify SHA-256 before installing.
- **Pre-release smoke checklist:** see `RELEASING.md` (run the one-liners against the real tagged release before announcing).

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
| `scripts/install.sh` | One-liner installer: macOS/Linux (`curl \| bash`) |
| `scripts/install.ps1` | One-liner installer: Windows PowerShell (`irm \| iex`) |
| `scripts/install.cmd` | One-liner installer: Windows pure-CMD fallback |
| `scripts/uninstall.sh` | One-liner uninstaller: macOS/Linux |
| `scripts/uninstall.ps1` | One-liner uninstaller: Windows PowerShell |
| `scripts/uninstall.cmd` | One-liner uninstaller: Windows pure-CMD fallback |
