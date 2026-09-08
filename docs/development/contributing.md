# Contributing

See [CONTRIBUTING.md](https://github.com/GreenNodeHub/greennode-cli/blob/main/CONTRIBUTING.md)
for the full contributing guide, and [Architecture & Adding a Service](architecture.md)
for how the codebase is structured and how to add a new product CLI.

## Quick start

```bash
git clone https://github.com/GreenNodeHub/greennode-cli.git
cd greennode-cli/go
go build -o grn .
./grn --version
```

## Adding a new command

1. Create `cmd/<service>/<command_name>.go`
2. Define a `cobra.Command` with `Use`, `Short`, `RunE`
3. Register it on the service's parent command in `cmd/<service>/<service>.go`
4. Build the client with `cli.NewClient(cmd, "<service>")`; print with `cli.Output(cmd, data)`
5. Validate any ID used in a URL with `validator.ValidateID(...)`
6. Add `--dry-run` for create/update/delete; `--force` + confirmation for delete
7. (Optional) Register flag value completion — see [Architecture](architecture.md#shell-completion)
8. Document it: create `docs/commands/<service>/<command-name>.md`, add it to the
   command index table and to the `mkdocs.yml` nav

The changelog/version are automated by release-please from your Conventional
Commit PR title — no manual changelog step.

## Adding a new service

A new product self-registers and is mounted without editing `root.go`. See
[Architecture & Adding a Service](architecture.md#adding-a-new-service) for the
full steps (parent command + `cli.RegisterService`, blank-import in
`cmd/register.go`, endpoint in `REGIONS`, CODEOWNERS entry).

## Running tests

```bash
cd go
go test ./...
```

> On macOS 26 (Darwin 25) with Go 1.22, `go test` may abort with
> `dyld: missing LC_UUID`. Use the external linker:
> `CGO_ENABLED=1 go test -ldflags='-linkmode=external' ./...`

### Live end-to-end runbooks

`go test ./...` cannot reach a command's `RunE` network path (resolve → request → output):
`cli.NewTokenProvider` only targets the prod/dev IAM presets, so a command cannot be pointed at a
local server without real credentials. Services still behind a build tag therefore carry a manual
runbook that is run against a real project before the tag comes off:

| Service | Runbook | Gate |
|---|---|---|
| VLB | `tests/vlb/RUNBOOK.md` | run in full before removing `-tags vks_only`; see `RELEASING.md` |

These runbooks create real, billable resources and take hours — do not run one casually. Everything
they create is prefixed and tagged so teardown can verify nothing leaked. Copy
`RESULTS.template.md` to `RESULTS.md` (gitignored) and attach it to the PR that removes the tag.

The installer and release-artifact harnesses in `tests/` (`install.test.sh`, `release.test.sh`, and
their PowerShell/CMD counterparts) are hermetic and run in CI.
